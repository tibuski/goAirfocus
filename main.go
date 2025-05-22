package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/tibus/goAirfocus/airfocus"
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

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// API endpoints
	http.HandleFunc("/api/workspace/id", server.handleGetWorkspaceID)
	http.HandleFunc("/api/field/id", server.handleGetFieldID)

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
			ID           string `json:"id"`
			Name         string `json:"name"`
			Alias        string `json:"alias"`
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

	// Web interface
	http.HandleFunc("/", server.handleIndex)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
