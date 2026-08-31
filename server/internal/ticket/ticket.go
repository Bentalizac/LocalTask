// Package ticket defines the core ticket domain types shared by the
// project/taskboard service, the HTTP API, and the MCP server.
package ticket

import "slices"

// Type is the kind of work a ticket represents.
type Type string

const (
	TypeEpic    Type = "epic"
	TypeStory   Type = "story"
	TypeTask    Type = "task"
	TypeBug     Type = "bug"
	TypeSubtask Type = "subtask"
)

// Valid reports whether t is one of the supported ticket types.
func (t Type) Valid() bool {
	switch t {
	case TypeEpic, TypeStory, TypeTask, TypeBug, TypeSubtask:
		return true
	}
	return false
}

// Status is a ticket's position in the workflow.
type Status string

const (
	StatusBacklog    Status = "backlog"
	StatusReady      Status = "ready"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusReview     Status = "review"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// Valid reports whether s is one of the supported ticket statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusBacklog, StatusReady, StatusInProgress, StatusBlocked, StatusReview, StatusDone, StatusCancelled:
		return true
	}
	return false
}

// Priority is a ticket's relative urgency.
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Valid reports whether p is one of the supported ticket priorities.
func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	}
	return false
}

// AcceptanceCriterion is a single measurable condition a ticket must
// satisfy to be considered done.
type AcceptanceCriterion struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Complete bool   `json:"complete"`
}

// Ticket is a single unit of work tracked in a project's taskboard.
type Ticket struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Type        Type     `json:"type"`
	Status      Status   `json:"status"`
	Priority    Priority `json:"priority"`

	Parent    string   `json:"parent,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`

	AcceptanceCriteria []AcceptanceCriterion `json:"acceptance_criteria,omitempty"`
	Constraints        []string              `json:"constraints,omitempty"`
	Tags               []string              `json:"tags,omitempty"`
	RelatedDocs        []string              `json:"related_docs,omitempty"`
	RelatedCode        []string              `json:"related_code,omitempty"`
}

// AcceptanceCriteriaProgress returns the number of complete acceptance
// criteria and the total number defined for the ticket.
func (t Ticket) AcceptanceCriteriaProgress() (complete, total int) {
	total = len(t.AcceptanceCriteria)
	for _, ac := range t.AcceptanceCriteria {
		if ac.Complete {
			complete++
		}
	}
	return complete, total
}

// HasTag reports whether the ticket carries the given tag.
func (t Ticket) HasTag(tag string) bool {
	return slices.Contains(t.Tags, tag)
}
