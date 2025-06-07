# Airfocus API Client

A Go client for the Airfocus API that provides easy access to workspace, user, and field management functionality.

## Features

- **User Management**
  - List all users with their roles
  - Get user workspace access with hierarchical group display
  - View user permissions across workspaces

- **Workspace Management**
  - List all workspaces
  - Search workspaces by name
  - Get workspace details including permissions
  - View workspace hierarchy with groups

- **Field Management**
  - List all fields
  - Search fields by workspace
  - View field usage across workspaces

## Usage

### User Workspace Access

The client provides a hierarchical view of user workspace access, showing:

- Workspaces grouped by their workspace groups
- Permission levels for each workspace (full, write, read, comment)
- Clear visual hierarchy with group paths
- Color-coded permission badges

Example:
```
Group A > Subgroup 1
  - Workspace 1 [full]
  - Workspace 2 [write]

Group B
  - Workspace 3 [read]
  - Workspace 4 [comment]

Ungrouped Workspaces
  - Workspace 5 [full]
```

### API Key

All requests require an Airfocus API key. You can obtain one from your Airfocus account settings.

## Installation

```bash
go get github.com/yourusername/airfocus
```

## Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/yourusername/airfocus"
)

func main() {
    client := airfocus.NewClient("your-api-key")
    ctx := context.Background()

    // Get user workspace access
    workspaces, err := client.GetUserWorkspaces(ctx, "user-id")
    if err != nil {
        log.Fatal(err)
    }

    // Workspaces are returned with group hierarchy information
    for _, ws := range workspaces {
        fmt.Printf("Workspace: %s\n", ws.WorkspaceName)
        fmt.Printf("Group: %s\n", ws.GroupPath)
        fmt.Printf("Permission: %s\n", ws.Permission)
    }
}
```

## License

MIT License
