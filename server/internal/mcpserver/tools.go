package mcpserver

import (
	"context"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/document"
	"localtask/internal/git"
	"localtask/internal/project"
	"localtask/internal/ticket"
)

func (s *server) registerTools(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_projects",
		Description: "List every configured project.",
	}, s.listProjects)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_project",
		Description: "Get metadata for one configured project.",
	}, s.getProject)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_project_context",
		Description: "Get concise context for a project: description, tickets actively being worked on, blocked tickets, and referenced documentation. Intended as the first call an agent makes when starting work on a project.",
	}, s.getProjectContext)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_tickets",
		Description: "List a project's tickets, optionally filtered by status, priority, type, tag, or parent.",
	}, s.listTickets)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_ticket",
		Description: "Get complete information for one ticket, including acceptance criteria, dependencies, and related docs/code.",
	}, s.getTicket)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "search_tickets",
		Description: "Search a project's tickets by matching text against title, description, tags, and Jira keys.",
	}, s.searchTickets)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_ticket",
		Description: "Create a new ticket in a project. Only title is required; id is generated if omitted, and type/status/priority default to task/backlog/medium.",
	}, s.createTicket)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_ticket",
		Description: "Update fields on an existing ticket. Only fields explicitly provided are changed. Use this to change status, leave a progress note, update acceptance criteria, release a ticket (assignee: \"\"), or add tags/related docs/related code as work progresses. Always pass your agent name.",
	}, s.updateTicket)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "claim_ticket",
		Description: "Atomically claim a ticket: assign it to you and move it to in_progress. Fails if another agent already holds it or it is done/cancelled — pick a different ticket rather than retrying. Re-claiming a ticket you already hold succeeds.",
	}, s.claimTicket)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "next_ready_tickets",
		Description: "List tickets that can be picked up right now: status ready, unassigned, and all dependencies done or cancelled. Ordered by priority, highest first.",
	}, s.nextReadyTickets)
}

// --- list_projects ---

type listProjectsArgs struct{}

type projectInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Version     int             `json:"version"`
	Metrics     project.Metrics `json:"metrics"`
	JiraBaseURL string          `json:"jira_base_url,omitempty" jsonschema:"Jira site that tickets' jira keys belong to; an issue's URL is <jira_base_url>/browse/<key>"`
}

func toProjectInfo(name string, p *project.Project) projectInfo {
	snap := p.Snapshot()
	return projectInfo{Name: name, Description: snap.Description, Version: snap.Version, Metrics: p.Metrics(), JiraBaseURL: p.JiraBaseURL}
}

// listProjectsResult wraps the project list in a named field rather than
// returning a bare array. A Go slice's inferred JSON Schema has type
// ["null","array"] at the root (nilable), and some MCP clients' output
// validation chokes on a non-"object" root schema even though the MCP spec
// itself (post SEP-2106) allows it. Every other tool here already returns
// an object; this keeps list-shaped tools consistent with that and
// sidesteps the client compatibility gap entirely.
type listProjectsResult struct {
	Projects []projectInfo `json:"projects"`
}

func (s *server) listProjects(_ context.Context, _ *mcp.CallToolRequest, _ listProjectsArgs) (*mcp.CallToolResult, listProjectsResult, error) {
	names := make([]string, 0, len(s.projects))
	for name := range s.projects {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]projectInfo, 0, len(names))
	for _, name := range names {
		out = append(out, toProjectInfo(name, s.projects[name]))
	}
	return nil, listProjectsResult{Projects: out}, nil
}

// --- get_project ---

type projectArgs struct {
	Project string `json:"project" jsonschema:"the configured project name"`
}

func (s *server) getProject(_ context.Context, _ *mcp.CallToolRequest, args projectArgs) (*mcp.CallToolResult, projectInfo, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, projectInfo{}, err
	}
	return nil, toProjectInfo(args.Project, p), nil
}

// --- get_project_context ---

