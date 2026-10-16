package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/config"
	"localtask/internal/registry"
	"localtask/internal/ticket"
)

// TestSessionSeesProjectAddedAfterStart covers LT-11: the web UI adds a
// project to the config file while an MCP session is already running.
// The session must then find that project without a restart.
func TestSessionSeesProjectAddedAfterStart(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "taskboard.config.yaml")
	if err := os.WriteFile(configPath, []byte("projects: []\n"), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// The MCP process: its own registry, loaded before the project exists.
	mcpCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	srv := New(registry.NewRegistry(mcpCfg, nil, nil))
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })

	listed, res := callTool[listProjectsResult](t, session, "list_projects", map[string]any{})
	if res.IsError || len(listed.Projects) != 0 {
		t.Fatalf("expected no projects before the add, got %+v (%s)", listed, errorText(res))
	}

	// The web process: a separate registry over the same config file.
	webCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if _, err := registry.NewRegistry(webCfg, nil, nil).Add("LocalTask", t.TempDir(), config.StorageRepo); err != nil {
		t.Fatalf("Add: %v", err)
	}

	listed, res = callTool[listProjectsResult](t, session, "list_projects", map[string]any{})
	if res.IsError {
		t.Fatalf("list_projects: %s", errorText(res))
	}
	if len(listed.Projects) != 1 || listed.Projects[0].Name != "LocalTask" {
		t.Fatalf("expected list_projects to include LocalTask, got %+v", listed.Projects)
	}

	created, res := callTool[ticket.Ticket](t, session, "create_ticket", map[string]any{
		"project": "LocalTask",
		"title":   "First ticket",
	})
	if res.IsError {
		t.Fatalf("create_ticket: %s", errorText(res))
	}
	if !strings.HasSuffix(created.ID, "-1") {
		t.Fatalf("expected the board's first ticket id, got %s", created.ID)
	}
}
