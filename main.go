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

//go:embed static/css/*
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

// handleListFieldsHTMX handles POST requests to list all fields and return HTML
func (s *Server) handleListFieldsHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleListFieldsHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		log.Printf("API key is missing")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating Airfocus client and calling ListFields for HTMX")
	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		log.Printf("Error listing fields: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list fields: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d fields for HTMX", len(fields))

	// Filter out fields created on 2025-03-20 with empty updatedAt
	filteredFields := make([]airfocus.FieldWithWorkspaceNames, 0)
	for _, field := range fields {
		shouldFilter := field.CreatedAt != "" &&
			strings.HasPrefix(field.CreatedAt, "2025-03-20") &&
			(field.UpdatedAt == "" || field.UpdatedAt == "")
		if !shouldFilter {
			filteredFields = append(filteredFields, field)
		}
	}

	// Generate HTML for the field dropdown
	var html strings.Builder
	html.WriteString(`<div class="mb-4">
		<label for="fieldSelect" class="block text-sm font-medium text-gray-700 mb-2">Choose from list:</label>
		<select id="fieldSelect" class="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
			<option value="">Select a field...</option>`)

	if len(filteredFields) > 0 {
		for _, field := range filteredFields {
			displayText := field.Name
			if field.IsTeamField {
				displayText += " (Team Field)"
				if len(field.WorkspaceNames) > 0 {
					displayText += fmt.Sprintf(" - Used in %d workspaces", len(field.WorkspaceNames))
				}
			} else {
				if len(field.WorkspaceNames) > 0 {
					displayText += fmt.Sprintf(" (%s)", strings.Join(field.WorkspaceNames, ", "))
				}
			}
			html.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, field.Name, displayText))
		}
		html.WriteString(fmt.Sprintf(`</select>
		<p class="mt-2 text-sm text-green-600">✓ Loaded %d fields successfully</p>`, len(filteredFields)))
	} else {
		html.WriteString(`</select>
		<p class="mt-2 text-sm text-gray-500">No fields found for this API key.</p>`)
	}

	html.WriteString(`</div>
	<div class="mb-4">
		<label for="fieldName" class="block text-sm font-medium text-gray-700 mb-2">Or enter field name:</label>
		<input type="text" 
			   id="fieldName" 
			   placeholder="Enter field name (spaces are allowed)" 
			   class="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
		<p class="mt-1 text-sm text-gray-500">Field names can contain spaces and will be matched partially</p>
	</div>`)

	w.Write([]byte(html.String()))
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

// handleGetUsersWithRolesHTMX handles POST requests to get users with roles and return HTML
func (s *Server) handleGetUsersWithRolesHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetUsersWithRolesHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		log.Printf("API key is missing")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating Airfocus client and calling FormatUsersWithRoles for HTMX")
	client := airfocus.NewClient(apiKey)
	users, err := client.FormatUsersWithRoles(r.Context())
	if err != nil {
		log.Printf("Error getting users with roles: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d users for HTMX", len(users))

	// Generate HTML for the user dropdown
	var html strings.Builder

	// User dropdown section
	html.WriteString(`<div class="mb-4">
		<label for="userSelect" class="block text-sm font-medium text-gray-700 mb-2">Choose a user to view their workspaces:</label>
		<select id="userSelect" onchange="loadUserWorkspaces()" class="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
			<option value="">Select a user...</option>`)

	if len(users) > 0 {
		for _, user := range users {
			displayText := fmt.Sprintf("%s (%s)", user.FullName, user.Email)
			if user.Role != "" {
				displayText += fmt.Sprintf(" - %s", user.Role)
			}
			html.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, user.UserID, displayText))
		}
		html.WriteString(fmt.Sprintf(`</select>
		<p class="mt-2 text-sm text-green-600">✓ Loaded %d users successfully</p>`, len(users)))
	} else {
		html.WriteString(`</select>
		<p class="mt-2 text-sm text-gray-500">No users found for this API key.</p>`)
	}

	html.WriteString(`</div>`)

	w.Write([]byte(html.String()))
}

