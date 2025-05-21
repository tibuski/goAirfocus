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

// Field represents a field in Airfocus
type Field struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	IsTeamField bool   `json:"isTeamField"`
	Embedded    struct {
		Workspaces []struct {
			WorkspaceID string `json:"workspaceId"`
			Order       int    `json:"order"`
		} `json:"workspaces"`
		AllWorkspaceIDs []string `json:"allWorkspaceIds"`
	} `json:"_embedded,omitempty"`
}

// GetWorkspaceCount returns the number of workspaces where this field is used
func (f *Field) GetWorkspaceCount() int {
	if f.IsTeamField && f.Embedded.AllWorkspaceIDs != nil {
		return len(f.Embedded.AllWorkspaceIDs)
	}
	if f.Embedded.Workspaces != nil {
		return len(f.Embedded.Workspaces)
	}
	return 0
}

// FieldSearchResponse represents the response from the field search API
type FieldSearchResponse struct {
	Items      []Field `json:"items"`
	TotalItems int     `json:"totalItems"`
}

// GetWorkspaceByID retrieves workspace details by ID
func (c *Client) GetWorkspaceByID(ctx context.Context, workspaceID string) (Workspace, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/workspaces/%s", baseURL, workspaceID), nil)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Workspace{}, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var workspace Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspace); err != nil {
		return Workspace{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return workspace, nil
}

// FieldWithWorkspaceNames extends Field with workspace names
type FieldWithWorkspaceNames struct {
	Field
	WorkspaceNames []string
}

// ListFields retrieves all fields accessible to the API key
func (c *Client) ListFields(ctx context.Context) ([]FieldWithWorkspaceNames, error) {
	// First, get all workspaces to build a lookup map
	workspaces, err := c.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Create a map of workspace IDs to names for quick lookup
	workspaceMap := make(map[string]string)
	for _, ws := range workspaces {
		workspaceMap[ws.ID] = ws.Name
	}

	query := struct {
		Query struct {
			Filter struct {
				TeamFields bool `json:"teamFields"`
			} `json:"filter"`
			Embed struct {
				Workspaces bool `json:"workspaces"`
			} `json:"embed"`
			Include struct {
				Workspaces bool `json:"workspaces"`
			} `json:"include"`
		} `json:"query"`
	}{
		Query: struct {
			Filter struct {
				TeamFields bool `json:"teamFields"`
			} `json:"filter"`
			Embed struct {
				Workspaces bool `json:"workspaces"`
			} `json:"embed"`
			Include struct {
				Workspaces bool `json:"workspaces"`
			} `json:"include"`
		}{
			Filter: struct {
				TeamFields bool `json:"teamFields"`
			}{
				TeamFields: true,
			},
			Embed: struct {
				Workspaces bool `json:"workspaces"`
			}{
				Workspaces: true,
			},
			Include: struct {
				Workspaces bool `json:"workspaces"`
			}{
				Workspaces: true,
			},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal field search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/fields/search", bytes.NewBuffer(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create field search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send field search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("field search request failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var searchResp FieldSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode field search response: %w", err)
	}

	// Convert fields to include workspace names
	fieldsWithNames := make([]FieldWithWorkspaceNames, len(searchResp.Items))
	for i, field := range searchResp.Items {
		fieldsWithNames[i] = FieldWithWorkspaceNames{
			Field: field,
		}

		// Get workspace names for both team fields and non-team fields
		var workspaceIDs []string
		if field.IsTeamField && field.Embedded.AllWorkspaceIDs != nil {
			workspaceIDs = field.Embedded.AllWorkspaceIDs
		} else if !field.IsTeamField && field.Embedded.Workspaces != nil {
			workspaceIDs = make([]string, len(field.Embedded.Workspaces))
			for j, ws := range field.Embedded.Workspaces {
				workspaceIDs[j] = ws.WorkspaceID
			}
		}

		if len(workspaceIDs) > 0 {
			workspaceNames := make([]string, 0, len(workspaceIDs))
			for _, wsID := range workspaceIDs {
				if name, ok := workspaceMap[wsID]; ok {
					workspaceNames = append(workspaceNames, name)
				}
			}
			fieldsWithNames[i].WorkspaceNames = workspaceNames
		}
	}

	return fieldsWithNames, nil
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
