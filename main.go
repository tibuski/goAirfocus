package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strings"

	"net/http"

	"github.com/tibuski/goAirfocus/airfocus"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	templates *template.Template
}

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

type WorkspaceIDResponse struct {
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Error  string `json:"error,omitempty"`
}

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
	Status string             `json:"status"`
	Data   WorkspaceUsersData `json:"data,omitempty"`
	Error  string             `json:"error,omitempty"`
}

// WorkspaceUsersData combines user statistics and the list of users
type WorkspaceUsersData struct {
	airfocus.WorkspaceUserStats
	Users []airfocus.WorkspaceUser `json:"users,omitempty"`
}

// handleGetWorkspaceUsers retrieves user statistics and list of users for a specific workspace
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

type FieldIDResponse struct {
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
	Field  *struct {
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		Type           string   `json:"type"`
		IsTeamField    bool     `json:"isTeamField"`
		WorkspaceNames []string `json:"workspaceNames,omitempty"`
	} `json:"field,omitempty"`
}

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

// FieldAPIResponse represents a single field in the API response for listing fields
type FieldAPIResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Type           string   `json:"type"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	IsTeamField    bool     `json:"isTeamField"`
	WorkspaceNames []string `json:"workspaceNames,omitempty"`
}

// FieldListResponse represents the response for listing fields
type FieldListResponse struct {
	Status string             `json:"status"`
	Data   []FieldAPIResponse `json:"data,omitempty"`
	Error  string             `json:"error,omitempty"`
}

func (s *Server) handleListFields(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(FieldListResponse{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		json.NewEncoder(w).Encode(FieldListResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode(FieldListResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to list fields: %v", err),
		})
		return
	}

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

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// UsersWithRolesResponse defines the structure for the API response for users with roles
type UsersWithRolesResponse struct {
	Status string                  `json:"status"`
	Data   []airfocus.UserWithRole `json:"data,omitempty"`
	Error  string                  `json:"error,omitempty"`
}

// UserWorkspacesResponse defines the structure for the API response for user workspaces
type UserWorkspacesResponse struct {
	Status string                         `json:"status"`
	Data   []airfocus.UserWorkspaceAccess `json:"data,omitempty"`
	Error  string                         `json:"error,omitempty"`
}

// handleGetUsersWithRoles retrieves and lists all users with their roles
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

// handleGetUserWorkspaces retrieves all workspaces a user has access to
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

	if apiKey == "" {
		json.NewEncoder(w).Encode(UserWorkspacesResponse{
			Status: "error",
			Error:  "API key is required",
		})
		return
	}

	if userID == "" {
		json.NewEncoder(w).Encode(UserWorkspacesResponse{
			Status: "error",
			Error:  "User ID is required",
		})
		return
	}

	client := airfocus.NewClient(apiKey)
	workspaces, err := client.GetUserWorkspaces(r.Context(), userID)

	response := UserWorkspacesResponse{}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else {
		response.Status = "success"
		response.Data = workspaces
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// TeamLicenseInfo represents the license information from the team endpoint
type TeamLicenseInfo struct {
	TeamID string `json:"teamId"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	State  struct {
		Features []string `json:"features"`
		Seats    struct {
			Admin struct {
				Total int `json:"total"`
				Used  int `json:"used"`
				Free  int `json:"free"`
			} `json:"admin"`
			Editor struct {
				Total int `json:"total"`
				Used  int `json:"used"`
				Free  int `json:"free"`
			} `json:"editor"`
			Contributor struct {
				Total int `json:"total"`
				Used  int `json:"used"`
				Free  int `json:"free"`
			} `json:"contributor"`
			Any struct {
				Total int `json:"total"`
				Used  int `json:"used"`
				Free  int `json:"free"`
			} `json:"any"`
		} `json:"seats"`
		Workspaces struct {
			Total int `json:"total"`
		} `json:"workspaces"`
		Subscription struct {
			Type string `json:"type"`
		} `json:"subscription"`
	} `json:"state"`
	Flags struct {
		EnableAi                  struct{ Value, Enforced, Explicit bool } `json:"enableAi"`
		EnableOkrApp              struct{ Value, Enforced, Explicit bool } `json:"enableOkrApp"`
		RemoveBranding            struct{ Value, Enforced, Explicit bool } `json:"removeBranding"`
		ForbidShareLinkCreation   struct{ Value, Enforced, Explicit bool } `json:"forbidShareLinkCreation"`
		RestrictShareLinkCreation struct{ Value, Enforced, Explicit bool } `json:"restrictShareLinkCreation"`
		RequireShareLinkPassword  struct{ Value, Enforced, Explicit bool } `json:"requireShareLinkPassword"`
		RequirePortalLogin        struct{ Value, Enforced, Explicit bool } `json:"requirePortalLogin"`
		RequirePortalPassword     struct{ Value, Enforced, Explicit bool } `json:"requirePortalPassword"`
	} `json:"flags"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

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

	// Web interface
	http.HandleFunc("/", server.handleIndex)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
