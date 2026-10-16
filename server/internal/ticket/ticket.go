// Package ticket defines the core ticket domain types shared by the
// project/taskboard service, the HTTP API, and the MCP server.
package ticket

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

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
	// Jira lists the keys of the Jira issues (e.g. "RT-1234") this ticket
	// is part of. LocalTask only stores and links them; it never calls Jira.
	Jira []string `json:"jira,omitempty"`

	// Assignee names the agent (or person) currently working the ticket.
	// Agents self-report a name; see the MCP server's claim_ticket tool.
	Assignee  string    `json:"assignee,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`

	// Activity is a capped, newest-last log of what happened to the ticket:
	// status/assignee changes recorded automatically, plus free-text notes
	// agents leave for whoever picks the work up next.
	Activity []ActivityEntry `json:"activity,omitempty"`
}

// ActivityKind classifies an ActivityEntry.
type ActivityKind string

const (
	ActivityCreated  ActivityKind = "created"
	ActivityStatus   ActivityKind = "status"
	ActivityAssignee ActivityKind = "assignee"
	ActivityNote     ActivityKind = "note"
)

// ActivityEntry is one line in a ticket's activity log.
type ActivityEntry struct {
	At     time.Time    `json:"at"`
	Author string       `json:"author,omitempty"`
	Kind   ActivityKind `json:"kind"`
	Text   string       `json:"text"`
}

// MaxActivity caps a ticket's activity log so taskboard.json (which lives
// in git) doesn't grow without bound; the oldest entries are dropped.
const MaxActivity = 50

// AddActivity appends an entry, dropping the oldest beyond MaxActivity.
func (t *Ticket) AddActivity(e ActivityEntry) {
	t.Activity = append(t.Activity, e)
	if over := len(t.Activity) - MaxActivity; over > 0 {
		t.Activity = append([]ActivityEntry(nil), t.Activity[over:]...)
	}
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

// HasJira reports whether the ticket links the given Jira key, ignoring
// case.
func (t Ticket) HasJira(key string) bool {
	return slices.ContainsFunc(t.Jira, func(k string) bool { return strings.EqualFold(k, key) })
}

// JiraKeyPattern matches a Jira issue key: a project key of uppercase
// letters, digits, or underscores starting with a letter, then a dash and
// an issue number.
var JiraKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*-[1-9][0-9]*$`)

// HasTag reports whether the ticket carries the given tag.
func (t Ticket) HasTag(tag string) bool {
	return slices.Contains(t.Tags, tag)
}