type projectContext struct {
	Project        string          `json:"project"`
	Description    string          `json:"description,omitempty"`
	Metrics        project.Metrics `json:"metrics"`
	JiraBaseURL    string          `json:"jira_base_url,omitempty" jsonschema:"Jira site that tickets' jira keys belong to; an issue's URL is <jira_base_url>/browse/<key>"`
	ActiveTickets  []ticket.Ticket `json:"active_tickets"`
	BlockedTickets []ticket.Ticket `json:"blocked_tickets"`
	Documentation  []string        `json:"documentation"`
	Git            git.Info        `json:"git"`
}

func (s *server) getProjectContext(ctx context.Context, _ *mcp.CallToolRequest, args projectArgs) (*mcp.CallToolResult, projectContext, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, projectContext{}, err
	}
	snap := p.Snapshot()

	var active, blocked []ticket.Ticket
	for _, t := range snap.Tickets {
		if t.Status == ticket.StatusInProgress {
			active = append(active, t)
		}
		if t.Status == ticket.StatusBlocked {
			blocked = append(blocked, t)
		}
	}
	if active == nil {
		active = []ticket.Ticket{}
	}
	if blocked == nil {
		blocked = []ticket.Ticket{}
	}

	docs, err := document.List(p.Root)
	if err != nil {
		return nil, projectContext{}, err
	}
	if docs == nil {
		docs = []string{}
	}

	// Git info is never a prerequisite (spec §21): a lookup failure just
	// means no git info, not a broken context call.
	gitInfo, err := git.Read(ctx, p.Root, 10)
	if err != nil || gitInfo == nil {
		gitInfo = &git.Info{IsRepo: false}
	}

	return nil, projectContext{
		Project:        args.Project,
		Description:    snap.Description,
		Metrics:        p.Metrics(),
		JiraBaseURL:    p.JiraBaseURL,
		ActiveTickets:  active,
		BlockedTickets: blocked,
		Documentation:  docs,
		Git:            *gitInfo,
	}, nil
}

// --- list_tickets ---

type listTicketsArgs struct {
	Project  string `json:"project" jsonschema:"the configured project name"`
	Status   string `json:"status,omitempty" jsonschema:"filter by status"`
	Priority string `json:"priority,omitempty" jsonschema:"filter by priority"`
	Type     string `json:"type,omitempty" jsonschema:"filter by ticket type"`
	Tag      string `json:"tag,omitempty" jsonschema:"filter by tag"`
	Parent   string `json:"parent,omitempty" jsonschema:"filter by parent ticket id"`
}

// listTicketsResult wraps the ticket list — see listProjectsResult for why.
type listTicketsResult struct {
	Tickets []ticket.Ticket `json:"tickets"`
}

func (s *server) listTickets(_ context.Context, _ *mcp.CallToolRequest, args listTicketsArgs) (*mcp.CallToolResult, listTicketsResult, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, listTicketsResult{}, err
	}

	var filter project.TicketFilter
	if args.Status != "" {
		v := ticket.Status(args.Status)
		filter.Status = &v
	}
	if args.Priority != "" {
		v := ticket.Priority(args.Priority)
		filter.Priority = &v
	}
	if args.Type != "" {
		v := ticket.Type(args.Type)
		filter.Type = &v
	}
	if args.Tag != "" {
		filter.Tag = &args.Tag
	}
	if args.Parent != "" {
		filter.Parent = &args.Parent
	}

	tickets := p.ListTickets(filter)
	if tickets == nil {
		tickets = []ticket.Ticket{}
	}
	return nil, listTicketsResult{Tickets: tickets}, nil
}

// --- get_ticket ---

type ticketArgs struct {
	Project  string `json:"project" jsonschema:"the configured project name"`
	TicketID string `json:"ticket_id" jsonschema:"the ticket id, e.g. DN-42"`
}

func (s *server) getTicket(_ context.Context, _ *mcp.CallToolRequest, args ticketArgs) (*mcp.CallToolResult, ticket.Ticket, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	t, err := p.GetTicket(args.TicketID)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	return nil, *t, nil
}

// --- search_tickets ---

type searchTicketsArgs struct {
	Project string `json:"project" jsonschema:"the configured project name"`
	Query   string `json:"query" jsonschema:"text to match against title, description, tags, and Jira keys"`
}

