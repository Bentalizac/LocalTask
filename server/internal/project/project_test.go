package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

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

	p, err := LoadProject("DanceNote", root, root)
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

	reloaded, err := LoadProject("DanceNote", root, root)
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
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	created, err := p.CreateTicket(ticket.Ticket{Title: "Add retry handling"}, "")
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
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if _, err := p.CreateTicket(ticket.Ticket{}, ""); err == nil {
		t.Fatal("expected error creating ticket without title")
	}
}

func TestDeleteTicketRejectsDanglingReferences(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root, root)
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
	p, err := LoadProject("DanceNote", root, root)
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
	p, err := LoadProject("DanceNote", root, root)
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
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	done := ticket.StatusDone
	tickets := p.ListTickets(TicketFilter{Status: &done})
	if len(tickets) != 2 {
		t.Fatalf("expected 2 done tickets, got %d", len(tickets))
	}
}

// TestConcurrentWritersDontLoseUpdates simulates two separate processes
// (e.g. two agents' MCP servers) holding independently loaded copies of
// the same project. Each writes a different ticket without ever reloading;
// both changes must survive.
func TestConcurrentWritersDontLoseUpdates(t *testing.T) {
	root := copyTestProject(t)
	a, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject a: %v", err)
	}
	b, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject b: %v", err)
	}

	titleA, titleB := "changed by a", "changed by b"
	if _, err := a.UpdateTicket("DN-42", TicketPatch{Title: &titleA}); err != nil {
		t.Fatalf("a.UpdateTicket: %v", err)
	}
	if _, err := b.UpdateTicket("DN-38", TicketPatch{Title: &titleB}); err != nil {
		t.Fatalf("b.UpdateTicket: %v", err)
	}

	final, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got, _ := final.GetTicket("DN-42"); got.Title != titleA {
		t.Fatalf("a's update lost: DN-42 title %q", got.Title)
	}
	if got, _ := final.GetTicket("DN-38"); got.Title != titleB {
		t.Fatalf("b's update lost: DN-38 title %q", got.Title)
	}
}

// TestParallelCreatesGetUniqueIDs hammers one taskboard from many
// independently loaded Projects at once; every create must land with a
// distinct ID.
func TestParallelCreatesGetUniqueIDs(t *testing.T) {
	root := copyTestProject(t)
	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := LoadProject("DanceNote", root, root)
			if err != nil {
				errs <- err
				return
			}
			_, err = p.CreateTicket(ticket.Ticket{Title: fmt.Sprintf("parallel %d", i)}, "")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("parallel create: %v", err)
		}
	}
	final, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := len(final.ListTickets(TicketFilter{})); got != 4+n {
		t.Fatalf("expected %d tickets, got %d", 4+n, got)
	}
}

func TestClaimTicket(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	created, err := p.CreateTicket(ticket.Ticket{Title: "claim me", Status: ticket.StatusReady}, "")
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	claimed, err := p.ClaimTicket(created.ID, "agent-a")
	if err != nil {
		t.Fatalf("ClaimTicket: %v", err)
	}
	if claimed.Assignee != "agent-a" || claimed.Status != ticket.StatusInProgress {
		t.Fatalf("expected agent-a/in_progress, got %q/%s", claimed.Assignee, claimed.Status)
	}
	if _, err := p.ClaimTicket(created.ID, "agent-a"); err != nil {
		t.Fatalf("re-claim by holder should succeed: %v", err)
	}
	if _, err := p.ClaimTicket(created.ID, "agent-b"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for second claimant, got %v", err)
	}

	got, _ := p.GetTicket(created.ID)
	var kinds []ticket.ActivityKind
	for _, e := range got.Activity {
		kinds = append(kinds, e.Kind)
	}
	want := []ticket.ActivityKind{ticket.ActivityCreated, ticket.ActivityStatus, ticket.ActivityAssignee}
	if !slices.Equal(kinds, want) {
		t.Fatalf("activity kinds = %v, want %v", kinds, want)
	}
}

func TestUpdateTicketPreconditions(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	cur, _ := p.GetTicket("DN-42")
	wrong := ticket.StatusDone
	if cur.Status == wrong {
		wrong = ticket.StatusBacklog
	}
	note := "should not land"
	if _, err := p.UpdateTicket("DN-42", TicketPatch{Note: &note, ExpectStatus: &wrong}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	after, _ := p.GetTicket("DN-42")
	if len(after.Activity) != len(cur.Activity) {
		t.Fatal("failed precondition must not change the ticket")
	}
}

func TestReadyTickets(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	low, _ := p.CreateTicket(ticket.Ticket{Title: "low", Status: ticket.StatusReady, Priority: ticket.PriorityLow}, "")
	high, _ := p.CreateTicket(ticket.Ticket{Title: "high", Status: ticket.StatusReady, Priority: ticket.PriorityHigh}, "")
	blocked, _ := p.CreateTicket(ticket.Ticket{Title: "waits on low", Status: ticket.StatusReady, DependsOn: []string{low.ID}}, "")
	taken, _ := p.CreateTicket(ticket.Ticket{Title: "taken", Status: ticket.StatusReady, Assignee: "someone"}, "")

	var ids []string
	for _, t := range p.ReadyTickets() {
		ids = append(ids, t.ID)
	}
	if slices.Contains(ids, blocked.ID) || slices.Contains(ids, taken.ID) {
		t.Fatalf("ready list includes unmet-dependency or assigned ticket: %v", ids)
	}
	if i, j := slices.Index(ids, high.ID), slices.Index(ids, low.ID); i == -1 || j == -1 || i > j {
		t.Fatalf("expected high before low in %v", ids)
	}
}

func TestActivityIsCapped(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root, root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	for i := range ticket.MaxActivity + 10 {
		note := fmt.Sprintf("note %d", i)
		if _, err := p.UpdateTicket("DN-42", TicketPatch{Note: &note}); err != nil {
			t.Fatalf("UpdateTicket: %v", err)
		}
	}
	got, _ := p.GetTicket("DN-42")
	if len(got.Activity) != ticket.MaxActivity {
		t.Fatalf("expected %d activity entries, got %d", ticket.MaxActivity, len(got.Activity))
	}
	if last := got.Activity[len(got.Activity)-1].Text; last != fmt.Sprintf("note %d", ticket.MaxActivity+9) {
		t.Fatalf("newest entry should be kept, got %q", last)
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
