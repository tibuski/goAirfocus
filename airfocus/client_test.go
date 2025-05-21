package airfocus

import (
	"os"
	"testing"
)

func TestGetWorkspaceIDByName(t *testing.T) {
	apiKey := os.Getenv("AIRFOCUS_API_KEY")
	if apiKey == "" {
		t.Skip("AIRFOCUS_API_KEY environment variable not set")
	}

	client := NewClient(apiKey)

	// Test with a known workspace name
	workspaceName := "start from scratch"
	id, err := client.GetWorkspaceIDByName(workspaceName)
	if err != nil {
		t.Errorf("GetWorkspaceIDByName failed: %v", err)
	}
	if id == "" {
		t.Error("Expected a workspace ID, got empty string")
	}
	t.Logf("Found workspace ID: %s", id)
}

func TestListWorkspaces(t *testing.T) {
	apiKey := os.Getenv("AIRFOCUS_API_KEY")
	if apiKey == "" {
		t.Skip("AIRFOCUS_API_KEY not set")
	}
	client := NewClient(apiKey)
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces failed: %v", err)
	}
	if len(workspaces) == 0 {
		t.Error("No workspaces returned")
	}
	for _, ws := range workspaces {
		t.Logf("Workspace: %s (%s)", ws.Name, ws.ID)
	}
}
