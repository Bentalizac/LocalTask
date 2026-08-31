package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/project"
	"localtask/internal/ticket"
)

// newTestSession loads a fresh copy of testdata/sample-project as
// "DanceNote", connects an in-memory MCP client to the server, and returns
// the session plus the project's root dir (for reload-and-verify
// assertions). This exercises the real MCP wire protocol, not just Go
// function calls, per the spec's "test MCP independently from the browser".
func newTestSession(t *testing.T) (*mcp.ClientSession, string) {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "sample-project")
	root := t.TempDir()

	data, err := os.ReadFile(filepath.Join(src, "taskboard.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "taskboard.json"), data, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "video-processing.md"), []byte("# Video Processing\n"), 0o644); err != nil {
		t.Fatalf("writing doc: %v", err)
	}

	p, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	srv := New(map[string]*project.Project{"DanceNote": p})

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

	return session, root
}

func callTool[Out any](t *testing.T, session *mcp.ClientSession, name string, args any) (Out, *mcp.CallToolResult) {
	t.Helper()
	var zero Out
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		return zero, res
	}
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var out Out
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	return out, res
}

func errorText(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestListProjectsTool(t *testing.T) {
	session, _ := newTestSession(t)
	out, res := callTool[[]projectInfo](t, session, "list_projects", map[string]any{})
	if res.IsError {
		t.Fatalf("unexpected error: %s", errorText(res))
	}
	if len(out) != 1 || out[0].Name != "DanceNote" || out[0].Metrics.Total != 4 {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestGetProjectContextTool(t *testing.T) {
	session, _ := newTestSession(t)
	out, res := callTool[projectContext](t, session, "get_project_context", map[string]any{"project": "DanceNote"})
	if res.IsError {
		t.Fatalf("unexpected error: %s", errorText(res))
	}
	if len(out.ActiveTickets) != 2 {
		t.Fatalf("expected 2 active (in_progress) tickets, got %d: %+v", len(out.ActiveTickets), out.ActiveTickets)
	}
	if len(out.BlockedTickets) != 0 {
		t.Fatalf("expected 0 blocked tickets, got %d", len(out.BlockedTickets))
	}
	if len(out.Documentation) != 1 || out.Documentation[0] != "docs/video-processing.md" {
		t.Fatalf("unexpected documentation: %+v", out.Documentation)
	}
	if out.Metrics.Total != 4 || out.Metrics.CompletionPercent != 50 {
		t.Fatalf("unexpected metrics: %+v", out.Metrics)
	}
	if out.Git.IsRepo {
		t.Fatalf("expected is_repo=false for a plain temp dir, got %+v", out.Git)
	}
}

func TestGetProjectUnknown(t *testing.T) {
	session, _ := newTestSession(t)
	_, res := callTool[projectInfo](t, session, "get_project", map[string]any{"project": "Nope"})
	if !res.IsError {
		t.Fatal("expected error for unknown project")
	}
}

func TestSearchTicketsTool(t *testing.T) {
	session, _ := newTestSession(t)
	out, res := callTool[[]ticket.Ticket](t, session, "search_tickets", map[string]any{
		"project": "DanceNote",
		"query":   "worker",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", errorText(res))
	}
	if len(out) != 1 || out[0].ID != "DN-42" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestCreateAndUpdateTicketToolsPersist(t *testing.T) {
	session, root := newTestSession(t)

	created, res := callTool[ticket.Ticket](t, session, "create_ticket", map[string]any{
		"project": "DanceNote",
		"title":   "Add retry handling",
	})
	if res.IsError {
		t.Fatalf("create_ticket: %s", errorText(res))
	}
	if created.ID != "DN-43" {
		t.Fatalf("expected generated id DN-43, got %s", created.ID)
	}

	updated, res := callTool[ticket.Ticket](t, session, "update_ticket", map[string]any{
		"project":   "DanceNote",
		"ticket_id": "DN-43",
		"status":    "ready",
	})
	if res.IsError {
		t.Fatalf("update_ticket: %s", errorText(res))
	}
	if updated.Status != ticket.StatusReady {
		t.Fatalf("expected status ready, got %s", updated.Status)
	}

	reloaded, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, err := reloaded.GetTicket("DN-43")
	if err != nil {
		t.Fatalf("GetTicket after reload: %v", err)
	}
	if got.Status != ticket.StatusReady {
		t.Fatalf("update not persisted: got status %s", got.Status)
	}
}

func TestUpdateTicketInvalidStatus(t *testing.T) {
	session, _ := newTestSession(t)
	_, res := callTool[ticket.Ticket](t, session, "update_ticket", map[string]any{
		"project":   "DanceNote",
		"ticket_id": "DN-42",
		"status":    "not-a-status",
	})
	if !res.IsError {
		t.Fatal("expected error for invalid status")
	}
}

func TestReadProjectResource(t *testing.T) {
	session, _ := newTestSession(t)
	res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "project://DanceNote"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(res.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(res.Contents))
	}
	var info projectInfo
	if err := json.Unmarshal([]byte(res.Contents[0].Text), &info); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if info.Name != "DanceNote" {
		t.Fatalf("unexpected project info: %+v", info)
	}
}

func TestReadTicketResource(t *testing.T) {
	session, _ := newTestSession(t)
	res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "project://DanceNote/ticket/DN-42"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	var tk ticket.Ticket
	if err := json.Unmarshal([]byte(res.Contents[0].Text), &tk); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tk.ID != "DN-42" {
		t.Fatalf("unexpected ticket: %+v", tk)
	}
}

func TestReadDocResource(t *testing.T) {
	session, _ := newTestSession(t)
	res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "project://DanceNote/doc/docs/video-processing.md"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if res.Contents[0].MIMEType != "text/markdown" {
		t.Fatalf("expected text/markdown, got %s", res.Contents[0].MIMEType)
	}
	if res.Contents[0].Text == "" {
		t.Fatal("expected non-empty doc content")
	}
}

func TestReadResourceUnknownProject(t *testing.T) {
	session, _ := newTestSession(t)
	_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "project://Nope"})
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
}
