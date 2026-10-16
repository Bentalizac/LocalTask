package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

	p, err := project.LoadProject("DanceNote", root, root)
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

// TestToolOutputSchemasAreObjectRooted guards against the whole bug class
// reported in production: a Go slice's inferred JSON Schema has type
// ["null","array"] at the root (nilable), which some MCP clients' output
// validation rejects even though the MCP spec itself allows it. Every tool
// here must return an object at the top level (see listProjectsResult /
// listTicketsResult for how list-shaped tools do this) — this test fails
// for any future tool that regresses to a bare array/slice Out type.
func TestToolOutputSchemasAreObjectRooted(t *testing.T) {
	session, _ := newTestSession(t)
	tools, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("expected at least one tool")
	}
	for _, tool := range tools.Tools {
		if tool.OutputSchema == nil {
			continue
		}
		data, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatalf("%s: marshal output schema: %v", tool.Name, err)
		}
		var schema struct {
			Type any `json:"type"`
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("%s: decode output schema: %v", tool.Name, err)
		}
		if typeStr, ok := schema.Type.(string); !ok || typeStr != "object" {
			t.Errorf("%s: output schema root type is %v (%T), want the plain string \"object\"", tool.Name, schema.Type, schema.Type)
		}
	}
}

func TestListProjectsTool(t *testing.T) {
	session, _ := newTestSession(t)
	out, res := callTool[listProjectsResult](t, session, "list_projects", map[string]any{})
	if res.IsError {
		t.Fatalf("unexpected error: %s", errorText(res))
	}
	if len(out.Projects) != 1 || out.Projects[0].Name != "DanceNote" || out.Projects[0].Metrics.Total != 4 {
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

func TestListTicketsTool(t *testing.T) {
	session, _ := newTestSession(t)
	out, res := callTool[listTicketsResult](t, session, "list_tickets", map[string]any{
		"project": "DanceNote",
		"status":  "done",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", errorText(res))
	}
	if len(out.Tickets) != 2 {
		t.Fatalf("expected 2 done tickets, got %d: %+v", len(out.Tickets), out.Tickets)
	}
}

func TestSearchTicketsTool(t *testing.T) {
	session, _ := newTestSession(t)
	out, res := callTool[listTicketsResult](t, session, "search_tickets", map[string]any{
		"project": "DanceNote",
		"query":   "worker",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", errorText(res))
	}
	if len(out.Tickets) != 1 || out.Tickets[0].ID != "DN-42" {
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

	reloaded, err := project.LoadProject("DanceNote", root, root)
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

func TestClaimAndNoteWorkflow(t *testing.T) {
	session, _ := newTestSession(t)
	created, res := callTool[ticket.Ticket](t, session, "create_ticket", map[string]any{
		"project": "DanceNote", "title": "agent work", "status": "ready", "agent": "planner",
	})
	if res.IsError {
		t.Fatalf("create_ticket: %s", errorText(res))
	}

	ready, res := callTool[listTicketsResult](t, session, "next_ready_tickets", map[string]any{"project": "DanceNote"})
	if res.IsError || len(ready.Tickets) == 0 || ready.Tickets[0].ID != created.ID {
		t.Fatalf("next_ready_tickets: %+v %s", ready, errorText(res))
	}

	claimed, res := callTool[ticket.Ticket](t, session, "claim_ticket", map[string]any{
		"project": "DanceNote", "ticket_id": created.ID, "agent": "worker-1",
	})
	if res.IsError || claimed.Assignee != "worker-1" || claimed.Status != ticket.StatusInProgress {
		t.Fatalf("claim_ticket: %+v %s", claimed, errorText(res))
	}

	_, res = callTool[ticket.Ticket](t, session, "claim_ticket", map[string]any{
		"project": "DanceNote", "ticket_id": created.ID, "agent": "worker-2",
	})
	if !res.IsError || !strings.Contains(errorText(res), "worker-1") {
		t.Fatalf("second claim should fail naming the holder, got %q", errorText(res))
	}

	noted, res := callTool[ticket.Ticket](t, session, "update_ticket", map[string]any{
		"project": "DanceNote", "ticket_id": created.ID, "agent": "worker-1", "note": "halfway there",
	})
	if res.IsError {
		t.Fatalf("update_ticket: %s", errorText(res))
	}
	last := noted.Activity[len(noted.Activity)-1]
	if last.Kind != ticket.ActivityNote || last.Author != "worker-1" || last.Text != "halfway there" {
		t.Fatalf("unexpected last activity: %+v", last)
	}
}
