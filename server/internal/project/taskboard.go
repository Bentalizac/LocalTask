// Package project implements the taskboard domain/service layer: loading,
// validating, mutating, and persisting a project's taskboard.json. This is
// the single source of truth shared by the HTTP API and the MCP server.
package project

import "localtask/internal/ticket"

// TaskboardFileName is the name of the per-project task file, always
// located at the project's root directory.
const TaskboardFileName = "taskboard.json"

// Taskboard is the on-disk (and in-memory) representation of a project's
// tickets, as stored in taskboard.json.
type Taskboard struct {
	Version     int             `json:"version"`
	Project     string          `json:"project"`
	Description string          `json:"description,omitempty"`
	Tickets     []ticket.Ticket `json:"tickets"`
}

// clone returns a deep-enough copy of the taskboard: the Tickets slice and
// its backing array are copied so callers can't mutate project state
// through a returned snapshot.
func (tb Taskboard) clone() Taskboard {
	tb.Tickets = append([]ticket.Ticket(nil), tb.Tickets...)
	return tb
}