func (s *server) searchTickets(_ context.Context, _ *mcp.CallToolRequest, args searchTicketsArgs) (*mcp.CallToolResult, listTicketsResult, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, listTicketsResult{}, err
	}
	q := strings.ToLower(strings.TrimSpace(args.Query))
	var matches []ticket.Ticket
	for _, t := range p.ListTickets(project.TicketFilter{}) {
		if q == "" ||
			strings.Contains(strings.ToLower(t.Title), q) ||
			strings.Contains(strings.ToLower(t.Description), q) ||
			t.HasTag(q) ||
			t.HasJira(q) {
			matches = append(matches, t)
		}
	}
	if matches == nil {
		matches = []ticket.Ticket{}
	}
	return nil, listTicketsResult{Tickets: matches}, nil
}

// --- create_ticket ---

type createTicketArgs struct {
	Project            string                       `json:"project" jsonschema:"the configured project name"`
	ID                 string                       `json:"id,omitempty" jsonschema:"optional explicit ticket id; generated if omitted"`
	Title              string                       `json:"title" jsonschema:"required title"`
	Description        string                       `json:"description,omitempty"`
	Type               string                       `json:"type,omitempty" jsonschema:"epic, story, task, bug, or subtask; defaults to task"`
	Status             string                       `json:"status,omitempty" jsonschema:"defaults to backlog"`
	Priority           string                       `json:"priority,omitempty" jsonschema:"defaults to medium"`
	Parent             string                       `json:"parent,omitempty" jsonschema:"parent ticket id"`
	DependsOn          []string                     `json:"depends_on,omitempty"`
	AcceptanceCriteria []ticket.AcceptanceCriterion `json:"acceptance_criteria,omitempty"`
	Constraints        []string                     `json:"constraints,omitempty"`
	Tags               []string                     `json:"tags,omitempty"`
	RelatedDocs        []string                     `json:"related_docs,omitempty"`
	RelatedCode        []string                     `json:"related_code,omitempty"`
	Jira               []string                     `json:"jira,omitempty" jsonschema:"keys of the Jira issues this ticket is part of, e.g. RT-1234"`
	Assignee           string                       `json:"assignee,omitempty" jsonschema:"agent to assign the ticket to"`
	Agent              string                       `json:"agent,omitempty" jsonschema:"your agent name, recorded in the ticket's activity log"`
}

func (s *server) createTicket(_ context.Context, _ *mcp.CallToolRequest, args createTicketArgs) (*mcp.CallToolResult, ticket.Ticket, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}

	created, err := p.CreateTicket(ticket.Ticket{
		ID:                 args.ID,
		Title:              args.Title,
		Description:        args.Description,
		Type:               ticket.Type(args.Type),
		Status:             ticket.Status(args.Status),
		Priority:           ticket.Priority(args.Priority),
		Parent:             args.Parent,
		DependsOn:          args.DependsOn,
		AcceptanceCriteria: args.AcceptanceCriteria,
		Constraints:        args.Constraints,
		Tags:               args.Tags,
		RelatedDocs:        args.RelatedDocs,
		RelatedCode:        args.RelatedCode,
		Jira:               args.Jira,
		Assignee:           args.Assignee,
	}, args.Agent)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	return nil, *created, nil
}

// --- update_ticket ---

