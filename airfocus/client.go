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

// Client represents an Airfocus API client
type Client struct {
	apiKey     string       // The API key used for authentication
	httpClient *http.Client // HTTP client for making requests

	// Cache fields for storing frequently accessed data
	cache struct {
		users           []User                    // Cached list of users
		workspaces      []Workspace               // Cached list of workspaces
		fields          []FieldWithWorkspaceNames // Cached list of fields
		workspaceGroups []WorkspaceGroup          // Cached list of workspace groups
		lastUpdate      time.Time                 // Timestamp of last cache update
	}
	cacheMutex sync.RWMutex  // Mutex for thread-safe cache access
	cacheTTL   time.Duration // Time-to-live for cached data
}

// NewClient creates a new Airfocus API client with the given API key
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
	Type string                  `json:"type"` // Type of sorting
	Name WorkspaceSearchSortName `json:"name"` // Name sorting options
}

// WorkspaceSearchFilter represents the filter options for workspace search
type WorkspaceSearchFilter struct {
	Type          string `json:"type"`          // Type of filter
	Mode          string `json:"mode"`          // Filter mode (e.g., "contain")
	Text          string `json:"text"`          // Filter text
	CaseSensitive bool   `json:"caseSensitive"` // Whether the filter is case-sensitive
}

// WorkspaceSearchQuery represents the query parameters for workspace search
type WorkspaceSearchQuery struct {
	Sort     WorkspaceSearchSort    `json:"sort"`             // Sorting options
	Archived bool                   `json:"archived"`         // Whether to include archived workspaces
	Filter   *WorkspaceSearchFilter `json:"filter,omitempty"` // Optional filter
}

// --- End Workspace Search Query Structs ---

// WorkspaceGroup represents a group that can contain workspaces and other groups
type WorkspaceGroup struct {
	ID                string `json:"id"`                 // Unique identifier for the group
	Name              string `json:"name"`               // Group name
	ParentID          string `json:"parentId,omitempty"` // ID of the parent group, if any
	Order             int    `json:"order"`              // Display order
	DefaultPermission string `json:"defaultPermission"`  // Default permission for the group
	CreatedAt         string `json:"createdAt"`          // Creation timestamp
	LastUpdatedAt     string `json:"lastUpdatedAt"`      // Last update timestamp
	TeamID            string `json:"teamId"`             // ID of the team this group belongs to
	Embedded          struct {
		Workspaces  []Workspace       `json:"workspaces,omitempty"`  // List of workspaces in this group
		Permissions map[string]string `json:"permissions,omitempty"` // Map of user IDs to their permissions for the group
	} `json:"_embedded,omitempty"`
	CurrentPermission Permission `json:"currentPermission,omitempty"` // Current user's permission for this group
}

// WorkspaceGroupSearchQuery represents the query parameters for workspace group search
type WorkspaceGroupSearchQuery struct {
	Sort struct {
		Type      string `json:"type"`      // Type of sorting
		Direction string `json:"direction"` // Sort direction ("asc" or "desc")
	} `json:"sort"`
}

// WorkspaceGroupResponse represents the response from the workspace group search API
type WorkspaceGroupResponse struct {
	Items      []WorkspaceGroup `json:"items"`      // List of workspace groups
	TotalItems int              `json:"totalItems"` // Total number of items
}

// ListWorkspaceGroups retrieves all workspace groups and their hierarchy
func (c *Client) ListWorkspaceGroups(ctx context.Context) ([]WorkspaceGroup, error) {
	// First check if we have cached groups that are still valid
	if err := c.RefreshCacheIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh cache: %w", err)
	}

	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	// Return a copy of the cached groups
	groups := make([]WorkspaceGroup, len(c.cache.workspaceGroups))
	copy(groups, c.cache.workspaceGroups)
	return groups, nil
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

