package airfocus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	baseURL = "https://app.airfocus.com/api"
)

type Client struct {
	apiKey     string
	httpClient *http.Client

	// Cache fields
	cache struct {
		users      []User
		workspaces []Workspace
		fields     []FieldWithWorkspaceNames
		lastUpdate time.Time
	}
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		cacheTTL:   5 * time.Minute, // Cache data for 5 minutes
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

// WorkspaceGroup represents a group that can contain workspaces and other groups
type WorkspaceGroup struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	ParentID          string `json:"parentId,omitempty"`
	Order             int    `json:"order"`
	DefaultPermission string `json:"defaultPermission"`
	CreatedAt         string `json:"createdAt"`
	LastUpdatedAt     string `json:"lastUpdatedAt"`
	TeamID            string `json:"teamId"`
	Embedded          struct {
		Workspaces []Workspace `json:"workspaces,omitempty"`
	} `json:"_embedded,omitempty"`
}

// WorkspaceGroupSearchQuery represents the query parameters for workspace group search
type WorkspaceGroupSearchQuery struct {
	Sort struct {
		Type      string `json:"type"`
		Direction string `json:"direction"`
	} `json:"sort"`
}

// WorkspaceGroupResponse represents the response from the workspace group search API
type WorkspaceGroupResponse struct {
	Items      []WorkspaceGroup `json:"items"`
	TotalItems int              `json:"totalItems"`
}

// ListWorkspaceGroups retrieves all workspace groups and their hierarchy
func (c *Client) ListWorkspaceGroups(ctx context.Context) ([]WorkspaceGroup, error) {
	query := WorkspaceGroupSearchQuery{
		Sort: struct {
			Type      string `json:"type"`
			Direction string `json:"direction"`
		}{
			Type:      "name",
			Direction: "asc",
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace group search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/workspaces/groups/search", bytes.NewBuffer(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace group search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send workspace group search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("airfocus API workspace group search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkspaceGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode workspace group search response: %w", err)
	}

	return result.Items, nil
}

// GetWorkspaceHierarchy returns a map of workspace groups organized by their hierarchy
func (c *Client) GetWorkspaceHierarchy(ctx context.Context) (map[string][]WorkspaceGroup, error) {
	groups, err := c.ListWorkspaceGroups(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map to store groups by their parent ID
	hierarchy := make(map[string][]WorkspaceGroup)

	// First, add all groups to the map
	for _, group := range groups {
		parentID := group.ParentID
		if parentID == "" {
			parentID = "root" // Use "root" for top-level groups
		}
		hierarchy[parentID] = append(hierarchy[parentID], group)
	}

	// Sort groups within each level by their order
	for parentID := range hierarchy {
		sort.Slice(hierarchy[parentID], func(i, j int) bool {
			return hierarchy[parentID][i].Order < hierarchy[parentID][j].Order
		})
	}

	return hierarchy, nil
}

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
	Namespace string   `json:"namespace"`
	Order     int      `json:"order"`
	TeamID    string   `json:"teamId"`
	Embedded  struct { // Correctly define _embedded to capture permissions
		Permissions map[string]string `json:"permissions"` // userId: Permission (string like "read", "write", "full", "comment")
	} `json:"_embedded,omitempty"`
	GroupID   string `json:"groupId,omitempty"`   // ID of the group this workspace belongs to
	GroupName string `json:"groupName,omitempty"` // Name of the group this workspace belongs to
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
	if err := c.refreshCacheIfNeeded(ctx); err != nil {
		return nil, err
	}

	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	// Return a copy of the cached fields
	fields := make([]FieldWithWorkspaceNames, len(c.cache.fields))
	copy(fields, c.cache.fields)
	return fields, nil
}

// ListWorkspaces returns a list of all workspaces accessible to the API key
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	if err := c.refreshCacheIfNeeded(ctx); err != nil {
		return nil, err
	}

	// Get workspace groups to map workspaces to their groups
	groups, err := c.ListWorkspaceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace groups: %w", err)
	}

	// Create a map of workspace IDs to their group information
	workspaceGroupMap := make(map[string]struct {
		GroupID   string
		GroupName string
	})
	for _, group := range groups {
		for _, ws := range group.Embedded.Workspaces {
			workspaceGroupMap[ws.ID] = struct {
				GroupID   string
				GroupName string
			}{
				GroupID:   group.ID,
				GroupName: group.Name,
			}
		}
	}

	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	// Return a copy of the cached workspaces with group information
	workspaces := make([]Workspace, len(c.cache.workspaces))
	copy(workspaces, c.cache.workspaces)

	// Add group information to each workspace
	for i := range workspaces {
		if groupInfo, ok := workspaceGroupMap[workspaces[i].ID]; ok {
			workspaces[i].GroupID = groupInfo.GroupID
			workspaces[i].GroupName = groupInfo.GroupName
		}
	}

	return workspaces, nil
}

