package project

import "localtask/internal/ticket"

// Metrics is a taskboard's basic progress metrics (spec §16). Computed here
// once so the HTTP API and MCP server never disagree on how a number is
// derived — see spec §2: "Do not implement separate task-management logic
// for MCP and HTTP."
type Metrics struct {
	Total      int `json:"total"`
	Backlog    int `json:"backlog"`
	Ready      int `json:"ready"`
	InProgress int `json:"in_progress"`
	Blocked    int `json:"blocked"`
	Review     int `json:"review"`
	Done       int `json:"done"`
	Cancelled  int `json:"cancelled"`

	// CompletionPercent is Done/Total as a rounded percentage (0 if Total
	// is 0). It is an indicator based on the task model, not a precise
	// measure of real-world progress (spec §16).
	CompletionPercent int `json:"completion_percent"`

	AcceptanceCriteriaComplete int `json:"acceptance_criteria_complete"`
	AcceptanceCriteriaTotal    int `json:"acceptance_criteria_total"`
}

// Metrics computes the taskboard's current progress metrics.
func (p *Project) Metrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var m Metrics
	m.Total = len(p.taskboard.Tickets)
	for _, t := range p.taskboard.Tickets {
		switch t.Status {
		case ticket.StatusBacklog:
			m.Backlog++
		case ticket.StatusReady:
			m.Ready++
		case ticket.StatusInProgress:
			m.InProgress++
		case ticket.StatusBlocked:
			m.Blocked++
		case ticket.StatusReview:
			m.Review++
		case ticket.StatusDone:
			m.Done++
		case ticket.StatusCancelled:
			m.Cancelled++
		}
		complete, total := t.AcceptanceCriteriaProgress()
		m.AcceptanceCriteriaComplete += complete
		m.AcceptanceCriteriaTotal += total
	}
	if m.Total > 0 {
		m.CompletionPercent = (m.Done*100 + m.Total/2) / m.Total
	}
	return m
}