// Workspace represents an Airfocus workspace
type Workspace struct {
	ID          string `json:"id"`    // Unique identifier for the workspace
	Name        string `json:"name"`  // Workspace name
	Alias       string `json:"alias"` // Workspace alias
	Description struct {
		Blocks []interface{} `json:"blocks"` // Description blocks
	} `json:"description"`
	ItemType      string `json:"itemType"`      // Type of items in the workspace
	ItemColor     string `json:"itemColor"`     // Color for items
	ProgressMode  string `json:"progressMode"`  // Progress tracking mode
	Archived      bool   `json:"archived"`      // Whether the workspace is archived
	CreatedAt     string `json:"createdAt"`     // Creation timestamp
	LastUpdatedAt string `json:"lastUpdatedAt"` // Last update timestamp
	Metadata      struct {
		Version    string `json:"version"`    // Workspace version
		Duplicated bool   `json:"duplicated"` // Whether this is a duplicated workspace
	} `json:"metadata"`
	Namespace string `json:"namespace"` // Workspace namespace
	Order     int    `json:"order"`     // Display order
	TeamID    string `json:"teamId"`    // ID of the team this workspace belongs to
	Embedded  struct {
		Permissions map[string]string `json:"permissions"` // Map of user IDs to their permissions
	} `json:"_embedded,omitempty"`
	GroupID           string     `json:"groupId,omitempty"`           // ID of the group this workspace belongs to
	GroupName         string     `json:"groupName,omitempty"`         // Name of the group this workspace belongs to
	CurrentPermission Permission `json:"currentPermission,omitempty"` // Current user's permission for this workspace
}

// WorkspaceResponse represents the response from the workspace search API
type WorkspaceResponse struct {
	Items      []Workspace `json:"items"`      // List of workspaces
	TotalItems int         `json:"totalItems"` // Total number of items
}

// WorkspaceResult represents the result of a workspace search
type WorkspaceResult struct {
	ID    string // Workspace ID
	Alias string // Workspace alias
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

// FieldSearchQuery represents the query parameters for field search
type FieldSearchQuery struct {
	IsTeamField  bool     `json:"isTeamField,omitempty"`  // Whether to search for team fields
	WorkspaceIDs []string `json:"workspaceIds,omitempty"` // List of workspace IDs to search in
}

// Field represents an Airfocus field
type Field struct {
	ID          string `json:"id"`          // Unique identifier for the field
	Name        string `json:"name"`        // Field name
	Description string `json:"description"` // Field description
	Type        string `json:"type"`        // Field type
	CreatedAt   string `json:"createdAt"`   // Creation timestamp
	UpdatedAt   string `json:"updatedAt"`   // Last update timestamp
	IsTeamField bool   `json:"isTeamField"` // Whether this is a team-wide field
	Embedded    struct {
		Workspaces []struct {
			WorkspaceID string `json:"workspaceId"` // ID of the workspace this field belongs to
			Order       int    `json:"order"`       // Display order in the workspace
		} `json:"workspaces"`
		AllWorkspaceIDs []string `json:"allWorkspaceIds"` // List of all workspace IDs this field belongs to
	} `json:"_embedded,omitempty"`
}

// GetWorkspaceCount returns the number of workspaces this field belongs to
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
	Items      []Field `json:"items"`      // List of fields
	TotalItems int     `json:"totalItems"` // Total number of items
}

// GetWorkspaceByID retrieves a workspace by its ID
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

// FieldWithWorkspaceNames extends Field with a list of workspace names
type FieldWithWorkspaceNames struct {
	Field
	WorkspaceNames []string // List of workspace names this field belongs to
}

// ListFields retrieves all fields with their workspace names
func (c *Client) ListFields(ctx context.Context) ([]FieldWithWorkspaceNames, error) {
	if err := c.RefreshCacheIfNeeded(ctx); err != nil {
		return nil, err
	}

	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	// Return a copy of the cached fields
	fields := make([]FieldWithWorkspaceNames, len(c.cache.fields))
	copy(fields, c.cache.fields)
	return fields, nil
}

// ListWorkspaces retrieves all workspaces
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	if err := c.RefreshCacheIfNeeded(ctx); err != nil {
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

	// Create a map of group IDs to their names for quick lookup
	groupNameMap := make(map[string]string)
	for _, group := range groups {
		groupNameMap[group.ID] = group.Name
	}

	// First, process all groups and their embedded workspaces
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

	// Process all workspaces and add group information
	var workspaces []Workspace
	for _, ws := range c.cache.workspaces {
		// Create a copy of the workspace
		workspace := ws

		// First check if the workspace already has a GroupID
		if workspace.GroupID != "" {
			// If it has a GroupID but no GroupName, look up the group name
			if workspace.GroupName == "" {
				if groupName, ok := groupNameMap[workspace.GroupID]; ok {
					workspace.GroupName = groupName
				}
			}
		} else {
			// If no GroupID, check if it's in the workspaceGroupMap
			if groupInfo, ok := workspaceGroupMap[ws.ID]; ok {
				workspace.GroupID = groupInfo.GroupID
				workspace.GroupName = groupInfo.GroupName
			}
		}

		workspaces = append(workspaces, workspace)
	}

	return workspaces, nil
}

