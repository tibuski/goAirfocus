package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"sort"
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
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"getPermissionColorClass": getPermissionColorClass,
		"join":                    strings.Join,
		"permToString":            permToString,
		"mul": func(a, b int) int {
			return a * b
		},
	}).ParseFS(templatesFS, "templates/*.html", "templates/*_partial.html")
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

// handleListFieldsHTMX handles POST requests to list all fields and return HTML
func (s *Server) handleListFieldsHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list fields: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter out fields created on 2025-03-20 with empty updatedAt
	filteredFields := make([]airfocus.FieldWithWorkspaceNames, 0)
	for _, field := range fields {
		shouldFilter := field.CreatedAt != "" &&
			strings.HasPrefix(field.CreatedAt, "2025-03-20") &&
			field.UpdatedAt == ""
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

// handleGetUsersWithRolesHTMX handles POST requests to get users with roles and return HTML
func (s *Server) handleGetUsersWithRolesHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	users, err := client.FormatUsersWithRoles(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate HTML for the user dropdown
	var html strings.Builder

	// User dropdown section
	html.WriteString(`<div class="mb-4">
		<label for="userSelect" class="block text-sm font-medium text-gray-700 mb-2">Choose a user to view their workspaces:</label>
		<select id="userSelect" class="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	userID := r.FormValue("user_id")

	if apiKey == "" || userID == "" {
		http.Error(w, "API key and user ID are required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)

	// Add context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Refresh the workspace cache before getting user workspaces
	if err := client.RefreshCacheIfNeeded(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to refresh workspace cache: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user workspaces
	userWorkspaces, err := client.GetUserWorkspaces(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user workspaces: %v", err), http.StatusInternalServerError)
		return
	}

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

// handleGetWorkspacesHTMX fetches workspaces and returns HTML for a select dropdown.
func (s *Server) handleGetWorkspacesHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	workspaces, err := client.ListWorkspaces(r.Context())
	if err != nil {
		log.Printf("Error listing workspaces for HTMX: %v", err)
		http.Error(w, "Failed to retrieve workspaces", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Workspaces": workspaces,
	}

	// It's crucial to specify the partial template here.
	if err := s.templates.ExecuteTemplate(w, "workspace_select_partial.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleGetWorkspaceIDHTMX retrieves workspace ID and renders HTML.
func (s *Server) handleGetWorkspaceIDHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceID := r.FormValue("workspace_select") // Get selected workspace ID from the form

	if apiKey == "" || workspaceID == "" {
		http.Error(w, "API key and Workspace ID are required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	workspace, err := client.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		log.Printf("Error getting workspace ID %s: %v", workspaceID, err)
		http.Error(w, "Failed to retrieve workspace ID", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"WorkspaceID":    workspace.ID,
		"WorkspaceAlias": workspace.Alias,
	}

	// Render only the partial for the workspace ID
	if err := s.templates.ExecuteTemplate(w, "workspace_id_partial.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleGetWorkspaceUsersHTMX retrieves workspace users and renders HTML.
func (s *Server) handleGetWorkspaceUsersHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceID := r.FormValue("workspace_select") // Get selected workspace ID from the form

	if apiKey == "" || workspaceID == "" {
		http.Error(w, "API key and Workspace ID are required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	users, err := client.GetWorkspaceUsers(r.Context(), workspaceID)
	if err != nil {
		log.Printf("Error getting workspace users for ID %s: %v", workspaceID, err)
		http.Error(w, "Failed to retrieve workspace users", http.StatusInternalServerError)
		return
	}

	// Group users by permission
	groupedUsers := make(map[string][]airfocus.WorkspaceUser)
	for _, user := range users {
		permission := strings.ToLower(user.Permission)
		groupedUsers[permission] = append(groupedUsers[permission], user)
	}

	data := map[string]interface{}{
		"GroupedWorkspaces": groupedUsers,
	}

	// Render only the partial for workspace users
	if err := s.templates.ExecuteTemplate(w, "workspace_users_partial.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleGetUsersHTMX fetches users and renders a select dropdown
func (s *Server) handleGetUsersHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	users, err := client.FormatUsersWithRoles(r.Context())
	if err != nil {
		log.Printf("Error getting users with roles for HTMX: %v", err)
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Users": users,
	}

	if err := s.templates.ExecuteTemplate(w, "user_select_partial.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleGetUserInfoHTMX fetches and displays detailed user info and their workspaces.
func (s *Server) handleGetUserInfoHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	userID := r.FormValue("user_select") // Get selected user ID from the form

	if apiKey == "" || userID == "" {
		http.Error(w, "API key and User ID are required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	user, err := client.GetUser(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting user %s: %v", userID, err)
		http.Error(w, "Failed to retrieve user info", http.StatusInternalServerError)
		return
	}

	// Fetch user's group access (this includes workspaces within groups)
	userGroups, err := client.GetUserGroupAccess(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting user group access for user %s: %v", userID, err)
		http.Error(w, "Failed to retrieve user group access", http.StatusInternalServerError)
		return
	}

	// Extract all workspaces from the groups for the User Workspaces section
	var allWorkspaces []airfocus.Workspace
	for _, group := range userGroups {
		allWorkspaces = append(allWorkspaces, group.Embedded.Workspaces...)
	}

	// No longer grouping by permission at the top level
	// Instead, sort all user groups by name
	sort.Slice(userGroups, func(i, j int) bool {
		return userGroups[i].Name < userGroups[j].Name
	})

	// Build the hierarchical group tree
	hierarchicalGroups := buildGroupTree(userGroups)

	// Group workspaces by permission (this is for the User Workspaces section, if it's still needed)
	groupedWorkspaces := make(map[string][]airfocus.Workspace)
	for _, ws := range allWorkspaces {
		permission := string(ws.CurrentPermission)
		groupedWorkspaces[permission] = append(groupedWorkspaces[permission], ws)
	}

	data := map[string]interface{}{
		"User":       user,
		"Workspaces": groupedWorkspaces,  // Renamed from GroupedWorkspaces to avoid confusion if it's not grouped by permission here
		"UserGroups": hierarchicalGroups, // Pass the hierarchical list of user groups
	}

	if err := s.templates.ExecuteTemplate(w, "user_details_partial.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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

func permToString(p airfocus.Permission) string {
	return string(p)
}

// HierarchicalGroup represents a group in a tree structure
type HierarchicalGroup struct {
	airfocus.WorkspaceGroup
	Children []HierarchicalGroup
	Level    int // New field to indicate depth for indentation
}

// buildGroupTree builds a hierarchical tree of groups from a flat list.
func buildGroupTree(groups []airfocus.WorkspaceGroup) []HierarchicalGroup {
	groupMap := make(map[string]airfocus.WorkspaceGroup)
	for _, group := range groups {
		groupMap[group.ID] = group
	}

	childrenMap := make(map[string][]HierarchicalGroup)
	for _, group := range groups {
		hg := HierarchicalGroup{WorkspaceGroup: group}
		childrenMap[group.ParentID] = append(childrenMap[group.ParentID], hg)
	}

	var rootGroups []HierarchicalGroup
	for _, group := range groups {
		if group.ParentID == "" {
			hg := HierarchicalGroup{WorkspaceGroup: group, Level: 0} // Root level is 0
			rootGroups = append(rootGroups, hg)
		}
	}

	var attachChildren func(*HierarchicalGroup, int)
	attachChildren = func(hg *HierarchicalGroup, level int) {
		if children, ok := childrenMap[hg.ID]; ok {
			sort.Slice(children, func(i, j int) bool {
				return children[i].Name < children[j].Name
			})
			for i := range children {
				children[i].Level = level + 1 // Increment level for children
				attachChildren(&children[i], level+1)
			}
			hg.Children = children
		}
	}

	for i := range rootGroups {
		attachChildren(&rootGroups[i], 0) // Start recursion for root groups with level 0
	}

	// Sort root groups by name
	sort.Slice(rootGroups, func(i, j int) bool {
		return rootGroups[i].Name < rootGroups[j].Name
	})

	return rootGroups
}

// handleGetFieldSelectHTMX handles POST requests to get field selection dropdown
func (s *Server) handleGetFieldSelectHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list fields: %v", err), http.StatusInternalServerError)
		return
	}

	// Debug logging
	log.Printf("Loaded %d fields from API", len(fields))

	// Filter out fields created on 2025-03-20 with empty updatedAt
	filteredFields := make([]airfocus.FieldWithWorkspaceNames, 0)
	for _, field := range fields {
		shouldFilter := field.CreatedAt != "" &&
			strings.HasPrefix(field.CreatedAt, "2025-03-20") &&
			field.UpdatedAt == ""
		if !shouldFilter {
			filteredFields = append(filteredFields, field)
		}
	}

	// Debug logging
	log.Printf("After filtering: %d fields", len(filteredFields))

	// Generate HTML for the field dropdown
	var html strings.Builder
	html.WriteString(`<div class="mb-4">
		<label for="fieldSelect" class="block text-sm font-medium text-gray-700 mb-2">Choose a field to view details:</label>
		<select id="fieldSelect" name="fieldSelect"
				hx-post="/api/field/info/htmx"
				hx-target="#fieldDetailsResult"
				hx-swap="innerHTML"
				hx-trigger="change"
				hx-include="#apiKey, #fieldSelect"
				class="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
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

	html.WriteString(`</div>`)

	w.Write([]byte(html.String()))
}

// handleGetFieldInfoHTMX handles POST requests to get field details and renders HTML
func (s *Server) handleGetFieldInfoHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	fieldName := r.FormValue("fieldSelect")

	// Debug logging (never log API key)
	log.Printf("Field info request - Field Name: '%s'", fieldName)

	if apiKey == "" || fieldName == "" {
		http.Error(w, "API key and field name are required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	fields, err := client.ListFields(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch fields: %v", err), http.StatusInternalServerError)
		return
	}

	// Debug logging
	log.Printf("Found %d fields total", len(fields))

	// Find the field by name (case-insensitive)
	var foundField *airfocus.FieldWithWorkspaceNames
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			foundField = &field
			log.Printf("Found matching field: %s", field.Name)
			break
		}
	}

	if foundField == nil {
		http.Error(w, fmt.Sprintf("Field '%s' not found", fieldName), http.StatusNotFound)
		return
	}

	// Generate HTML for field details
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<!-- Field Details Block -->
		<div class="content-block h-full">
			<h3 class="text-xl font-semibold mb-2 text-gray-700">Field Details</h3>
			<div class="text-gray-700">
				<p><strong>ID:</strong> %s</p>
				<p><strong>Name:</strong> %s</p>
				<p><strong>Description:</strong> %s</p>
				<p><strong>Type:</strong> %s</p>
				<p><strong>Team Field:</strong> %t</p>
				<p><strong>Created At:</strong> %s</p>
				<p><strong>Updated At:</strong> %s</p>
			</div>
		</div>

		<!-- Field Workspaces Block -->
		<div class="content-block h-full">
			<h3 class="text-xl font-semibold mb-2 text-gray-700">Used in Workspaces</h3>
			<div class="space-y-4">`,
		foundField.ID,
		foundField.Name,
		foundField.Description,
		foundField.Type,
		foundField.IsTeamField,
		foundField.CreatedAt,
		foundField.UpdatedAt))

	if len(foundField.WorkspaceNames) > 0 {
		html.WriteString(fmt.Sprintf(`<div>
			<h4 class="text-lg font-medium text-gray-700 mb-2">Workspace Count: <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-blue-100 text-blue-800">%d</span></h4>
			<ul class="list-disc list-inside text-gray-700 ml-4">`, len(foundField.WorkspaceNames)))

		for _, workspaceName := range foundField.WorkspaceNames {
			html.WriteString(fmt.Sprintf(`<li>%s</li>`, workspaceName))
		}

		html.WriteString(`</ul></div>`)
	} else {
		html.WriteString(`<p class="text-gray-500">This field is not used in any workspaces.</p>`)
	}

	html.WriteString(`</div></div>`)

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

	// HTMX endpoints only (removed redundant JSON endpoints)
	http.HandleFunc("/api/fields/htmx", server.handleListFieldsHTMX)
	http.HandleFunc("/api/field/select/htmx", server.handleGetFieldSelectHTMX)
	http.HandleFunc("/api/field/info/htmx", server.handleGetFieldInfoHTMX)
	http.HandleFunc("/api/team/license/htmx", server.handleGetLicenseInfoHTMX)
	http.HandleFunc("/api/users/roles/htmx", server.handleGetUsersWithRolesHTMX)
	http.HandleFunc("/api/user/workspaces/htmx", server.handleGetUserWorkspacesHTMX)
	http.HandleFunc("/api/workspaces/htmx", server.handleGetWorkspacesHTMX)
	http.HandleFunc("/api/workspace/id/htmx", server.handleGetWorkspaceIDHTMX)
	http.HandleFunc("/api/workspace/users/htmx", server.handleGetWorkspaceUsersHTMX)
	http.HandleFunc("/api/users/htmx", server.handleGetUsersHTMX)
	http.HandleFunc("/api/user/info/htmx", server.handleGetUserInfoHTMX)

	// Root handler
	http.HandleFunc("/", server.handleIndex)

	log.Printf("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
