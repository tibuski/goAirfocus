package airfocus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	baseURL = "https://app.airfocus.com/api"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// WorkspaceSearchQuery represents the query parameters for workspace search
type WorkspaceSearchQuery struct {
	Sort struct {
		Type string `json:"type"`
		Name struct {
			Direction string `json:"direction"`
		} `json:"name"`
	} `json:"sort"`
	Archived bool `json:"archived"`
	Filter   *struct {
		Type          string `json:"type"`
		Mode          string `json:"mode"`
		Text          string `json:"text"`
		CaseSensitive bool   `json:"caseSensitive"`
	} `json:"filter,omitempty"`
}

// WorkspaceSearchQuerySort represents the sorting options for workspace search
type WorkspaceSearchQuerySort struct {
	Name WorkspaceSearchQuerySortName `json:"name"`
}

// WorkspaceSearchQuerySortName represents sorting by name
type WorkspaceSearchQuerySortName struct {
	Direction string `json:"direction"` // "asc" or "desc"
}

type Workspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	Description struct {
		Blocks []interface{} `json:"blocks"`
	} `json:"description"`
	ItemType      string `json:"itemType"`
	ItemColor     string `json:"itemColor"`
	ProgressMode  string `json:"progressMode"`
	Archived      bool   `json:"archived"`
	CreatedAt     string `json:"createdAt"`
	LastUpdatedAt string `json:"lastUpdatedAt"`
	Metadata      struct {
		Version    string `json:"version"`
		Duplicated bool   `json:"duplicated"`
	} `json:"metadata"`
	Namespace string `json:"namespace"`
	Order     int    `json:"order"`
	TeamID    string `json:"teamId"`
}

type WorkspaceResponse struct {
	Items      []Workspace `json:"items"`
	TotalItems int         `json:"totalItems"`
}

// WorkspaceResult represents the result of a workspace search
type WorkspaceResult struct {
	ID    string
	Alias string
}

// GetWorkspaceIDByName searches for a workspace by name and returns its ID and alias
func (c *Client) GetWorkspaceIDByName(ctx context.Context, name string) (WorkspaceResult, error) {
	name = strings.Trim(name, "\"")
	query := WorkspaceSearchQuery{
		Sort: struct {
			Type string `json:"type"`
			Name struct {
				Direction string `json:"direction"`
			} `json:"name"`
		}{
			Type: "name",
			Name: struct {
				Direction string `json:"direction"`
			}{
				Direction: "asc",
			},
		},
		Archived: false,
	}

	// Add filter for name search
	query.Filter = &struct {
		Type          string `json:"type"`
		Mode          string `json:"mode"`
		Text          string `json:"text"`
		CaseSensitive bool   `json:"caseSensitive"`
	}{
		Type:          "name",
		Mode:          "contain",
		Text:          name,
		CaseSensitive: false,
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/workspaces/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return WorkspaceResult{}, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Items) == 0 {
		return WorkspaceResult{}, fmt.Errorf("no workspace found with name: %s", name)
	}

	// Return both ID and alias of the first matching workspace
	return WorkspaceResult{
		ID:    result.Items[0].ID,
		Alias: result.Items[0].Alias,
	}, nil
}

type FieldSearchQuery struct {
	Filter struct {
		Type string `json:"type"`
		Name struct {
			Mode          string `json:"mode"`
			Text          string `json:"text"`
			CaseSensitive bool   `json:"caseSensitive"`
		} `json:"name"`
	} `json:"filter"`
	Sort struct {
		Type string `json:"type"`
		Name struct {
			Direction string `json:"direction"`
		} `json:"name"`
	} `json:"sort"`
}

type Field struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FieldResponse struct {
	Items      []Field `json:"items"`
	TotalItems int     `json:"totalItems"`
}

func (c *Client) GetFieldIDByName(ctx context.Context, name string) (string, error) {
	query := FieldSearchQuery{}
	query.Filter.Type = "name"
	query.Filter.Name.Mode = "contain"
	query.Filter.Name.Text = name
	query.Filter.Name.CaseSensitive = false
	query.Sort.Type = "name"
	query.Sort.Name.Direction = "asc"

	jsonData, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/fields/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result FieldResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("no field found with name: %s", name)
	}

	// Return the first matching field ID
	return result.Items[0].ID, nil
}

// ListWorkspaces returns a list of all workspaces accessible to the API key
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	query := WorkspaceSearchQuery{
		Sort: struct {
			Type string `json:"type"`
			Name struct {
				Direction string `json:"direction"`
			} `json:"name"`
		}{
			Type: "name",
			Name: struct {
				Direction string `json:"direction"`
			}{
				Direction: "asc",
			},
		},
		Archived: false,
		Filter:   nil,
	}

	reqBody, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/workspaces/search", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return result.Items, nil
}
