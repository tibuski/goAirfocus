package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/tibus/goAirfocus/airfocus"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*

var staticFS embed.FS

type Server struct {
	templates *template.Template
	mu        sync.RWMutex      // protects apiKeys
	apiKeys   map[string]string // session ID -> API key
}

func NewServer() (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Server{
		templates: tmpl,
		apiKeys:   make(map[string]string),
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	workspaceName := r.FormValue("workspace_name")

	if apiKey == "" || workspaceName == "" {
		http.Error(w, "API key and workspace name are required", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

type FieldIDResponse struct {
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (s *Server) handleGetFieldID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.FormValue("api_key")
	fieldName := r.FormValue("field_name")

	if apiKey == "" || fieldName == "" {
		http.Error(w, "API key and field name are required", http.StatusBadRequest)
		return
	}

	client := airfocus.NewClient(apiKey)
	fieldID, err := client.GetFieldIDByName(r.Context(), fieldName)

	response := FieldIDResponse{}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else {
		response.Status = "success"
		response.ID = fieldID
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
			w.Header().Set("Content-Type", "application/json")
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
			Description  string `json:"description"`
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
				Description:  fmt.Sprintf("%v", ws.Description.Blocks), // Convert blocks to string
				ItemType:     ws.ItemType,
				ProgressMode: ws.ProgressMode,
				Archived:     ws.Archived,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   summaries,
		})
	})

	// Web interface
	http.HandleFunc("/", server.handleIndex)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