// User represents a team member from the /api/team/users endpoint
type User struct {
	UserID   string `json:"userId"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Role     string `json:"role"` // e.g., "admin", "contributor", "editor"
}

// ListUsers retrieves all users in the team (now uses cache)
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	if err := c.refreshCacheIfNeeded(ctx); err != nil {
		return nil, err
	}

	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	// Create a copy of the cached users
	users := make([]User, len(c.cache.users))
	copy(users, c.cache.users)

	// Sort users alphabetically by full name
	sort.Slice(users, func(i, j int) bool {
		return strings.ToLower(users[i].FullName) < strings.ToLower(users[j].FullName)
	})

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

// UserState represents the state of a user in the team
type UserState struct {
	Pending  bool `json:"pending"`
	Unseated bool `json:"unseated"`
}

// UserWithRole represents a user with their role information
type UserWithRole struct {
	UserID   string     `json:"userId"`
	FullName string     `json:"fullName"`
	Email    string     `json:"email"`
	Role     string     `json:"role"`
	State    *UserState `json:"state,omitempty"`
}

// UserWorkspaceAccess represents a workspace and the user's permission level for it
type UserWorkspaceAccess struct {
	WorkspaceID   string `json:"workspaceId"`
	WorkspaceName string `json:"workspaceName"`
	Permission    string `json:"permission"`
	GroupID       string `json:"groupId,omitempty"`
	GroupName     string `json:"groupName,omitempty"`
	GroupPath     string `json:"groupPath,omitempty"` // Full path to the group (e.g., "Parent Group > Child Group")
}

// FormatUsersWithRoles returns a list of users with their roles in parentheses
func (c *Client) FormatUsersWithRoles(ctx context.Context) ([]UserWithRole, error) {
	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	formattedUsers := make([]UserWithRole, len(users))
	for i, user := range users {
		formattedUsers[i] = UserWithRole{
			UserID:   user.UserID,
			FullName: user.FullName,
			Email:    user.Email,
			Role:     user.Role,
		}
	}

	return formattedUsers, nil
}

// GetUserWorkspaces retrieves all workspaces a user has access to and their permission level
func (c *Client) GetUserWorkspaces(ctx context.Context, userID string) ([]UserWorkspaceAccess, error) {
	// First get all workspaces with their group information
	workspaces, err := c.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Get workspace groups to build the hierarchy
	groups, err := c.ListWorkspaceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace groups: %w", err)
	}

	// Build a map of group IDs to their full paths
	groupPaths := make(map[string]string)
	groupNames := make(map[string]string)

	// First pass: collect all group names
	for _, group := range groups {
		groupNames[group.ID] = group.Name
	}

	// Second pass: build full paths
	for _, group := range groups {
		if group.ParentID == "" {
			groupPaths[group.ID] = group.Name
			continue
		}

		// Build path by traversing up the hierarchy
		path := group.Name
		currentID := group.ParentID
		for currentID != "" {
			if parentName, ok := groupNames[currentID]; ok {
				path = parentName + " > " + path
				// Find the parent's parent
				for _, g := range groups {
					if g.ID == currentID {
						currentID = g.ParentID
						break
					}
				}
			} else {
				break
			}
		}
		groupPaths[group.ID] = path
	}

	var userWorkspaces []UserWorkspaceAccess

	// For each workspace, check if the user has access
	for _, workspace := range workspaces {
		// Get workspace users to check permissions
		workspaceUsers, err := c.GetWorkspaceUsers(ctx, workspace.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get users for workspace %s: %w", workspace.ID, err)
		}

		// Find the user's permission in this workspace
		for _, workspaceUser := range workspaceUsers {
			if workspaceUser.UserID == userID {
				access := UserWorkspaceAccess{
					WorkspaceID:   workspace.ID,
					WorkspaceName: workspace.Name,
					Permission:    workspaceUser.Permission,
				}

				// Add group information if the workspace belongs to a group
				if workspace.GroupID != "" {
					access.GroupID = workspace.GroupID
					access.GroupName = workspace.GroupName
					if path, ok := groupPaths[workspace.GroupID]; ok {
						access.GroupPath = path
					}
				}

				userWorkspaces = append(userWorkspaces, access)
				break
			}
		}
	}

	// Sort workspaces by group path and then by workspace name
	sort.Slice(userWorkspaces, func(i, j int) bool {
		// First sort by group path (empty paths go last)
		if userWorkspaces[i].GroupPath != userWorkspaces[j].GroupPath {
			if userWorkspaces[i].GroupPath == "" {
				return false
			}
			if userWorkspaces[j].GroupPath == "" {
				return true
			}
			return userWorkspaces[i].GroupPath < userWorkspaces[j].GroupPath
		}
		// Then sort by workspace name
		return userWorkspaces[i].WorkspaceName < userWorkspaces[j].WorkspaceName
	})

	return userWorkspaces, nil
}

// WorkspaceUserStats represents the statistics of users in a workspace
type WorkspaceUserStats struct {
	TotalUsers   int `json:"totalUsers"`
	TotalEditors int `json:"totalEditors"`
	TotalAdmins  int `json:"totalAdmins"`
}

// GetWorkspaceUserStats retrieves user statistics for a specific workspace
func (c *Client) GetWorkspaceUserStats(ctx context.Context, workspaceID string) (WorkspaceUserStats, error) {
	users, err := c.GetWorkspaceUsers(ctx, workspaceID)
	if err != nil {
		return WorkspaceUserStats{}, fmt.Errorf("failed to get workspace users: %w", err)
	}

	stats := WorkspaceUserStats{
		TotalUsers: len(users),
	}

	// Count editors and admins based on their permissions
	for _, user := range users {
		switch user.Permission {
		case "write", "full":
			stats.TotalEditors++
			if user.Permission == "full" {
				stats.TotalAdmins++
			}
		}
	}

	return stats, nil
}

// refreshCacheIfNeeded checks if the cache needs to be refreshed and updates it if necessary
func (c *Client) refreshCacheIfNeeded(ctx context.Context) error {
	c.cacheMutex.RLock()
	needsRefresh := time.Since(c.cache.lastUpdate) > c.cacheTTL
	c.cacheMutex.RUnlock()

	if !needsRefresh {
		return nil
	}

	// Need to refresh cache, acquire write lock
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Double check if another goroutine already refreshed the cache
	if time.Since(c.cache.lastUpdate) <= c.cacheTTL {
		return nil
	}

	// Fetch all data in parallel
	var wg sync.WaitGroup
	var errChan = make(chan error, 3)

	// Fetch users
	wg.Add(1)
	go func() {
		defer wg.Done()
		users, err := c.fetchUsers(ctx)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch users: %w", err)
			return
		}
		c.cache.users = users
	}()

	// Fetch workspaces
	wg.Add(1)
	go func() {
		defer wg.Done()
		workspaces, err := c.fetchWorkspaces(ctx)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch workspaces: %w", err)
			return
		}
		c.cache.workspaces = workspaces
	}()

	// Fetch fields
	wg.Add(1)
	go func() {
		defer wg.Done()
		fields, err := c.fetchFields(ctx)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch fields: %w", err)
			return
		}
		c.cache.fields = fields
	}()

	// Wait for all fetches to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	c.cache.lastUpdate = time.Now()
	return nil
}

// fetchUsers makes the actual API call to get users
func (c *Client) fetchUsers(ctx context.Context) ([]User, error) {
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

// fetchWorkspaces makes the actual API call to get workspaces
func (c *Client) fetchWorkspaces(ctx context.Context) ([]Workspace, error) {
	query := WorkspaceSearchQuery{
		Sort: WorkspaceSearchSort{
			Type: "name",
			Name: WorkspaceSearchSortName{
				Direction: "asc",
			},
		},
		Archived: false,
		Filter:   nil,
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

// fetchFields makes the actual API call to get fields
func (c *Client) fetchFields(ctx context.Context) ([]FieldWithWorkspaceNames, error) {
	// Create a map of workspace IDs to names for quick lookup
	workspaceMap := make(map[string]string)
	for _, ws := range c.cache.workspaces {
		workspaceMap[ws.ID] = ws.Name
	}

	query := FieldSearchQuery{
		IsTeamField:  true,
		WorkspaceIDs: nil,
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

	fieldsWithNames := make([]FieldWithWorkspaceNames, len(searchResp.Items))
	for i, field := range searchResp.Items {
		fieldsWithNames[i] = FieldWithWorkspaceNames{
			Field: field,
		}

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
