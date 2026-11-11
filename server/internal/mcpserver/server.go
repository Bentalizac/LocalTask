// Package mcpserver exposes the taskboard domain to coding agents over MCP.
// It is a thin layer over the project package — the same service the HTTP
// API uses — so a human in the browser and an agent through MCP always see
// consistent state (see spec §17).
package mcpserver

import (
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/project"
	"localtask/internal/registry"
)

// instructions is shown to a connecting agent once, at session start (see
// spec §20, "Agent-Oriented Behavior"). It's the primary lever for getting
// an agent to actually use this server as intended rather than treating it
// as an optional extra — keep it concrete and workflow-shaped, not a
// feature list.
const instructions = `This server exposes a coding project's tickets, documentation, and read-only git status to you, a coding agent working in that project's repo.

Recommended workflow:
1. Call get_project_context first when starting work on a project. It returns the description, tickets currently in progress (with who holds them), blocked tickets, available documentation, and current git branch/status in one call — cheaper than reconstructing that from list_tickets.
2. Use one agent name for your whole session, and pass it as "agent" on every write, so a human watching the board can see who is doing what. Several agents and subagents may share this board at once. If your prompt gives you a name, use it. Otherwise call pick_agent_name once and keep the name it returns.
3. To find work, call next_ready_tickets (ready, unassigned, dependencies met) or search_tickets/list_tickets to check whether relevant work already exists before creating anything.
4. Before starting a ticket, call claim_ticket. If it fails because another agent holds it, leave it alone and pick another — don't work a ticket you haven't claimed.
5. Call get_ticket to read its full detail: acceptance criteria, dependencies, constraints, related docs/code, and the activity log (notes from whoever worked it before).
6. Do the work, leaving short progress notes via update_ticket(note=...) at meaningful milestones — what you finished, what you're stuck on, what the next agent needs to know.
7. When done, update_ticket to move status to "review" (or "done"), with a closing note. Mark acceptance criteria complete only once you've actually verified them. If you stop without finishing, call release_ticket with a note saying where you left off.
8. A ticket's "jira" field lists the Jira issues (e.g. PROJ-123) it is part of; get_project gives the Jira site URL. If you have Jira tools, read those issues for the wider goal. Add a ticket's Jira keys when you learn them.
9. If you discover follow-up work that's out of scope for the current ticket, call create_ticket for it instead of doing it inline. Keep changes scoped to what the ticket you're working describes.

If your prompt already names your ticket, an orchestrator (a workflow script, or a parent agent) assigned it to you. Then:
- Skip steps 1 and 3. Don't call get_project_context or next_ready_tickets; the orchestrator already chose your work.
- Claim that ticket (step 4) under the agent name your prompt gives, or a short name for your role. Read it (step 5) only for detail your prompt doesn't include.
- Notes and status changes (steps 6 and 7) still apply. They're how a human watching the board follows the run.
- Don't call create_ticket for follow-up work, unless your prompt says to. Report it in your result, so the orchestrator can merge duplicates across agents and file each one once.
- If your prompt says you're reviewing or verifying another agent's work, don't touch the board at all.

Use update_ticket/create_ticket for every change. They validate the result (unique IDs, valid enum values, resolvable dependencies) before writing, write atomically, and are serialized against every other agent and the web UI so no one's change is lost. A direct file write gets none of that.

The board is usually stored outside the project's repo, so the repo may have no taskboard.json. Never create one there, and don't add LocalTask notes or ignore rules to the repo. If the repo does have a taskboard.json, you may read it for quick context.`

// New builds an MCP server exposing tools and resources over the projects
// in reg. A lookup that misses re-reads the config first (see
// registry.Registry.Refresh), so a project added after this server started
// is found without a restart.
func New(reg *registry.Registry) *mcp.Server {
	s := &server{registry: reg}

	srv := mcp.NewServer(
		&mcp.Implementation{Name: "taskboard-mcp", Version: "0.1.0"},
		&mcp.ServerOptions{Instructions: instructions},
	)
	s.registerTools(srv)
	s.registerResources(srv)
	return srv
}

type server struct {
	registry *registry.Registry
}

func (s *server) project(name string) (*project.Project, error) {
	if p, ok := s.registry.Get(name); ok {
		return p, nil
	}
	s.refresh()
	p, ok := s.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("project %q not found", name)
	}
	return p, nil
}

// refresh picks up projects added to the config since this server started.
func (s *server) refresh() {
	for _, err := range s.registry.Refresh() {
		log.Printf("refreshing config: %v", err)
	}
}
