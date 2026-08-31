package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"localtask/internal/storage"
	"localtask/internal/ticket"
)

// ErrTicketNotFound is returned when a ticket ID doesn't exist on the
// taskboard.
var ErrTicketNotFound = errors.New("ticket not found")

// ErrStaleTaskboard is returned by Save when taskboard.json changed on disk
// since it was last loaded/saved. The caller must Reload before retrying,
// so external edits are never silently overwritten.
var ErrStaleTaskboard = errors.New("taskboard.json changed on disk since load; reload before saving")

// Project is a loaded, in-memory view of a single project's taskboard,
// backed by taskboard.json at Root. It is safe for concurrent use — the
// HTTP API and MCP server are expected to share one Project per project.
type Project struct {
	Name string
	Root string

	mu        sync.RWMutex
	taskboard Taskboard
	modTime   time.Time
}

// LoadProject reads and validates taskboard.json from root.
func LoadProject(name, root string) (*Project, error) {
	tb, modTime, err := readTaskboardFile(root)
	if err != nil {
		return nil, err
	}
	if err := validateTaskboard(&tb, root); err != nil {
		return nil, err
	}
	return &Project{Name: name, Root: root, taskboard: tb, modTime: modTime}, nil
}

func readTaskboardFile(root string) (Taskboard, time.Time, error) {
	path := filepath.Join(root, TaskboardFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Taskboard{}, time.Time{}, fmt.Errorf("reading taskboard file: %w", err)
	}
	var tb Taskboard
	if err := json.Unmarshal(data, &tb); err != nil {
		return Taskboard{}, time.Time{}, fmt.Errorf("parsing taskboard file: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Taskboard{}, time.Time{}, err
	}
	return tb, info.ModTime(), nil
}

// Reload re-reads taskboard.json from disk, discarding any in-memory
// changes that were never saved. Returns an error (leaving the in-memory
// state untouched) if the file is missing, unparseable, or now invalid.
func (p *Project) Reload() error {
	tb, modTime, err := readTaskboardFile(p.Root)
	if err != nil {
		return err
	}
	if err := validateTaskboard(&tb, p.Root); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.taskboard = tb
	p.modTime = modTime
	return nil
}

// Save writes the in-memory taskboard to taskboard.json atomically. It
// refuses to overwrite a file that changed on disk since it was last
// loaded/saved (see ErrStaleTaskboard).
func (p *Project) Save() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := filepath.Join(p.Root, TaskboardFileName)
	if info, err := os.Stat(path); err == nil {
		if info.ModTime().After(p.modTime) {
			return ErrStaleTaskboard
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	data, err := json.MarshalIndent(p.taskboard, "", "  ")
	if err != nil {
		return err
	}
	if err := storage.WriteFileAtomic(path, data, 0o644); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	p.modTime = info.ModTime()
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

// CreateTicket adds a new ticket to the taskboard. Only Title is required;
// ID is generated when empty, and Type/Status/Priority default to
// task/backlog/medium. The mutation is validated before it is committed —
// on failure the taskboard is left unchanged.
func (p *Project) CreateTicket(t ticket.Ticket) (*ticket.Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if t.Title == "" {
		return nil, errors.New("title is required")
	}
	if t.ID == "" {
		t.ID = generateTicketID(&p.taskboard)
	} else {
		for _, existing := range p.taskboard.Tickets {
			if existing.ID == t.ID {
				return nil, fmt.Errorf("ticket id %q already exists", t.ID)
			}
		}
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

	updated := append(append([]ticket.Ticket(nil), p.taskboard.Tickets...), t)
	if err := p.tryCommitTickets(updated); err != nil {
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

	DependsOn          *[]string
	AcceptanceCriteria *[]ticket.AcceptanceCriterion
	Constraints        *[]string
	Tags               *[]string
	RelatedDocs        *[]string
	RelatedCode        *[]string
}

func applyPatch(t ticket.Ticket, patch TicketPatch) ticket.Ticket {
	if patch.Title != nil {
		t.Title = *patch.Title
	}
	if patch.Description != nil {
		t.Description = *patch.Description
	}
	if patch.Type != nil {
		t.Type = *patch.Type
	}
	if patch.Status != nil {
		t.Status = *patch.Status
	}
	if patch.Priority != nil {
		t.Priority = *patch.Priority
	}
	if patch.Parent != nil {
		t.Parent = *patch.Parent
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
	return t
}

// UpdateTicket applies patch to the ticket with the given ID. The mutation
// is validated before it is committed — on failure the taskboard is left
// unchanged.
func (p *Project) UpdateTicket(id string, patch TicketPatch) (*ticket.Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := indexOf(p.taskboard.Tickets, id)
	if idx == -1 {
		return nil, ErrTicketNotFound
	}

	updatedTicket := applyPatch(p.taskboard.Tickets[idx], patch)
	updated := append([]ticket.Ticket(nil), p.taskboard.Tickets...)
	updated[idx] = updatedTicket

	if err := p.tryCommitTickets(updated); err != nil {
		return nil, err
	}
	return &updatedTicket, nil
}

// DeleteTicket removes the ticket with the given ID. The mutation is
// validated before it is committed, so deleting a ticket that other
// tickets still reference as a parent/dependency is rejected.
func (p *Project) DeleteTicket(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := indexOf(p.taskboard.Tickets, id)
	if idx == -1 {
		return ErrTicketNotFound
	}

	updated := append([]ticket.Ticket(nil), p.taskboard.Tickets[:idx]...)
	updated = append(updated, p.taskboard.Tickets[idx+1:]...)

	if err := p.tryCommitTickets(updated); err != nil {
		return fmt.Errorf("cannot delete %q: %w", id, err)
	}
	return nil
}

// tryCommitTickets validates a candidate ticket list against a trial
// taskboard and, only if valid, commits it as the new in-memory state.
// Callers must hold p.mu.
func (p *Project) tryCommitTickets(tickets []ticket.Ticket) error {
	trial := p.taskboard
	trial.Tickets = tickets
	if err := validateTaskboard(&trial, p.Root); err != nil {
		return err
	}
	p.taskboard.Tickets = tickets
	return nil
}

func indexOf(tickets []ticket.Ticket, id string) int {
	for i, t := range tickets {
		if t.ID == id {
			return i
		}
	}
	return -1
}
