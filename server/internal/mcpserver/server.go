// Package mcpserver exposes the taskboard domain to coding agents over MCP.
// It is a thin layer over the project package — the same service the HTTP
// API uses — so a human in the browser and an agent through MCP always see
// consistent state (see spec §17).
package mcpserver

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/project"
)

// instructions is shown to a connecting agent once, at session start (see
// spec §20, "Agent-Oriented Behavior"). It's the primary lever for getting
// an agent to actually use this server as intended rather than treating it
// as an optional extra — keep it concrete and workflow-shaped, not a
// feature list.
const instructions = `This server exposes a coding project's tickets, documentation, and read-only git status to you, a coding agent working in that project's repo.

Recommended workflow:
1. Call get_project_context first when starting work on a project. It returns the description, tickets currently in progress, blocked tickets, available documentation, and current git branch/status in one call — cheaper than reconstructing that from list_tickets.
2. Before starting something new, call search_tickets or list_tickets (filterable by status/priority/type/tag/parent) to check whether relevant work already exists.
3. Call get_ticket to read a ticket's full detail before working on it: acceptance criteria, dependencies, constraints, and related docs/code.
4. Do the work.
5. Update the ticket as you go via update_ticket: move status forward (e.g. to "in_progress", then "review" when done), and mark acceptance criteria complete only once you've actually verified them — not just because they describe the ticket's goal.
6. If you discover follow-up work that's out of scope for the current ticket, call create_ticket for it instead of doing it inline. Keep changes scoped to what the ticket you're working describes.

taskboard.json lives in the project's own repo and you can read it directly for quick context — that's fine. But prefer update_ticket/create_ticket over hand-editing it for any change: they validate the result (unique IDs, valid enum values, resolvable dependencies) before writing, write atomically, and refuse to silently overwrite a concurrent edit from the web UI or another session. A direct file write gets none of that.`

// New builds an MCP server exposing tools and resources over the given
// projects, keyed by configured name.
func New(projects map[string]*project.Project) *mcp.Server {
	s := &server{projects: projects}

	srv := mcp.NewServer(
		&mcp.Implementation{Name: "taskboard-mcp", Version: "0.1.0"},
		&mcp.ServerOptions{Instructions: instructions},
	)
	s.registerTools(srv)
	s.registerResources(srv)
	return srv
}

type server struct {
	projects map[string]*project.Project
}

func (s *server) project(name string) (*project.Project, error) {
	p, ok := s.projects[name]
	if !ok {
		return nil, fmt.Errorf("project %q not found", name)
	}
	return p, nil
}
