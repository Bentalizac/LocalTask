package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"localtask/internal/storage"
	"localtask/internal/ticket"
)

// ErrTicketNotFound is returned when a ticket ID doesn't exist on the
// taskboard.
var ErrTicketNotFound = errors.New("ticket not found")

// Project is a loaded, in-memory view of a single project's taskboard,
// backed by taskboard.json in BoardDir. It is safe for concurrent use — the
// HTTP API and MCP server are expected to share one Project per project.
type Project struct {
	Name string
	// Root is the project's own directory: docs, git info, and related
	// paths resolve against it. LocalTask only reads from it, unless
	// BoardDir is the same directory.
	Root string
	// BoardDir holds taskboard.json. It equals Root for a board committed
	// with the project, or is a directory outside the project otherwise
	// (see config.Storage).
	BoardDir string
	// JiraBaseURL is the Jira site that tickets' Jira keys link to, e.g.
	// "https://example.atlassian.net". Empty means keys aren't linked.
	JiraBaseURL string

	mu        sync.RWMutex
	taskboard Taskboard
}

// LoadProject reads taskboard.json from boardDir and validates it against
// the project at root.
func LoadProject(name, root, boardDir string) (*Project, error) {
	tb, err := readTaskboardFile(boardDir)
	if err != nil {
		return nil, err
	}
	if err := validateTaskboard(&tb, root); err != nil {
		return nil, err
	}
	return &Project{Name: name, Root: root, BoardDir: boardDir, taskboard: tb}, nil
}

func readTaskboardFile(boardDir string) (Taskboard, error) {
	data, err := os.ReadFile(filepath.Join(boardDir, TaskboardFileName))
	if err != nil {
		return Taskboard{}, fmt.Errorf("reading taskboard file: %w", err)
	}
	var tb Taskboard
	if err := json.Unmarshal(data, &tb); err != nil {
		return Taskboard{}, fmt.Errorf("parsing taskboard file: %w", err)
	}
	return tb, nil
}

// Reload re-reads taskboard.json from disk, discarding any in-memory
// changes that were never saved. Returns an error (leaving the in-memory
// state untouched) if the file is missing, unparseable, or now invalid.
func (p *Project) Reload() error {
	tb, err := readTaskboardFile(p.BoardDir)
	if err != nil {
		return err
	}
	if err := validateTaskboard(&tb, p.Root); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.taskboard = tb
	return nil
}

// mutate applies fn to the freshest on-disk taskboard and persists the
// result, all under a cross-process lock on the board's directory. Every
// write goes through here, so concurrent writers — the HTTP server and any
// number of MCP server processes, one per agent session — serialize
// instead of overwriting each other: each one re-reads whatever the last
// writer left, applies only its own change, and writes that back. fn
// mutates tb in place; the result is validated before anything is written,
// and on any error neither the file nor the in-memory state changes.
func (p *Project) mutate(fn func(tb *Taskboard) error) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	unlock, err := storage.LockDir(p.BoardDir)
	if err != nil {
		return fmt.Errorf("locking project: %w", err)
	}
	defer unlock()

	tb, err := readTaskboardFile(p.BoardDir)
	if err != nil {
		return err
	}
	if err := fn(&tb); err != nil {
		return err
	}
	if err := validateTaskboard(&tb, p.Root); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tb, "", "  ")
	if err != nil {
		return err
	}
	if err := storage.WriteFileAtomic(filepath.Join(p.BoardDir, TaskboardFileName), data, 0o644); err != nil {
		return err
	}
	p.taskboard = tb
	return nil
}

// Validate re-checks the in-memory taskboard against the domain rules.
func (p *Project) Validate() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return validateTaskboard(&p.taskboard, p.Root)
}

// Snapshot returns a copy of the full in-memory taskboard.
func (p *Project) Snapshot() Taskboard {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.taskboard.clone()
}

