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

// --- Workspace Search Query Structs ---

// WorkspaceSearchSortName represents sorting by name direction
type WorkspaceSearchSortName struct {
	Direction string `json:"direction"` // "asc" or "desc"
}

// WorkspaceSearchSort represents the sorting options for workspace search
type WorkspaceSearchSort struct {
	Type string                  `json:"type"`
	Name WorkspaceSearchSortName `json:"name"`
}

// WorkspaceSearchFilter represents the filter options for workspace search
type WorkspaceSearchFilter struct {
	Type          string `json:"type"`
	Mode          string `json:"mode"`
	Text          string `json:"text"`
	CaseSensitive bool   `json:"caseSensitive"`
}

// WorkspaceSearchQuery represents the query parameters for workspace search
type WorkspaceSearchQuery struct {
	Sort     WorkspaceSearchSort    `json:"sort"`
	Archived bool                   `json:"archived"`
	Filter   *WorkspaceSearchFilter `json:"filter,omitempty"`
}

// --- End Workspace Search Query Structs ---

type Workspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	Description struct {
		Blocks []interface{} `json:"blocks"` // Consider json.RawMessage if just passing through
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
	Embedded  struct { // Correctly define _embedded to capture permissions
		Permissions map[string]string `json:"permissions"` // userId: Permission (string like "read", "write", "full", "comment")
	} `json:"_embedded,omitempty"`
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
	// Trim quotes if the input name is expected to be quoted.
	// Consider if the caller should provide a clean name instead.
	name = strings.Trim(name, "\"")

	query := WorkspaceSearchQuery{
		Sort: WorkspaceSearchSort{
			Type: "name",
			Name: WorkspaceSearchSortName{
				Direction: "asc",
			},
		},
		Archived: false,
		Filter: &WorkspaceSearchFilter{
			Type:          "name",
			Mode:          "contain",
			Text:          name,
			CaseSensitive: false,
		},
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to marshal workspace search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/workspaces/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to create workspace search request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to send workspace search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return WorkspaceResult{}, fmt.Errorf("airfocus API workspace search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return WorkspaceResult{}, fmt.Errorf("failed to decode workspace search response: %w", err)
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

// FieldSearchQuery matches the OpenAPI definition for /api/fields/search
// This replaces the old, unused FieldSearchQuery and the FieldListQuery related structs.
type FieldSearchQuery struct {
	IsTeamField  bool     `json:"isTeamField,omitempty"`
	WorkspaceIDs []string `json:"workspaceIds,omitempty"`
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
		AllWorkspaceIDs []string `json:"allWorkspaceIds"` // This field is not in OpenAPI spec, but API might return it. Go will unmarshal if present.
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
		return Workspace{}, fmt.Errorf("failed to create get workspace by ID request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to send get workspace by ID request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Workspace{}, fmt.Errorf("airfocus API get workspace by ID failed with status %d: %s", resp.StatusCode, string(body))
	}

	var workspace Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspace); err != nil {
		return Workspace{}, fmt.Errorf("failed to decode get workspace by ID response: %w", err)
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
		return nil, fmt.Errorf("failed to list workspaces for field lookup: %w", err)
	}

	// Create a map of workspace IDs to names for quick lookup
	workspaceMap := make(map[string]string)
	for _, ws := range workspaces {
		workspaceMap[ws.ID] = ws.Name
	}

	// Construct the query according to the OpenAPI spec for /api/fields/search
	// The spec shows a flat structure for FieldSearchQuery, not nested 'query' or 'filter' objects.
	query := FieldSearchQuery{
		IsTeamField:  true, // Assuming we want team fields as per original logic
		WorkspaceIDs: nil,  // No specific workspace IDs filter for listing all
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
		return nil, fmt.Errorf("airfocus API field search request failed with status: %d, body: %s", resp.StatusCode, string(body))
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
		Sort: WorkspaceSearchSort{
			Type: "name",
			Name: WorkspaceSearchSortName{
				Direction: "asc",
			},
		},
		Archived: false,
		Filter:   nil, // No filter for listing all
	}

	reqBody, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list workspaces query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/workspaces/search", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create list workspaces request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send list workspaces request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("airfocus API list workspaces failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode list workspaces response: %w", err)
	}

	return result.Items, nil
}

// User represents a team member from the /api/team/users endpoint
type User struct {
	UserID   string `json:"userId"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Role     string `json:"role"` // e.g., "admin", "contributor", "editor"
}

// ListUsers retrieves all users in the team
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/team/users", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list users request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send list users request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("airfocus API list users failed with status %d: %s", resp.StatusCode, string(body))
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode list users response: %w", err)
	}
	return users, nil
}

// WorkspaceUser combines user details with their specific permission for a workspace
type WorkspaceUser struct {
	UserID     string `json:"userId"`
	FullName   string `json:"fullName"`
	Email      string `json:"email"`
	Permission string `json:"permission"` // "read", "write", "full", "comment"
}

// GetWorkspaceUsers retrieves users and their permissions for a specific workspace
func (c *Client) GetWorkspaceUsers(ctx context.Context, workspaceID string) ([]WorkspaceUser, error) {
	// Get workspace details to access embedded permissions
	workspace, err := c.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace details for users: %w", err)
	}

	if workspace.Embedded.Permissions == nil || len(workspace.Embedded.Permissions) == 0 {
		return []WorkspaceUser{}, nil // No explicit permissions found
	}

	// Get all team users to map user IDs to names and emails
	allUsers, err := c.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all team users: %w", err)
	}

	userMap := make(map[string]User)
	for _, u := range allUsers {
		userMap[u.UserID] = u
	}

	var workspaceUsers []WorkspaceUser
	for userID, permission := range workspace.Embedded.Permissions {
		if user, ok := userMap[userID]; ok {
			workspaceUsers = append(workspaceUsers, WorkspaceUser{
				UserID:     userID,
				FullName:   user.FullName,
				Email:      user.Email,
				Permission: permission,
			})
		} else {
			// User not found in the team list, might be an external user or a deleted user.
			// Or, the API might return default permissions for users not explicitly listed.
			// For now, just add with ID and permission, indicating unknown name/email.
			workspaceUsers = append(workspaceUsers, WorkspaceUser{
				UserID:     userID,
				FullName:   "Unknown User", // Fallback
				Email:      "",
				Permission: permission,
			})
		}
	}

	return workspaceUsers, nil
}