// User represents an Airfocus user
type User struct {
	UserID        string     `json:"userId"`        // Unique identifier for the user
	TeamID        string     `json:"teamId"`        // ID of the team this user belongs to
	FullName      string     `json:"fullName"`      // User's full name
	Email         string     `json:"email"`         // User's email address
	Role          string     `json:"role"`          // User's role (e.g., "admin", "contributor", "editor")
	State         *UserState `json:"state"`         // User's state (pending, unseated)
	IsTeamCreator bool       `json:"isTeamCreator"` // Whether this user created the team
	Disabled      bool       `json:"disabled"`      // Whether the user is disabled
	EmailVerified bool       `json:"emailVerified"` // Whether the user's email is verified
	CreatedAt     string     `json:"createdAt"`     // When the user was created
	UpdatedAt     string     `json:"updatedAt"`     // When the user was last updated
}

// ListUsers retrieves all users
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	if err := c.RefreshCacheIfNeeded(ctx); err != nil {
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

// WorkspaceUser represents a user with their permission level in a workspace
type WorkspaceUser struct {
	UserID     string `json:"userId"`     // Unique identifier for the user
	FullName   string `json:"fullName"`   // User's full name
	Email      string `json:"email"`      // User's email address
	Permission string `json:"permission"` // User's permission level ("read", "write", "full", "comment")
}

// GetWorkspaceUsers retrieves all users with their permissions for a specific workspace
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

// UserState represents the state of a user in the system
type UserState struct {
	Pending  bool `json:"pending"`  // Whether the user's invitation is pending
	Unseated bool `json:"unseated"` // Whether the user is unseated
}

// UserWithRole represents a user with their role and state
type UserWithRole struct {
	UserID        string     `json:"userId"`        // Unique identifier for the user
	TeamID        string     `json:"teamId"`        // ID of the team this user belongs to
	FullName      string     `json:"fullName"`      // User's full name
	Email         string     `json:"email"`         // User's email address
	Role          string     `json:"role"`          // User's role
	State         *UserState `json:"state"`         // User's state
	IsTeamCreator bool       `json:"isTeamCreator"` // Whether this user created the team
	Disabled      bool       `json:"disabled"`      // Whether the user is disabled
	EmailVerified bool       `json:"emailVerified"` // Whether the user's email is verified
	CreatedAt     string     `json:"createdAt"`     // When the user was created
	UpdatedAt     string     `json:"updatedAt"`     // When the user was last updated
}

// UserWorkspaceAccess represents a user's access to a workspace
type UserWorkspaceAccess struct {
	WorkspaceID   string `json:"workspaceId"`         // ID of the workspace
	WorkspaceName string `json:"workspaceName"`       // Name of the workspace
	Permission    string `json:"permission"`          // User's permission level
	GroupID       string `json:"groupId,omitempty"`   // ID of the group this workspace belongs to
	GroupName     string `json:"groupName,omitempty"` // Name of the group this workspace belongs to
	GroupPath     string `json:"groupPath,omitempty"` // Full path to the group (e.g., "Parent Group > Child Group")
}

// FormatUsersWithRoles retrieves and formats all users with their roles
func (c *Client) FormatUsersWithRoles(ctx context.Context) ([]UserWithRole, error) {
	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	formattedUsers := make([]UserWithRole, len(users))
	for i, user := range users {
		formattedUsers[i] = UserWithRole{
			UserID:        user.UserID,
			TeamID:        user.TeamID,
			FullName:      user.FullName,
			Email:         user.Email,
			Role:          user.Role,
			State:         user.State,
			IsTeamCreator: user.IsTeamCreator,
			Disabled:      user.Disabled,
			EmailVerified: user.EmailVerified,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
		}
	}

	return formattedUsers, nil
}

// GetUserWorkspaces retrieves all workspaces a selected user has access to
func (c *Client) GetUserWorkspaces(ctx context.Context, userID string) ([]UserWorkspaceAccess, error) {
	if err := c.RefreshCacheIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh workspace cache before getting user workspaces: %w", err)
	}

	// Get all workspaces with their group information
	workspaces, err := c.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Get workspace groups for path construction
	groups, err := c.ListWorkspaceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace groups: %w", err)
	}

	// Create a map of group IDs to their names for quick lookup
	groupMap := make(map[string]string)
	for _, group := range groups {
		groupMap[group.ID] = group.Name
	}

	// Create a map of group IDs to their parent IDs for path construction
	groupParentMap := make(map[string]string)
	for _, group := range groups {
		if group.ParentID != "" {
			groupParentMap[group.ID] = group.ParentID
		}
	}

	// Helper function to get the full group path
	getFullGroupPath := func(groupID string) string {
		if groupID == "" {
			return ""
		}

		var path []string
		currentID := groupID
		for currentID != "" {
			if name, ok := groupMap[currentID]; ok {
				path = append([]string{name}, path...)
				currentID = groupParentMap[currentID]
			} else {
				break
			}
		}

		return strings.Join(path, " > ")
	}

	var userWorkspaces []UserWorkspaceAccess

	// Iterate through all workspaces
	for _, workspace := range workspaces {
		// Check if the user ID exists in the workspace's permissions map
		if permission, ok := workspace.Embedded.Permissions[userID]; ok {
			// Get the full group path for this workspace
			groupPath := getFullGroupPath(workspace.GroupID)

			// User has a specific permission for this workspace
			userWorkspaces = append(userWorkspaces, UserWorkspaceAccess{
				WorkspaceID:   workspace.ID,
				WorkspaceName: workspace.Name,
				Permission:    permission,
				GroupID:       workspace.GroupID,
				GroupName:     workspace.GroupName,
				GroupPath:     groupPath,
			})
		}
	}

	return userWorkspaces, nil
}

