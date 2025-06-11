package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"net/http"

	"github.com/tibuski/goAirfocus/airfocus"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server represents the HTTP server for the Airfocus API Tools application
type Server struct {
	templates *template.Template
}

// NewServer creates and initializes a new Server instance
func NewServer() (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		templates: tmpl,
	}, nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// WorkspaceIDResponse represents the API response for workspace ID lookup
type WorkspaceIDResponse struct {
	Status string `json:"status"`          // Status of the request ("success" or "error")
	ID     string `json:"id,omitempty"`    // The workspace ID if found
	Alias  string `json:"alias,omitempty"` // The workspace alias if found
	Error  string `json:"error,omitempty"` // Error message if the request failed
}

// handleGetWorkspaceID handles POST requests to get a workspace ID by name
func (s *Server) handleGetWorkspaceID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(WorkspaceIDResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceName := r.FormValue("workspace_name")

	if apiKey == "" || workspaceName == "" {
		json.NewEncoder(w).Encode(WorkspaceIDResponse{
			Status: "error",
			Error:  "API key and workspace name are required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)
	result, err := client.GetWorkspaceIDByName(r.Context(), workspaceName)

	response := WorkspaceIDResponse{}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else {
		response.Status = "success"
		response.ID = result.ID
		response.Alias = result.Alias
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		// Note: Can't send http.Error after writing to header/body,
		// but logging is still important.
	}
}

// WorkspaceUsersResponse defines the structure for the API response for workspace users
type WorkspaceUsersResponse struct {
	Status string             `json:"status"`          // Status of the request ("success" or "error")
	Data   WorkspaceUsersData `json:"data,omitempty"`  // Contains user statistics and list of users
	Error  string             `json:"error,omitempty"` // Error message if the request failed
}

// WorkspaceUsersData combines user statistics and the list of users for a workspace
type WorkspaceUsersData struct {
	airfocus.WorkspaceUserStats                          // Embedded user statistics
	Users                       []airfocus.WorkspaceUser `json:"users,omitempty"` // List of users with their permissions
}

// handleGetWorkspaceUsers handles POST requests to get user statistics and list of users for a workspace
func (s *Server) handleGetWorkspaceUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(WorkspaceUsersResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceID := r.FormValue("workspace_id")     // Can be empty
	workspaceName := r.FormValue("workspace_name") // Can be empty

	if apiKey == "" {
		json.NewEncoder(w).Encode(WorkspaceUsersResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)

	// If workspaceID is not provided, try to resolve it from workspaceName
	if workspaceID == "" && workspaceName != "" {
		result, err := client.GetWorkspaceIDByName(r.Context(), workspaceName)
		if err != nil {
			json.NewEncoder(w).Encode(WorkspaceUsersResponse{
				Status: "error",
				Error:  fmt.Sprintf("Failed to resolve workspace ID from name '%s': %v", workspaceName, err),
			})
			return
		}
		workspaceID = result.ID
	}

	// If after all attempts, workspaceID is still empty, return an error
	if workspaceID == "" {
		json.NewEncoder(w).Encode(WorkspaceUsersResponse{
			Status: "error",
			Error:  "Workspace ID or name is required",
		})
		return
	}

	// Fetch both statistics and the list of users
	stats, err := client.GetWorkspaceUserStats(r.Context(), workspaceID)
	if err != nil {
		json.NewEncoder(w).Encode(WorkspaceUsersResponse{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	users, err := client.GetWorkspaceUsers(r.Context(), workspaceID)
	if err != nil {
		json.NewEncoder(w).Encode(WorkspaceUsersResponse{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	response := WorkspaceUsersResponse{}
	response.Status = "success"
	response.Data = WorkspaceUsersData{
		WorkspaceUserStats: stats,
		Users:              users,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// FieldIDResponse represents the API response for field ID lookup
type FieldIDResponse struct {
	Status string `json:"status"`          // Status of the request ("success" or "error")
	ID     string `json:"id,omitempty"`    // The field ID if found
	Error  string `json:"error,omitempty"` // Error message if the request failed
	Field  *struct {
		Name           string   `json:"name"`                     // Field name
		Description    string   `json:"description"`              // Field description
		Type           string   `json:"type"`                     // Field type
		IsTeamField    bool     `json:"isTeamField"`              // Whether this is a team-wide field
		WorkspaceNames []string `json:"workspaceNames,omitempty"` // List of workspace names where this field is used
	} `json:"field,omitempty"`
}

// handleGetFieldID handles POST requests to get a field ID by name
func (s *Server) handleGetFieldID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(FieldIDResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		json.NewEncoder(w).Encode(FieldIDResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	fieldName := r.FormValue("field_name")
	if fieldName == "" {
		json.NewEncoder(w).Encode(FieldIDResponse{
			Status: "error",
			Error:  "Field name is required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode(FieldIDResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to list fields: %v", err),
		})
		return
	}

	// Find the field by name (case-insensitive)
	var foundField *airfocus.FieldWithWorkspaceNames
	for i, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			foundField = &fields[i]
			break
		}
	}

	if foundField == nil {
		json.NewEncoder(w).Encode(FieldIDResponse{
			Status: "error",
			Error:  fmt.Sprintf("No field found with name: %s", fieldName),
		})
		return
	}

	response := FieldIDResponse{
		Status: "success",
		ID:     foundField.ID,
		Field: &struct {
			Name           string   `json:"name"`
			Description    string   `json:"description"`
			Type           string   `json:"type"`
			IsTeamField    bool     `json:"isTeamField"`
			WorkspaceNames []string `json:"workspaceNames,omitempty"`
		}{
			Name:           foundField.Name,
			Description:    foundField.Description,
			Type:           foundField.Type,
			IsTeamField:    foundField.IsTeamField,
			WorkspaceNames: foundField.WorkspaceNames,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// FieldAPIResponse represents a field in the API response
type FieldAPIResponse struct {
	ID             string   `json:"id"`                       // Unique identifier for the field
	Name           string   `json:"name"`                     // Field name
	Description    string   `json:"description"`              // Field description
	Type           string   `json:"type"`                     // Field type
	CreatedAt      string   `json:"createdAt"`                // Creation timestamp
	UpdatedAt      string   `json:"updatedAt"`                // Last update timestamp
	IsTeamField    bool     `json:"isTeamField"`              // Whether this is a team-wide field
	WorkspaceNames []string `json:"workspaceNames,omitempty"` // List of workspace names where this field is used
}

// FieldListResponse represents the API response for listing fields
type FieldListResponse struct {
	Status string             `json:"status"`          // Status of the request ("success" or "error")
	Data   []FieldAPIResponse `json:"data,omitempty"`  // List of fields
	Error  string             `json:"error,omitempty"` // Error message if the request failed
}

// handleListFields handles POST requests to list all fields
func (s *Server) handleListFields(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleListFields called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		json.NewEncoder(w).Encode(FieldListResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		log.Printf("API key is missing")
		json.NewEncoder(w).Encode(FieldListResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	log.Printf("Creating Airfocus client and calling ListFields")
	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		log.Printf("Error listing fields: %v", err)
		json.NewEncoder(w).Encode(FieldListResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to list fields: %v", err),
		})
		return
	}

	log.Printf("Successfully retrieved %d fields", len(fields))

	// Convert fields to a format suitable for JSON response
	responseFields := make([]FieldAPIResponse, len(fields))
	for i, field := range fields {
		responseFields[i] = FieldAPIResponse{
			ID:             field.ID,
			Name:           field.Name,
			Description:    field.Description,
			Type:           field.Type,
			CreatedAt:      field.CreatedAt,
			UpdatedAt:      field.UpdatedAt,
			IsTeamField:    field.IsTeamField,
			WorkspaceNames: field.WorkspaceNames,
		}
	}

	response := FieldListResponse{
		Status: "success",
		Data:   responseFields,
	}

	log.Printf("Sending response with %d fields", len(responseFields))
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// UsersWithRolesResponse represents the API response for listing users with their roles
type UsersWithRolesResponse struct {
	Status string                  `json:"status"`          // Status of the request ("success" or "error")
	Data   []airfocus.UserWithRole `json:"data,omitempty"`  // List of users with their roles
	Error  string                  `json:"error,omitempty"` // Error message if the request failed
}

// UserWorkspacesResponse represents the API response for listing a user's workspace access
type UserWorkspacesResponse struct {
	Status string                         `json:"status"`          // Status of the request ("success" or "error")
	Data   []airfocus.UserWorkspaceAccess `json:"data,omitempty"`  // List of workspaces the user has access to
	Error  string                         `json:"error,omitempty"` // Error message if the request failed
}

// handleGetUsersWithRoles handles POST requests to get a list of users with their roles
func (s *Server) handleGetUsersWithRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(UsersWithRolesResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		json.NewEncoder(w).Encode(UsersWithRolesResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)
	users, err := client.FormatUsersWithRoles(r.Context())

	response := UsersWithRolesResponse{}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else {
		response.Status = "success"
		response.Data = users
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleGetUserWorkspaces handles requests to get all workspaces a specific user has access to.
func (s *Server) handleGetUserWorkspaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(UserWorkspacesResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	userID := r.FormValue("user_id")

	if apiKey == "" || userID == "" {
		http.Error(w, "API key and user ID are required", http.StatusBadRequest)
		return
	}

	// Add context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second) // Increased timeout slightly
	defer cancel()

	client := airfocus.NewClient(apiKey)

	// --- ADDED: Refresh the workspace cache before getting user workspaces ---
	if err := client.RefreshCacheIfNeeded(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to refresh workspace cache: %v", err), http.StatusInternalServerError)
		return
	}

	// Use the client method to get user workspaces
	// NOTE: The GetUserWorkspaces client method will now read from the cache.
	userWorkspaces, err := client.GetUserWorkspaces(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user workspaces: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare and send the response
	response := UserWorkspacesResponse{
		Status: "success",
		Data:   userWorkspaces,
	}
	json.NewEncoder(w).Encode(response)
}

// TeamLicenseInfo represents the license information for a team
type TeamLicenseInfo struct {
	TeamID string `json:"teamId"` // Unique identifier for the team
	Slug   string `json:"slug"`   // Team slug
	Name   string `json:"name"`   // Team name
	State  struct {
		Features []string `json:"features"` // List of enabled features
		Seats    struct {
			Admin struct {
				Total int `json:"total"` // Total number of admin seats
				Used  int `json:"used"`  // Number of used admin seats
				Free  int `json:"free"`  // Number of free admin seats
			} `json:"admin"`
			Editor struct {
				Total int `json:"total"` // Total number of editor seats
				Used  int `json:"used"`  // Number of used editor seats
				Free  int `json:"free"`  // Number of free editor seats
			} `json:"editor"`
			Contributor struct {
				Total int `json:"total"` // Total number of contributor seats
				Used  int `json:"used"`  // Number of used contributor seats
				Free  int `json:"free"`  // Number of free contributor seats
			} `json:"contributor"`
			Any struct {
				Total int `json:"total"` // Total number of any-type seats
				Used  int `json:"used"`  // Number of used any-type seats
				Free  int `json:"free"`  // Number of free any-type seats
			} `json:"any"`
		} `json:"seats"`
		Workspaces struct {
			Total int `json:"total"` // Total number of workspaces allowed
		} `json:"workspaces"`
		Subscription struct {
			Type string `json:"type"` // Type of subscription
		} `json:"subscription"`
	} `json:"state"`
	Flags struct {
		EnableAi                  struct{ Value, Enforced, Explicit bool } `json:"enableAi"`                  // AI feature flag
		EnableOkrApp              struct{ Value, Enforced, Explicit bool } `json:"enableOkrApp"`              // OKR app feature flag
		RemoveBranding            struct{ Value, Enforced, Explicit bool } `json:"removeBranding"`            // Branding removal flag
		ForbidShareLinkCreation   struct{ Value, Enforced, Explicit bool } `json:"forbidShareLinkCreation"`   // Share link creation restriction flag
		RestrictShareLinkCreation struct{ Value, Enforced, Explicit bool } `json:"restrictShareLinkCreation"` // Share link creation restriction flag
		RequireShareLinkPassword  struct{ Value, Enforced, Explicit bool } `json:"requireShareLinkPassword"`  // Share link password requirement flag
		RequirePortalLogin        struct{ Value, Enforced, Explicit bool } `json:"requirePortalLogin"`        // Portal login requirement flag
		RequirePortalPassword     struct{ Value, Enforced, Explicit bool } `json:"requirePortalPassword"`     // Portal password requirement flag
	} `json:"flags"`
	CreatedAt string `json:"createdAt"` // Creation timestamp
	UpdatedAt string `json:"updatedAt"` // Last update timestamp
}

// handleGetTeamLicense handles GET requests to retrieve team license information
func handleGetTeamLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Make request to Airfocus API
	req, err := http.NewRequest("GET", "https://api.airfocus.com/api/team", nil)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making request to Airfocus API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var licenseInfo TeamLicenseInfo
	if err := json.NewDecoder(resp.Body).Decode(&licenseInfo); err != nil {
		http.Error(w, "Error decoding response", http.StatusInternalServerError)
		return
	}

	// Set response headers and return the license info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"state": map[string]interface{}{
				"seats": map[string]interface{}{
					"any": licenseInfo.State.Seats.Any,
				},
			},
		},
	})
}

// ContributorsResponse represents the API response for listing contributors
type ContributorsResponse struct {
	Status string                  `json:"status"`          // Status of the request ("success" or "error")
	Data   []airfocus.UserWithRole `json:"data,omitempty"`  // List of contributors
	Error  string                  `json:"error,omitempty"` // Error message if the request failed
}

// handleGetContributors handles POST requests to get a list of contributors
func (s *Server) handleGetContributors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(ContributorsResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		json.NewEncoder(w).Encode(ContributorsResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)
	users, err := client.FormatUsersWithRoles(r.Context())

	response := ContributorsResponse{}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else {
		// Filter only contributors
		var contributors []airfocus.UserWithRole
		for _, user := range users {
			if strings.ToLower(user.Role) == "contributor" {
				contributors = append(contributors, user)
			}
		}
		response.Status = "success"
		response.Data = contributors
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// main is the entry point of the application
func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// API endpoints
	http.HandleFunc("/api/workspace/id", server.handleGetWorkspaceID)
	http.HandleFunc("/api/workspace/users", server.handleGetWorkspaceUsers)
	http.HandleFunc("/api/field/id", server.handleGetFieldID)
	http.HandleFunc("/api/users/roles", server.handleGetUsersWithRoles)     // New endpoint for users with roles
	http.HandleFunc("/api/user/workspaces", server.handleGetUserWorkspaces) // New endpoint for user workspaces

	// Add new endpoint for listing all workspaces
	http.HandleFunc("/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  "Method not allowed",
			})
			return
		}

		apiKey := r.FormValue("api_key")
		if apiKey == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  "API key is required",
			})
			return
		}

		client := airfocus.NewClient(apiKey)
		workspaces, err := client.ListWorkspaces(r.Context())
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		// Transform workspaces to include only relevant fields
		type WorkspaceSummary struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Alias string `json:"alias"`
			// Description removed as fmt.Sprintf("%v", ws.Description.Blocks) is not a good JSON representation
			ItemType     string `json:"itemType"`
			ProgressMode string `json:"progressMode"`
			Archived     bool   `json:"archived"`
		}

		summaries := make([]WorkspaceSummary, len(workspaces))
		for i, ws := range workspaces {
			summaries[i] = WorkspaceSummary{
				ID:           ws.ID,
				Name:         ws.Name,
				Alias:        ws.Alias,
				ItemType:     ws.ItemType,
				ProgressMode: ws.ProgressMode,
				Archived:     ws.Archived,
			}
		}

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   summaries,
		}); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	})

	// Add the new route for listing fields
	http.HandleFunc("/api/fields", server.handleListFields)

	// Add the new endpoint
	http.HandleFunc("/api/team/license", handleGetTeamLicense)

	// Add the new endpoint for listing contributors
	http.HandleFunc("/api/contributors", server.handleGetContributors)

	// Web interface
	http.HandleFunc("/", server.handleIndex)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
