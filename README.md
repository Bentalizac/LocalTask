# LocalTask

A lightweight, local-only Jira-like task tracker for software projects. It runs
entirely on your machine, reads and writes plain JSON (`taskboard.json`) for
each project, and exposes that data to both a web UI and
coding agents (via [MCP](https://modelcontextprotocol.io)) through the same
underlying service — so a human in the browser and an agent working through
MCP always see consistent state.

This is **not** an attempt to reproduce Jira. It's a small, fast,
developer-oriented layer over your existing coding projects: open a project,
immediately see what needs doing, what's blocked, what's in progress, and
give a coding agent access to exactly the same context.

## Quick start (Docker)

```sh
git clone https://github.com/Bentalizac/LocalTask.git
cd LocalTask
mkdir -p projects
cat > projects/taskboard.config.yaml <<'EOF'
projects:
  - name: MyProject
    path: /projects/my-project
EOF
docker compose up -d --build
```

Then open **http://taskboard.localhost:8420** (any `*.localhost` hostname
resolves to `127.0.0.1` automatically — no `/etc/hosts` edit needed).

`projects/` on your host is bind-mounted to `/projects` in the container (see
`compose.yaml`) — put project subdirectories there, or mount other host paths
in the same file if your projects live elsewhere. Each entry in
`taskboard.config.yaml` points at one, by its container-side path.

### Where each board lives

By default, LocalTask writes nothing into a project's directory. Each board is
stored at `projects/boards/<name>/taskboard.json`, next to
`taskboard.config.yaml`, and is created empty the first time it's needed. Use
this for repos you don't want LocalTask to touch, such as work repos.

To commit a board with its project instead, set `storage: repo` on that
project. LocalTask then reads and writes `taskboard.json` at the project root:

```yaml
jira_base_url: https://your-site.atlassian.net   # optional; see "Jira links"
projects:
  - name: WorkService
    path: /code/work-service          # board in projects/boards/WorkService/
  - name: SideProject
    path: /code/side-project
    storage: repo                     # board in /code/side-project/taskboard.json
```

A `storage: repo` project must already have a `taskboard.json`, or it fails to
load. The "Add Project" button in the UI has a checkbox for the same choice,
and creates the board for you. Project names can't contain `/` or `\`,
because an external board's directory is named after its project.

In both modes, LocalTask only reads the project itself: its Markdown docs and
its git branch, status, and log. Git runs with `GIT_OPTIONAL_LOCKS=0`, so
reading status never rewrites `.git/index`.

The host port (`8420` by default) only binds to `127.0.0.1` — the app is
never exposed beyond localhost. See `compose.yaml` for the full explanation
of why the container itself binds `0.0.0.0` internally without weakening that.

## Local development (no Docker)

Backend:

```sh
cd server
go run ./cmd/server -config ../projects/taskboard.config.yaml -port 8420
```

Frontend (proxies `/api` to the Go server above):

```sh
cd web
npm install
LOCALTASK_API_URL=http://127.0.0.1:8420 npm run dev
```

Run the backend test suite with `go test ./...` from `server/`.

## MCP server

A separate binary, `taskboard-mcp`, exposes the same project/ticket data to a
coding agent over stdio:

```sh
cd server
go build -o taskboard-mcp ./cmd/mcp
```

Point your MCP client at the built binary with `-config` pointing to your
`taskboard.config.yaml`. Tools: `list_projects`, `get_project`,
`get_project_context`, `list_tickets`, `get_ticket`, `search_tickets`,
`next_ready_tickets`, `claim_ticket`, `release_ticket`, `create_ticket`,
`update_ticket`, `pick_agent_name`.

### Multiple agents

Any number of agents (separate sessions, each with its own MCP server
process, plus their subagents) can share a board. Every write takes a lock
on the board's directory, re-reads `taskboard.json`, applies only its own
change, and writes it back, so concurrent changes are never lost.
`claim_ticket` atomically assigns a ticket to the calling agent, so two
agents can't both pick up the same work. Agents pass an `agent` name on
writes. An agent whose prompt gives it no name calls `pick_agent_name` for a
random Wheel of Time character name that nobody on the board holds. Status/assignee changes and agents' `note`s go into each ticket's
activity log, which the web UI's **Activity** page shows live. That page
is laid out to work in a narrow side pane next to your agent sessions. Resources: `project://<name>`,
`project://<name>/context`, `project://<name>/ticket/<id>`,
`project://<name>/doc/<path>`.

### Using it from a coding agent's `CLAUDE.md`

For a `storage: repo` project, drop this into the repo's `CLAUDE.md`, filling
in the placeholders. For a repo you don't want to change, skip it, or put it
in your user-level `~/.claude/CLAUDE.md` instead. The MCP server sends its own
workflow instructions either way.

```markdown
## Task tracking

This project's tasks live in `taskboard.json` (plus any Markdown under `docs/`),
tracked with [LocalTask](https://github.com/Bentalizac/LocalTask), running locally at
http://taskboard.localhost:8420. Project name in LocalTask: `<PROJECT_NAME>` —
if unsure, call `list_projects` via MCP or check `taskboard.config.yaml` in the
LocalTask deployment.

- **Before starting non-trivial work**, check for an existing ticket covering it
  (`search_tickets`/`list_tickets` via MCP, or read `taskboard.json` directly).
  Don't duplicate tracked work.
- **As you work**, keep the ticket's status and acceptance criteria current —
  move it to `in_progress`, then `review` when done. Only check off an
  acceptance criterion once you've actually verified it.
- **Discovered follow-up work** (bugs found, scope you're deliberately not
  doing now) gets filed as a new ticket, not done inline and not silently
  dropped.
- **If the `taskboard-mcp` MCP server is connected**, use its tools
  (`create_ticket`, `update_ticket`) for any change — it validates the result
  and writes atomically, which a direct edit to `taskboard.json` doesn't.
  **If it isn't connected**, read/edit `taskboard.json` directly; it's plain
  JSON for exactly this reason. Match the existing ticket shape (`id`, `title`,
  `type`, `status`, `priority`; see other entries in the file for the rest).
- Don't invent new `status`/`priority`/`type` values — use what's already in
  the file's existing tickets.
- You can point the user at http://taskboard.localhost:8420 to see the board
  in a browser — useful to mention after a batch of ticket updates.
```

The `taskboard-mcp` server also sends its own workflow instructions to any
connecting agent automatically (see `internal/mcpserver/server.go`) — the
snippet above is a complement to that, not a replacement.

## Repository layout

```
server/           Go backend
  cmd/server/      HTTP API + web UI (Docker's entrypoint)
  cmd/mcp/         MCP server (stdio)
  internal/        domain logic, shared by both binaries
  testdata/        fixtures used by the test suite
web/               React + TypeScript + Vite frontend
compose.yaml       Docker deployment
Dockerfile         multi-stage build (frontend + both Go binaries)
```

## Ticket model

Tickets support `id`, `title`, `status`, `priority`, `type` (required), plus
`description`, `parent`, `depends_on`, `acceptance_criteria`, `constraints`,
`tags`, `related_docs`, `related_code`, and `jira`. Statuses:
`backlog`/`ready`/`in_progress`/`blocked`/`review`/`done`/`cancelled`.
Priorities: `low`/`medium`/`high`/`critical`. Types:
`epic`/`story`/`task`/`bug`/`subtask`.

## Security model

Local-only by design: no authentication, no cloud sync, no telemetry, no
outbound requests. The HTTP server binds `127.0.0.1` directly when run
outside Docker; in Docker, the host-side port publish (`127.0.0.1:...` in
`compose.yaml`) is what keeps it off the LAN. Reads/writes of a project's
`taskboard.json` and docs are constrained to that project's own root — the
one deliberate exception is the "Add Project" directory browser, which can
list any directory the server process can read, by design (see
`internal/browse`), since its entire purpose is finding a new project
before it's configured.

## Jira links

A ticket's `jira` field lists the keys of the Jira issues it's part of, such
as `["RT-1234", "RT-1250"]`. Keys show as chips on the board, the list, and
the ticket page. Ticket search matches them too.

Set `jira_base_url` at the top of `taskboard.config.yaml` to make each chip
link to `<jira_base_url>/browse/<key>`. A project can override it with its
own `jira_base_url`. Without one, keys show unlinked.

LocalTask never calls Jira. It needs no credentials and doesn't sync Jira
status. Agents get the keys and the base URL over MCP, so an agent with Jira
tools can look up the issue behind a task.