// WorkspaceUserStats represents statistics about users in a workspace
type WorkspaceUserStats struct {
	TotalUsers   int `json:"totalUsers"`   // Total number of users
	TotalEditors int `json:"totalEditors"` // Total number of editors
	TotalAdmins  int `json:"totalAdmins"`  // Total number of administrators
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

// fetchWorkspaceGroups retrieves and caches the list of workspace groups
func (c *Client) fetchWorkspaceGroups(ctx context.Context) ([]WorkspaceGroup, error) {
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

// RefreshCacheIfNeeded checks if the cache is stale and refreshes it if necessary.
func (c *Client) RefreshCacheIfNeeded(ctx context.Context) error {
	c.cacheMutex.RLock()
	needsRefresh := time.Since(c.cache.lastUpdate) > c.cacheTTL
	c.cacheMutex.RUnlock()

	if !needsRefresh {
		return nil
	}

	// Acquire a write lock to update the cache
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Double check if another goroutine already refreshed the cache
	if time.Since(c.cache.lastUpdate) <= c.cacheTTL {
		return nil
	}

	// Fetch all data in parallel
	var wg sync.WaitGroup
	var errChan = make(chan error, 4) // Increased to 4 for workspace groups

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

	// Fetch workspace groups
	wg.Add(1)
	go func() {
		defer wg.Done()
		groups, err := c.fetchWorkspaceGroups(ctx)
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch workspace groups: %w", err)
			return
		}
		c.cache.workspaceGroups = groups
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

// fetchUsers retrieves and caches the list of users
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

// fetchWorkspaces retrieves and caches the list of workspaces
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

// fetchFields retrieves and caches the list of fields
func (c *Client) fetchFields(ctx context.Context) ([]FieldWithWorkspaceNames, error) {
	// Create a map of workspace IDs to names for quick lookup
	workspaceMap := make(map[string]string)

	// Check if we have workspaces in cache, if not fetch them directly
	if len(c.cache.workspaces) == 0 {
		workspaces, err := c.fetchWorkspaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch workspaces for field mapping: %w", err)
		}
		for _, ws := range workspaces {
			workspaceMap[ws.ID] = ws.Name
		}
	} else {
		for _, ws := range c.cache.workspaces {
			workspaceMap[ws.ID] = ws.Name
		}
	}

	query := FieldSearchQuery{
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

// Permission represents the access level to a workspace or workspace group.
type Permission string

const (
	PermissionRead    Permission = "read"
	PermissionComment Permission = "comment"
	PermissionWrite   Permission = "write"
	PermissionFull    Permission = "full"
)

// GetUser fetches a single user by their ID.
func (c *Client) GetUser(ctx context.Context, userID string) (User, error) {
	// Ensure user cache is up-to-date
	if err := c.RefreshCacheIfNeeded(ctx); err != nil {
		return User{}, fmt.Errorf("failed to refresh cache for users: %w", err)
	}

	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	for _, user := range c.cache.users {
		if user.UserID == userID {
			return user, nil
		}
	}
	return User{}, fmt.Errorf("user with ID %s not found", userID)
}

// GetUserGroupAccess retrieves workspace groups the user has access to,
// populating their current permission and embedding relevant workspaces with user permissions.
// This function properly handles hierarchical group permissions.
func (c *Client) GetUserGroupAccess(ctx context.Context, userID string) ([]WorkspaceGroup, error) {
	// Fetch all workspace groups
	groups, err := c.ListWorkspaceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace groups: %w", err)
	}

	// Fetch all workspaces once
	allWorkspaces, err := c.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all workspaces: %w", err)
	}

	// Create maps for quick lookup
	groupMap := make(map[string]WorkspaceGroup)
	groupParentMap := make(map[string]string)
	workspacesByGroupID := make(map[string][]Workspace)

	// Build lookup maps
	for _, group := range groups {
		groupMap[group.ID] = group
		if group.ParentID != "" {
			groupParentMap[group.ID] = group.ParentID
		}
	}

	// Group workspaces by their GroupID
	for _, ws := range allWorkspaces {
		workspacesByGroupID[ws.GroupID] = append(workspacesByGroupID[ws.GroupID], ws)
	}

	// Helper function to get the highest permission from a set of permissions
	getHighestPermission := func(currentPerm Permission, newPerm Permission) Permission {
		pOrder := map[Permission]int{
			PermissionRead:    1,
			PermissionComment: 2,
			PermissionWrite:   3,
			PermissionFull:    4,
		}
		if pOrder[newPerm] > pOrder[currentPerm] {
			return newPerm
		}
		return currentPerm
	}

	// Helper function to calculate effective permission for a group
	calculateGroupPermission := func(groupID string) Permission {
		effectivePermission := PermissionRead // Start with lowest permission

		// Traverse up the group hierarchy
		currentGroupID := groupID
		for currentGroupID != "" {
			if group, ok := groupMap[currentGroupID]; ok {
				// Check explicit permission for the user in this group
				if permStr, ok := group.Embedded.Permissions[userID]; ok {
					effectivePermission = getHighestPermission(effectivePermission, Permission(permStr))
				}
				// Check default team permission for this group
				if group.DefaultPermission != "" {
					effectivePermission = getHighestPermission(effectivePermission, Permission(group.DefaultPermission))
				}
				currentGroupID = groupParentMap[currentGroupID] // Move up the hierarchy
			} else {
				break // Group not found, stop traversing
			}
		}

		return effectivePermission
	}

	// Helper function to calculate effective permission for a workspace
	calculateWorkspacePermission := func(workspace Workspace) Permission {
		effectivePermission := PermissionRead // Start with lowest permission

		// 1. Check explicit permission for the specific user in the workspace settings
		if permStr, ok := workspace.Embedded.Permissions[userID]; ok {
			effectivePermission = getHighestPermission(effectivePermission, Permission(permStr))
		}

		// 2. Check group hierarchy permissions (if workspace belongs to a group)
		if workspace.GroupID != "" {
			groupPermission := calculateGroupPermission(workspace.GroupID)
			effectivePermission = getHighestPermission(effectivePermission, groupPermission)
		}

		return effectivePermission
	}

	var userAccessibleGroups []WorkspaceGroup

	// Process each group
	for _, group := range groups {
		// Calculate the user's effective permission for this group
		groupPermission := calculateGroupPermission(group.ID)

		// Only include groups the user has some permission for
		if groupPermission != "" {
			group.CurrentPermission = groupPermission

			// Populate workspaces for this group
			if wsList, ok := workspacesByGroupID[group.ID]; ok {
				var workspacesWithUserPermissions []Workspace
				for _, ws := range wsList {
					ws.CurrentPermission = calculateWorkspacePermission(ws)
					workspacesWithUserPermissions = append(workspacesWithUserPermissions, ws)
				}
				group.Embedded.Workspaces = workspacesWithUserPermissions
			}

			userAccessibleGroups = append(userAccessibleGroups, group)
		}
	}

	return userAccessibleGroups, nil
}