// handleGetUserWorkspacesHTMX handles POST requests to get user workspaces and return HTML
func (s *Server) handleGetUserWorkspacesHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetUserWorkspacesHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	userID := r.FormValue("user_id")

	if apiKey == "" || userID == "" {
		log.Printf("API key or user ID is missing")
		http.Error(w, "API key and user ID are required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating Airfocus client and calling GetUserWorkspaces for HTMX")
	client := airfocus.NewClient(apiKey)

	// Add context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Refresh the workspace cache before getting user workspaces
	if err := client.RefreshCacheIfNeeded(ctx); err != nil {
		log.Printf("Failed to refresh workspace cache: %v", err)
		http.Error(w, fmt.Sprintf("Failed to refresh workspace cache: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user workspaces
	userWorkspaces, err := client.GetUserWorkspaces(ctx, userID)
	if err != nil {
		log.Printf("Error getting user workspaces: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get user workspaces: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d workspaces for user %s", len(userWorkspaces), userID)

	// Generate HTML for the user workspaces tree view
	var html strings.Builder

	if len(userWorkspaces) > 0 {
		// Group workspaces by group
		groupMap := make(map[string][]airfocus.UserWorkspaceAccess)
		ungroupedWorkspaces := []airfocus.UserWorkspaceAccess{}

		for _, workspace := range userWorkspaces {
			if workspace.GroupName != "" {
				groupMap[workspace.GroupName] = append(groupMap[workspace.GroupName], workspace)
			} else {
				ungroupedWorkspaces = append(ungroupedWorkspaces, workspace)
			}
		}

		html.WriteString(fmt.Sprintf(`<h4 class="text-lg font-medium text-gray-700 mb-2">User Workspaces & Permissions (%d workspaces)</h4>
		<div class="space-y-2 max-h-96 overflow-y-auto">`, len(userWorkspaces)))

		// Display grouped workspaces
		for groupName, workspaces := range groupMap {
			html.WriteString(fmt.Sprintf(`<div class="border border-gray-200 rounded-md overflow-hidden">
				<div class="bg-gray-50 px-3 py-2 border-b border-gray-200">
					<h5 class="text-sm font-medium text-gray-700 flex items-center">
						<svg class="w-4 h-4 mr-2 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
						</svg>
						%s
					</h5>
				</div>
				<div class="bg-white">`, groupName))

			for _, workspace := range workspaces {
				html.WriteString(fmt.Sprintf(`<div class="px-3 py-2 border-b border-gray-100 last:border-b-0 flex items-center justify-between">
					<div class="flex items-center">
						<svg class="w-3 h-3 mr-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
						</svg>
						<span class="text-sm text-gray-900">%s</span>
					</div>
					<span class="inline-flex px-2 py-1 text-xs font-semibold rounded-full %s">%s</span>
				</div>`,
					workspace.WorkspaceName,
					getPermissionColorClass(workspace.Permission),
					workspace.Permission))
			}

			html.WriteString(`</div>
			</div>`)
		}

		// Display ungrouped workspaces
		if len(ungroupedWorkspaces) > 0 {
			html.WriteString(`<div class="border border-gray-200 rounded-md overflow-hidden">
				<div class="bg-gray-50 px-3 py-2 border-b border-gray-200">
					<h5 class="text-sm font-medium text-gray-700 flex items-center">
						<svg class="w-4 h-4 mr-2 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
						</svg>
						Ungrouped Workspaces
					</h5>
				</div>
				<div class="bg-white">`)

			for _, workspace := range ungroupedWorkspaces {
				html.WriteString(fmt.Sprintf(`<div class="px-3 py-2 border-b border-gray-100 last:border-b-0 flex items-center justify-between">
					<div class="flex items-center">
						<svg class="w-3 h-3 mr-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
						</svg>
						<span class="text-sm text-gray-900">%s</span>
					</div>
					<span class="inline-flex px-2 py-1 text-xs font-semibold rounded-full %s">%s</span>
				</div>`,
					workspace.WorkspaceName,
					getPermissionColorClass(workspace.Permission),
					workspace.Permission))
			}

			html.WriteString(`</div>
			</div>`)
		}

		html.WriteString(`</div>`)
	} else {
		html.WriteString(`<p class="text-gray-500">No workspaces found for this user.</p>`)
	}

	w.Write([]byte(html.String()))
}

// getPermissionColorClass returns the appropriate CSS class for permission styling
func getPermissionColorClass(permission string) string {
	switch strings.ToLower(permission) {
	case "full":
		return "bg-red-100 text-red-800"
	case "write":
		return "bg-blue-100 text-blue-800"
	case "comment":
		return "bg-yellow-100 text-yellow-800"
	case "read":
		return "bg-green-100 text-green-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// handleGetLicenseInfoHTMX handles POST requests to get license information and return HTML
func (s *Server) handleGetLicenseInfoHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetLicenseInfoHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		log.Printf("API key is missing")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	log.Printf("Making request to Airfocus API for license info")

	// Make request to Airfocus API for license info
	req, err := http.NewRequest("GET", "https://api.airfocus.com/api/team", nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request to Airfocus API: %v", err)
		http.Error(w, "Error making request to Airfocus API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var licenseInfo TeamLicenseInfo
	if err := json.NewDecoder(resp.Body).Decode(&licenseInfo); err != nil {
		log.Printf("Error decoding response: %v", err)
		http.Error(w, "Error decoding response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved license info for HTMX")

	// Get actual user data for role statistics
	airfocusClient := airfocus.NewClient(apiKey)
	users, err := airfocusClient.FormatUsersWithRoles(r.Context())
	if err != nil {
		log.Printf("Error getting users with roles: %v", err)
		// Continue with license info only if user data fails
	}

	// Count users by role
	roleStats := struct {
		Total       int
		Admin       int
		Editor      int
		Contributor int
	}{}

	if err == nil && len(users) > 0 {
		roleStats.Total = len(users)
		for _, user := range users {
			switch strings.ToLower(user.Role) {
			case "admin":
				roleStats.Admin++
			case "editor":
				roleStats.Editor++
			case "contributor":
				roleStats.Contributor++
			}
		}
	} else {
		// Fallback to license seat data if user data is not available
		roleStats.Total = licenseInfo.State.Seats.Admin.Used + licenseInfo.State.Seats.Editor.Used + licenseInfo.State.Seats.Contributor.Used
		roleStats.Admin = licenseInfo.State.Seats.Admin.Used
		roleStats.Editor = licenseInfo.State.Seats.Editor.Used
		roleStats.Contributor = licenseInfo.State.Seats.Contributor.Used
	}

	// Generate HTML for the license information
	var html strings.Builder

	html.WriteString(`<div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6">
		<div class="bg-white p-4 rounded-lg shadow text-center">
			<div class="text-2xl font-bold text-blue-600">` + fmt.Sprintf("%d", licenseInfo.State.Seats.Any.Total) + `</div>
			<div class="text-sm text-gray-600">Total Licenses</div>
		</div>
		<div class="bg-white p-4 rounded-lg shadow text-center">
			<div class="text-2xl font-bold text-green-600">` + fmt.Sprintf("%d", licenseInfo.State.Seats.Any.Used) + `</div>
			<div class="text-sm text-gray-600">Used Licenses</div>
		</div>
		<div class="bg-white p-4 rounded-lg shadow text-center">
			<div class="text-2xl font-bold text-yellow-600">` + fmt.Sprintf("%d", licenseInfo.State.Seats.Any.Free) + `</div>
			<div class="text-sm text-gray-600">Free Licenses</div>
		</div>
	</div>`)

	// Role Statistics
	html.WriteString(`<div class="mt-6">
		<h3 class="text-lg font-medium text-gray-700 mb-3">Role Statistics</h3>
		<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
			<div class="bg-white p-4 rounded-lg shadow text-center">
				<div class="text-2xl font-bold text-blue-600">` + fmt.Sprintf("%d", roleStats.Total) + `</div>
				<div class="text-sm text-gray-600">Total Users</div>
			</div>
			<div class="bg-white p-4 rounded-lg shadow text-center">
				<div class="text-2xl font-bold text-purple-600">` + fmt.Sprintf("%d", roleStats.Admin) + `</div>
				<div class="text-sm text-gray-600">Admins</div>
			</div>
			<div class="bg-white p-4 rounded-lg shadow text-center">
				<div class="text-2xl font-bold text-green-600">` + fmt.Sprintf("%d", roleStats.Editor) + `</div>
				<div class="text-sm text-gray-600">Editors</div>
			</div>
			<div class="bg-white p-4 rounded-lg shadow text-center">
				<div class="text-2xl font-bold text-yellow-600">` + fmt.Sprintf("%d", roleStats.Contributor) + `</div>
				<div class="text-sm text-gray-600">Contributors</div>
			</div>
		</div>
	</div>`)

	w.Write([]byte(html.String()))
}

// handleGetWorkspacesHTMX handles POST requests to get workspaces and return HTML
func (s *Server) handleGetWorkspacesHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetWorkspacesHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		log.Printf("API key is missing")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating Airfocus client and calling ListWorkspaces for HTMX")
	client := airfocus.NewClient(apiKey)
	workspaces, err := client.ListWorkspaces(r.Context())
	if err != nil {
		log.Printf("Error getting workspaces: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get workspaces: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d workspaces for HTMX", len(workspaces))

	// Generate HTML for the workspace dropdown
	var html strings.Builder

	html.WriteString(`<div class="flex flex-col md:flex-row gap-4 mb-4 items-end">
		<div class="flex-1 w-full">
			<label for="workspaceSelect" class="block text-sm font-medium text-gray-700">Choose from list:</label>
			<select id="workspaceSelect" 
					name="workspace_select"
					class="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
					hx-post="/api/workspace/id/htmx"
					hx-target="#workspaceResult"
					hx-swap="innerHTML"
					hx-trigger="change from:body"
					hx-include="#apiKey, #workspaceSelect"
					hx-indicator="#workspaceIDLoadingIndicator"
					hx-on:change="htmx.trigger('#workspaceUsersTrigger', 'change')">
				<option value="">Select a workspace...</option>`)

	if len(workspaces) > 0 {
		for _, workspace := range workspaces {
			displayText := workspace.Name
			if workspace.Alias != "" {
				displayText += fmt.Sprintf(" (%s)", workspace.Alias)
			}
			html.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, workspace.ID, displayText))
		}
		html.WriteString(fmt.Sprintf(`</select>
		<p class="mt-2 text-sm text-green-600">✓ Loaded %d workspaces successfully</p>`, len(workspaces)))
	} else {
		html.WriteString(`</select>
		<p class="mt-2 text-sm text-gray-500">No workspaces found for this API key.</p>`)
	}

	html.WriteString(`</div>
	</div>`)

	w.Write([]byte(html.String()))
}

// handleGetFieldIDHTMX handles POST requests to get field ID and return HTML
func (s *Server) handleGetFieldIDHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetFieldIDHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	fieldName := r.FormValue("field_name")

	if apiKey == "" || fieldName == "" {
		log.Printf("API key or field name is missing")
		http.Error(w, "API key and field name are required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating Airfocus client and calling ListFields for HTMX")
	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		log.Printf("Error getting fields: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get fields: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d fields for HTMX", len(fields))

	// Find the field by name
	var foundField *airfocus.FieldWithWorkspaceNames
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			foundField = &field
			break
		}
	}

	// Generate HTML for the field result
	var html strings.Builder

	if foundField != nil {
		html.WriteString(fmt.Sprintf(`<div class="bg-green-50 border border-green-200 rounded-md p-4">
			<h4 class="text-lg font-medium text-green-800 mb-2">Field Found</h4>
			<div class="space-y-2">
				<div><strong>Field ID:</strong> <code class="bg-gray-100 px-2 py-1 rounded">%s</code></div>
				<div><strong>Name:</strong> %s</div>
				<div><strong>Type:</strong> %s</div>
				<div><strong>Description:</strong> %s</div>
				<div><strong>Team Field:</strong> %t</div>`,
			foundField.ID,
			foundField.Name,
			foundField.Type,
			foundField.Description,
			foundField.IsTeamField))

		if len(foundField.WorkspaceNames) > 0 {
			html.WriteString(fmt.Sprintf(`<div><strong>Used in Workspaces:</strong> %s</div>`, strings.Join(foundField.WorkspaceNames, ", ")))
		}

		html.WriteString(`</div>
		</div>`)
	} else {
		html.WriteString(fmt.Sprintf(`<div class="bg-red-50 border border-red-200 rounded-md p-4">
			<h4 class="text-lg font-medium text-red-800 mb-2">Field Not Found</h4>
			<p class="text-red-700">No field found with name: "%s"</p>
			<p class="text-sm text-red-600 mt-2">Try using the "Get Fields" button above to see available fields.</p>
		</div>`, fieldName))
	}

	w.Write([]byte(html.String()))
}

// handleGetWorkspaceIDHTMX handles POST requests to get workspace ID and return HTML
func (s *Server) handleGetWorkspaceIDHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetWorkspaceIDHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceName := r.FormValue("workspace_name")
	workspaceSelect := r.FormValue("workspace_select")

	if apiKey == "" {
		log.Printf("API key is missing")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Determine which workspace to use
	var workspaceID string
	if workspaceSelect != "" {
		workspaceID = workspaceSelect
	} else if workspaceName != "" {
		// Search for workspace by name
		client := airfocus.NewClient(apiKey)
		workspaces, err := client.ListWorkspaces(r.Context())
		if err != nil {
			log.Printf("Error getting workspaces: %v", err)
			http.Error(w, fmt.Sprintf("Failed to get workspaces: %v", err), http.StatusInternalServerError)
			return
		}

		// Find workspace by name (case-insensitive, partial match)
		for _, ws := range workspaces {
			if strings.Contains(strings.ToLower(ws.Name), strings.ToLower(workspaceName)) {
				workspaceID = ws.ID
				break
			}
		}
	}

	// Generate HTML for the workspace result
	var html strings.Builder

	if workspaceID != "" {
		html.WriteString(fmt.Sprintf(`<div class="bg-green-50 border border-green-200 rounded-md p-4">
			<h4 class="text-lg font-medium text-green-800 mb-2">Workspace ID</h4>
			<div class="space-y-2">
				<code class="bg-gray-100 px-2 py-1 rounded">%s</code>
			</div>
		</div>`, workspaceID))
	} else {
		html.WriteString(`<div class="bg-yellow-50 border border-yellow-200 rounded-md p-4">
			<h4 class="text-lg font-medium text-yellow-800 mb-2">No Workspace Selected</h4>
			<p class="text-yellow-700">Select a workspace from the dropdown to see its ID.</p>
		</div>`)
	}

	w.Write([]byte(html.String()))
}

// handleGetWorkspaceUsersHTMX handles POST requests to get workspace users and return HTML
func (s *Server) handleGetWorkspaceUsersHTMX(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGetWorkspaceUsersHTMX called - Method: %s", r.Method)
	w.Header().Set("Content-Type", "text/html")

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceID := r.FormValue("workspace_id")

	if apiKey == "" {
		log.Printf("API key is missing")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	if workspaceID == "" {
		// Try to get workspace ID from workspace select or name
		workspaceSelect := r.FormValue("workspace_select")
		workspaceName := r.FormValue("workspace_name")

		if workspaceSelect != "" {
			workspaceID = workspaceSelect
		} else if workspaceName != "" {
			// Search for workspace by name
			client := airfocus.NewClient(apiKey)
			workspaces, err := client.ListWorkspaces(r.Context())
			if err != nil {
				log.Printf("Error getting workspaces: %v", err)
				http.Error(w, fmt.Sprintf("Failed to get workspaces: %v", err), http.StatusInternalServerError)
				return
			}

			// Find workspace by name (case-insensitive, partial match)
			for _, ws := range workspaces {
				if strings.Contains(strings.ToLower(ws.Name), strings.ToLower(workspaceName)) {
					workspaceID = ws.ID
					break
				}
			}
		}
	}

	if workspaceID == "" {
		http.Error(w, "Workspace ID or name is required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	workspaceUsers, err := client.GetWorkspaceUsers(r.Context(), workspaceID)
	if err != nil {
		log.Printf("Error getting workspace users: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get workspace users: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate HTML for the workspace users
	var html strings.Builder

	if len(workspaceUsers) > 0 {
		// Group users by permission
		usersByPermission := make(map[string][]airfocus.WorkspaceUser)
		for _, user := range workspaceUsers {
			usersByPermission[user.Permission] = append(usersByPermission[user.Permission], user)
		}

		// Define permission order (highest to lowest) - correct Airfocus permissions
		permissionOrder := []string{"full", "write", "comment", "read"}

		html.WriteString(fmt.Sprintf(`<div class="bg-green-50 border border-green-200 rounded-md p-4">
			<h4 class="text-lg font-medium text-green-800 mb-4">Workspace Users (%d total)</h4>
			<div class="space-y-6">`, len(workspaceUsers)))

		// Generate sections for each permission level
		for _, permission := range permissionOrder {
			if users, exists := usersByPermission[permission]; exists {
				// Get color classes for this permission
				colorClass := getPermissionColorClass(permission)
				borderColor := "border-gray-400"
				textColor := "text-gray-800"

				// Set specific colors based on permission
				switch permission {
				case "full":
					borderColor = "border-red-400"
					textColor = "text-red-800"
				case "write":
					borderColor = "border-blue-400"
					textColor = "text-blue-800"
				case "comment":
					borderColor = "border-yellow-400"
					textColor = "text-yellow-800"
				case "read":
					borderColor = "border-green-400"
					textColor = "text-green-800"
				}

				html.WriteString(fmt.Sprintf(`<div class="border-l-4 %s pl-4">
					<h5 class="text-md font-semibold %s mb-2 flex items-center">
						<span class="px-2 py-1 rounded text-sm %s mr-2">%s</span>
						<span>(%d users)</span>
					</h5>
					<div class="space-y-2 ml-4">`,
					borderColor,
					textColor,
					colorClass,
					strings.Title(permission),
					len(users)))

				for _, user := range users {
					html.WriteString(fmt.Sprintf(`<div class="text-sm text-gray-700">
						• %s
					</div>`, user.FullName))
				}

				html.WriteString(`</div>
				</div>`)
			}
		}

		html.WriteString(`</div>
		</div>`)
	} else {
		html.WriteString(`<div class="bg-yellow-50 border border-yellow-200 rounded-md p-4">
			<h4 class="text-lg font-medium text-yellow-800 mb-2">No Users Found</h4>
			<p class="text-yellow-700">No users found for this workspace.</p>
		</div>`)
	}

	w.Write([]byte(html.String()))
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

	// Add the new HTMX route for listing fields
	http.HandleFunc("/api/fields/htmx", server.handleListFieldsHTMX)

	// Add the new endpoint
	http.HandleFunc("/api/team/license", handleGetTeamLicense)

	// Add the new endpoint for listing contributors
	http.HandleFunc("/api/contributors", server.handleGetContributors)

	// Add the new endpoint for listing users with roles and return HTML
	http.HandleFunc("/api/users/roles/htmx", server.handleGetUsersWithRolesHTMX)

	// Add the new endpoint for getting user workspaces and return HTML
	http.HandleFunc("/api/user/workspaces/htmx", server.handleGetUserWorkspacesHTMX)

	// Add the new endpoint for getting license information and return HTML
	http.HandleFunc("/api/team/license/htmx", server.handleGetLicenseInfoHTMX)

	// Add the new endpoint for getting workspaces and return HTML
	http.HandleFunc("/api/workspaces/htmx", server.handleGetWorkspacesHTMX)

	// Add the new endpoint for getting field ID and return HTML
	http.HandleFunc("/api/field/id/htmx", server.handleGetFieldIDHTMX)

	// Add the new endpoint for getting workspace ID and return HTML
	http.HandleFunc("/api/workspace/id/htmx", server.handleGetWorkspaceIDHTMX)

	// Add the new endpoint for getting workspace users and return HTML
	http.HandleFunc("/api/workspace/users/htmx", server.handleGetWorkspaceUsersHTMX)

	// Web interface
	http.HandleFunc("/", server.handleIndex)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
