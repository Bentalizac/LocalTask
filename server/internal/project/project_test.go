package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"localtask/internal/ticket"
)

// copyTestProject copies testdata/sample-project into a fresh temp dir so
// each test gets its own writable taskboard.json.
func copyTestProject(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "sample-project")
	dst := t.TempDir()

	data, err := os.ReadFile(filepath.Join(src, "taskboard.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "taskboard.json"), data, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return dst
}

// TestAcceptanceFlow mirrors the Phase 1 acceptance test from the spec:
// load, list, modify, save, reload, and confirm data is preserved.
func TestAcceptanceFlow(t *testing.T) {
	root := copyTestProject(t)

	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	tickets := p.ListTickets(TicketFilter{})
	if len(tickets) != 4 {
		t.Fatalf("expected 4 tickets, got %d", len(tickets))
	}

	newStatus := ticket.StatusReview
	updated, err := p.UpdateTicket("DN-42", TicketPatch{Status: &newStatus})
	if err != nil {
		t.Fatalf("UpdateTicket: %v", err)
	}
	if updated.Status != ticket.StatusReview {
		t.Fatalf("expected status review, got %s", updated.Status)
	}

	if err := p.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("reload via LoadProject: %v", err)
	}
	got, err := reloaded.GetTicket("DN-42")
	if err != nil {
		t.Fatalf("GetTicket after reload: %v", err)
	}
	if got.Status != ticket.StatusReview {
		t.Fatalf("data not preserved: expected status review, got %s", got.Status)
	}
	complete, total := got.AcceptanceCriteriaProgress()
	if complete != 2 || total != 3 {
		t.Fatalf("acceptance criteria not preserved: got %d/%d", complete, total)
	}
}

func TestCreateTicketGeneratesID(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	created, err := p.CreateTicket(ticket.Ticket{Title: "Add retry handling"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if created.ID != "DN-43" {
		t.Fatalf("expected generated id DN-43, got %s", created.ID)
	}
	if created.Type != ticket.TypeTask || created.Status != ticket.StatusBacklog || created.Priority != ticket.PriorityMedium {
		t.Fatalf("expected defaults task/backlog/medium, got %s/%s/%s", created.Type, created.Status, created.Priority)
	}

	if _, err := p.GetTicket("DN-43"); err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
}

func TestCreateTicketRequiresTitle(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if _, err := p.CreateTicket(ticket.Ticket{}); err == nil {
		t.Fatal("expected error creating ticket without title")
	}
}

func TestDeleteTicketRejectsDanglingReferences(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	// DN-42 depends on DN-38; deleting DN-38 would leave a dangling dependency.
	if err := p.DeleteTicket("DN-38"); err == nil {
		t.Fatal("expected delete to be rejected due to dangling dependency")
	}
	if _, err := p.GetTicket("DN-38"); err != nil {
		t.Fatalf("ticket should still exist after rejected delete: %v", err)
	}
}

func TestDeleteTicketOK(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	// DN-42 is a leaf: nothing depends on it or has it as a parent.
	if err := p.DeleteTicket("DN-42"); err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}
	if _, err := p.GetTicket("DN-42"); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestUpdateTicketUnknownID(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	title := "x"
	if _, err := p.UpdateTicket("NOPE-1", TicketPatch{Title: &title}); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestListTicketsFilter(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	done := ticket.StatusDone
	tickets := p.ListTickets(TicketFilter{Status: &done})
	if len(tickets) != 2 {
		t.Fatalf("expected 2 done tickets, got %d", len(tickets))
	}
}

func TestSaveRejectsStaleTaskboard(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	// Simulate an external edit landing after load but before save.
	path := filepath.Join(root, TaskboardFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading taskboard: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("rewriting taskboard: %v", err)
	}
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	title := "changed"
	if _, err := p.UpdateTicket("DN-42", TicketPatch{Title: &title}); err != nil {
		t.Fatalf("UpdateTicket: %v", err)
	}
	if err := p.Save(); !errors.Is(err, ErrStaleTaskboard) {
		t.Fatalf("expected ErrStaleTaskboard, got %v", err)
	}
}

func TestValidateRejectsInvalidStatus(t *testing.T) {
	root := t.TempDir()
	tb := Taskboard{
		Version: 1,
		Project: "Test",
		Tickets: []ticket.Ticket{
			{ID: "T-1", Title: "x", Type: ticket.TypeTask, Status: "not-a-status", Priority: ticket.PriorityLow},
		},
	}
	if err := validateTaskboard(&tb, root); err == nil {
		t.Fatal("expected validation error for invalid status")
	}
}

func TestValidateRejectsPathEscapingRoot(t *testing.T) {
	root := t.TempDir()
	tb := Taskboard{
		Version: 1,
		Project: "Test",
		Tickets: []ticket.Ticket{
			{
				ID: "T-1", Title: "x", Type: ticket.TypeTask,
				Status: ticket.StatusBacklog, Priority: ticket.PriorityLow,
				RelatedDocs: []string{"../../etc/passwd"},
			},
		},
	}
	if err := validateTaskboard(&tb, root); err == nil {
		t.Fatal("expected validation error for path escaping root")
	}
}

func TestValidateRejectsDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	tb := Taskboard{
		Version: 1,
		Project: "Test",
		Tickets: []ticket.Ticket{
			{ID: "T-1", Title: "a", Type: ticket.TypeTask, Status: ticket.StatusBacklog, Priority: ticket.PriorityLow},
			{ID: "T-1", Title: "b", Type: ticket.TypeTask, Status: ticket.StatusBacklog, Priority: ticket.PriorityLow},
		},
	}
	if err := validateTaskboard(&tb, root); err == nil {
		t.Fatal("expected validation error for duplicate ids")
	}
}