// TicketFilter selects a subset of tickets in ListTickets. Nil fields are
// unconstrained.
type TicketFilter struct {
	Status   *ticket.Status
	Priority *ticket.Priority
	Type     *ticket.Type
	Tag      *string
	Parent   *string
}

func (f TicketFilter) matches(t ticket.Ticket) bool {
	if f.Status != nil && t.Status != *f.Status {
		return false
	}
	if f.Priority != nil && t.Priority != *f.Priority {
		return false
	}
	if f.Type != nil && t.Type != *f.Type {
		return false
	}
	if f.Parent != nil && t.Parent != *f.Parent {
		return false
	}
	if f.Tag != nil && !t.HasTag(*f.Tag) {
		return false
	}
	return true
}

// ListTickets returns every ticket matching filter, in taskboard order.
func (p *Project) ListTickets(filter TicketFilter) []ticket.Ticket {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var result []ticket.Ticket
	for _, t := range p.taskboard.Tickets {
		if filter.matches(t) {
			result = append(result, t)
		}
	}
	return result
}

// GetTicket returns a copy of the ticket with the given ID, or
// ErrTicketNotFound.
func (p *Project) GetTicket(id string) (*ticket.Ticket, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, t := range p.taskboard.Tickets {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, ErrTicketNotFound
}

// ErrConflict is returned when a conditional update's precondition no
// longer holds — e.g. another agent claimed the ticket first.
var ErrConflict = errors.New("conflict")

// now is the clock used for UpdatedAt and activity timestamps; tests
// override it.
var now = func() time.Time { return time.Now().UTC().Truncate(time.Second) }

// CreateTicket adds a new ticket to the taskboard and persists it. Only
// Title is required; ID is generated when empty, and Type/Status/Priority
// default to task/backlog/medium. author (optional) is recorded in the
// activity log.
func (p *Project) CreateTicket(t ticket.Ticket, author string) (*ticket.Ticket, error) {
	if t.Title == "" {
		return nil, errors.New("title is required")
	}
	err := p.mutate(func(tb *Taskboard) error {
		if t.ID == "" {
			t.ID = generateTicketID(tb)
		} else if indexOf(tb.Tickets, t.ID) != -1 {
			return fmt.Errorf("ticket id %q already exists", t.ID)
		}
		if t.Type == "" {
			t.Type = ticket.TypeTask
		}
		if t.Status == "" {
			t.Status = ticket.StatusBacklog
		}
		if t.Priority == "" {
			t.Priority = ticket.PriorityMedium
		}
		t.UpdatedAt = now()
		t.Activity = nil
		t.AddActivity(ticket.ActivityEntry{At: t.UpdatedAt, Author: author, Kind: ticket.ActivityCreated, Text: "created"})
		tb.Tickets = append(tb.Tickets, t)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TicketPatch describes an explicit, partial update to a ticket: only
// non-nil fields are applied, so a client can never accidentally clear a
// field it didn't intend to touch.
type TicketPatch struct {
	Title       *string
	Description *string
	Type        *ticket.Type
	Status      *ticket.Status
	Priority    *ticket.Priority
	Parent      *string
	Assignee    *string

	DependsOn          *[]string
	AcceptanceCriteria *[]ticket.AcceptanceCriterion
	Constraints        *[]string
	Tags               *[]string
	RelatedDocs        *[]string
	RelatedCode        *[]string
	Jira               *[]string

	// Note, if set, is appended to the ticket's activity log.
	Note *string
	// Author is who's making the change, recorded on activity entries.
	Author string

	// ExpectStatus and ExpectAssignee are optional preconditions: if set and
	// the ticket's current value differs, the update fails with ErrConflict
	// and nothing changes.
	ExpectStatus   *ticket.Status
	ExpectAssignee *string
}

func applyPatch(t ticket.Ticket, patch TicketPatch) ticket.Ticket {
	at := now()
	if patch.Title != nil {
		t.Title = *patch.Title
	}
	if patch.Description != nil {
		t.Description = *patch.Description
	}
	if patch.Type != nil {
		t.Type = *patch.Type
	}
	if patch.Status != nil && *patch.Status != t.Status {
		t.AddActivity(ticket.ActivityEntry{
			At: at, Author: patch.Author, Kind: ticket.ActivityStatus,
			Text: fmt.Sprintf("%s → %s", t.Status, *patch.Status),
		})
		t.Status = *patch.Status
	}
	if patch.Priority != nil {
		t.Priority = *patch.Priority
	}
	if patch.Parent != nil {
		t.Parent = *patch.Parent
	}
	if patch.Assignee != nil && *patch.Assignee != t.Assignee {
		text := "assigned to " + *patch.Assignee
		if *patch.Assignee == "" {
			text = "unassigned (was " + t.Assignee + ")"
		}
		t.AddActivity(ticket.ActivityEntry{At: at, Author: patch.Author, Kind: ticket.ActivityAssignee, Text: text})
		t.Assignee = *patch.Assignee
	}
	if patch.DependsOn != nil {
		t.DependsOn = *patch.DependsOn
	}
	if patch.AcceptanceCriteria != nil {
		t.AcceptanceCriteria = *patch.AcceptanceCriteria
	}
	if patch.Constraints != nil {
		t.Constraints = *patch.Constraints
	}
	if patch.Tags != nil {
		t.Tags = *patch.Tags
	}
	if patch.RelatedDocs != nil {
		t.RelatedDocs = *patch.RelatedDocs
	}
	if patch.RelatedCode != nil {
		t.RelatedCode = *patch.RelatedCode
	}
	if patch.Jira != nil {
		t.Jira = *patch.Jira
	}
	if patch.Note != nil && *patch.Note != "" {
		t.AddActivity(ticket.ActivityEntry{At: at, Author: patch.Author, Kind: ticket.ActivityNote, Text: *patch.Note})
	}
	t.UpdatedAt = at
	return t
}

// UpdateTicket applies patch to the ticket with the given ID and persists
// it. The patch is applied to the latest on-disk state, so concurrent
// updates to other tickets (or other fields) are never lost.
func (p *Project) UpdateTicket(id string, patch TicketPatch) (*ticket.Ticket, error) {
	var updated ticket.Ticket
	err := p.mutate(func(tb *Taskboard) error {
		idx := indexOf(tb.Tickets, id)
		if idx == -1 {
			return ErrTicketNotFound
		}
		cur := tb.Tickets[idx]
		if patch.ExpectStatus != nil && cur.Status != *patch.ExpectStatus {
			return fmt.Errorf("%w: %s status is %q, expected %q", ErrConflict, id, cur.Status, *patch.ExpectStatus)
		}
		if patch.ExpectAssignee != nil && cur.Assignee != *patch.ExpectAssignee {
			return fmt.Errorf("%w: %s is assigned to %q, expected %q", ErrConflict, id, cur.Assignee, *patch.ExpectAssignee)
		}
		updated = applyPatch(cur, patch)
		tb.Tickets[idx] = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// ClaimTicket atomically assigns the ticket to agent and moves it to
// in_progress (unless it's already in_progress or review). It fails with
// ErrConflict if someone else holds the ticket or it's already finished,
// so two agents can never both claim the same work. Re-claiming a ticket
// you already hold is a no-op success.
func (p *Project) ClaimTicket(id, agent string) (*ticket.Ticket, error) {
	if agent == "" {
		return nil, errors.New("agent name is required to claim a ticket")
	}
	var updated ticket.Ticket
	err := p.mutate(func(tb *Taskboard) error {
		idx := indexOf(tb.Tickets, id)
		if idx == -1 {
			return ErrTicketNotFound
		}
		cur := tb.Tickets[idx]
		if cur.Assignee != "" && cur.Assignee != agent {
			return fmt.Errorf("%w: %s is already claimed by %q", ErrConflict, id, cur.Assignee)
		}
		if cur.Status == ticket.StatusDone || cur.Status == ticket.StatusCancelled {
			return fmt.Errorf("%w: %s is already %s", ErrConflict, id, cur.Status)
		}
		patch := TicketPatch{Assignee: &agent, Author: agent}
		if cur.Status != ticket.StatusInProgress && cur.Status != ticket.StatusReview {
			s := ticket.StatusInProgress
			patch.Status = &s
		}
		updated = applyPatch(cur, patch)
		tb.Tickets[idx] = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// ReleaseTicket unassigns a ticket agent holds, appending note (if any) to
// its activity log. An in_progress ticket goes back to ready so another
// agent can pick it up. It fails with ErrConflict if someone else holds the
// ticket. Releasing an unassigned ticket only appends the note.
func (p *Project) ReleaseTicket(id, agent, note string) (*ticket.Ticket, error) {
	if agent == "" {
		return nil, errors.New("agent name is required to release a ticket")
	}
	var updated ticket.Ticket
	err := p.mutate(func(tb *Taskboard) error {
		idx := indexOf(tb.Tickets, id)
		if idx == -1 {
			return ErrTicketNotFound
		}
		cur := tb.Tickets[idx]
		if cur.Assignee != "" && cur.Assignee != agent {
			return fmt.Errorf("%w: %s is claimed by %q, not %q", ErrConflict, id, cur.Assignee, agent)
		}
		unassigned := ""
		patch := TicketPatch{Assignee: &unassigned, Author: agent}
		if note != "" {
			patch.Note = &note
		}
		if cur.Status == ticket.StatusInProgress {
			s := ticket.StatusReady
			patch.Status = &s
		}
		updated = applyPatch(cur, patch)
		tb.Tickets[idx] = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// ReadyTickets returns tickets an agent could pick up right now: status
// ready, unassigned, and every dependency done or cancelled. Ordered by
// priority (critical first), then board order.
func (p *Project) ReadyTickets() []ticket.Ticket {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := make(map[string]ticket.Status, len(p.taskboard.Tickets))
	for _, t := range p.taskboard.Tickets {
		status[t.ID] = t.Status
	}
	var out []ticket.Ticket
	for _, t := range p.taskboard.Tickets {
		if t.Status != ticket.StatusReady || t.Assignee != "" {
			continue
		}
		ready := true
		for _, dep := range t.DependsOn {
			if s := status[dep]; s != ticket.StatusDone && s != ticket.StatusCancelled {
				ready = false
				break
			}
		}
		if ready {
			out = append(out, t)
		}
	}
	slices.SortStableFunc(out, func(a, b ticket.Ticket) int {
		return priorityRank[b.Priority] - priorityRank[a.Priority]
	})
	return out
}

var priorityRank = map[ticket.Priority]int{
	ticket.PriorityLow: 0, ticket.PriorityMedium: 1, ticket.PriorityHigh: 2, ticket.PriorityCritical: 3,
}

// DeleteTicket removes the ticket with the given ID and persists the
// change. Deleting a ticket that other tickets still reference as a
// parent/dependency is rejected by validation.
func (p *Project) DeleteTicket(id string) error {
	err := p.mutate(func(tb *Taskboard) error {
		idx := indexOf(tb.Tickets, id)
		if idx == -1 {
			return ErrTicketNotFound
		}
		tb.Tickets = slices.Delete(tb.Tickets, idx, idx+1)
		return nil
	})
	if err != nil && !errors.Is(err, ErrTicketNotFound) {
		return fmt.Errorf("cannot delete %q: %w", id, err)
	}
	return err
}

func indexOf(tickets []ticket.Ticket, id string) int {
	for i, t := range tickets {
		if t.ID == id {
			return i
		}
	}
	return -1
}