// updateTicketArgs uses pointer fields (except the required Project/TicketID)
// so that a field omitted from the call leaves the ticket's current value
// untouched — updates must be explicit (see spec §7).
type updateTicketArgs struct {
	Project            string                        `json:"project" jsonschema:"the configured project name"`
	TicketID           string                        `json:"ticket_id" jsonschema:"the ticket id, e.g. DN-42"`
	Title              *string                       `json:"title,omitempty"`
	Description        *string                       `json:"description,omitempty"`
	Type               *string                       `json:"type,omitempty"`
	Status             *string                       `json:"status,omitempty"`
	Priority           *string                       `json:"priority,omitempty"`
	Parent             *string                       `json:"parent,omitempty"`
	DependsOn          *[]string                     `json:"depends_on,omitempty"`
	AcceptanceCriteria *[]ticket.AcceptanceCriterion `json:"acceptance_criteria,omitempty" jsonschema:"the full replacement acceptance criteria list; do not mark criteria complete unless verified"`
	Constraints        *[]string                     `json:"constraints,omitempty"`
	Tags               *[]string                     `json:"tags,omitempty"`
	RelatedDocs        *[]string                     `json:"related_docs,omitempty"`
	RelatedCode        *[]string                     `json:"related_code,omitempty"`
	Jira               *[]string                     `json:"jira,omitempty" jsonschema:"the full replacement list of Jira issue keys this ticket is part of, e.g. RT-1234"`
	Assignee           *string                       `json:"assignee,omitempty" jsonschema:"set the assignee; empty string releases the ticket"`
	Note               *string                       `json:"note,omitempty" jsonschema:"a progress or handoff note to append to the ticket's activity log"`
	Agent              string                        `json:"agent,omitempty" jsonschema:"your agent name, recorded in the ticket's activity log"`
	ExpectedStatus     *string                       `json:"expected_status,omitempty" jsonschema:"only apply the update if the ticket currently has this status"`
	ExpectedAssignee   *string                       `json:"expected_assignee,omitempty" jsonschema:"only apply the update if the ticket is currently assigned to this agent (empty string: unassigned)"`
}

func (a updateTicketArgs) toPatch() project.TicketPatch {
	patch := project.TicketPatch{
		Title:              a.Title,
		Description:        a.Description,
		Parent:             a.Parent,
		DependsOn:          a.DependsOn,
		AcceptanceCriteria: a.AcceptanceCriteria,
		Constraints:        a.Constraints,
		Tags:               a.Tags,
		RelatedDocs:        a.RelatedDocs,
		RelatedCode:        a.RelatedCode,
		Jira:               a.Jira,
		Assignee:           a.Assignee,
		Note:               a.Note,
		Author:             a.Agent,
		ExpectAssignee:     a.ExpectedAssignee,
	}
	if a.ExpectedStatus != nil {
		v := ticket.Status(*a.ExpectedStatus)
		patch.ExpectStatus = &v
	}
	if a.Type != nil {
		v := ticket.Type(*a.Type)
		patch.Type = &v
	}
	if a.Status != nil {
		v := ticket.Status(*a.Status)
		patch.Status = &v
	}
	if a.Priority != nil {
		v := ticket.Priority(*a.Priority)
		patch.Priority = &v
	}
	return patch
}

func (s *server) updateTicket(_ context.Context, _ *mcp.CallToolRequest, args updateTicketArgs) (*mcp.CallToolResult, ticket.Ticket, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	updated, err := p.UpdateTicket(args.TicketID, args.toPatch())
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	return nil, *updated, nil
}

// --- claim_ticket ---

type claimTicketArgs struct {
	Project  string `json:"project" jsonschema:"the configured project name"`
	TicketID string `json:"ticket_id" jsonschema:"the ticket id, e.g. DN-42"`
	Agent    string `json:"agent" jsonschema:"your agent name; becomes the ticket's assignee"`
}

func (s *server) claimTicket(_ context.Context, _ *mcp.CallToolRequest, args claimTicketArgs) (*mcp.CallToolResult, ticket.Ticket, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	claimed, err := p.ClaimTicket(args.TicketID, args.Agent)
	if err != nil {
		return nil, ticket.Ticket{}, err
	}
	return nil, *claimed, nil
}

// --- next_ready_tickets ---

func (s *server) nextReadyTickets(_ context.Context, _ *mcp.CallToolRequest, args projectArgs) (*mcp.CallToolResult, listTicketsResult, error) {
	p, err := s.project(args.Project)
	if err != nil {
		return nil, listTicketsResult{}, err
	}
	tickets := p.ReadyTickets()
	if tickets == nil {
		tickets = []ticket.Ticket{}
	}
	return nil, listTicketsResult{Tickets: tickets}, nil
}
