# Local Taskboard — Implementation Handoff

## 1. Project Goal

Build a lightweight, local-only Jira-like task/project tracker for software development projects.

The application should:

1. Run entirely locally.
2. Be accessible through `localhost`.
3. Read project/task information from local coding-project directories.
4. Render structured task data as a visually appealing Kanban/list interface.
5. Render project documentation alongside tasks.
6. Provide an MCP server so coding agents can read and modify project/task state.
7. Treat the local project files as the primary source of project/task information wherever practical.
8. Provide useful project progress metrics without requiring a cloud service.

This is **not intended to reproduce Jira**. The goal is a small, fast, developer-oriented task tracker that sits alongside coding projects and provides useful context to both humans and coding agents.

---

# 2. Core Design Principle

The system should act as a layer over a coding project rather than becoming a separate project-management system.

Example:

```text
my-project/
├── taskboard.json
├── README.md
├── docs/
│   ├── architecture.md
│   └── api.md
├── cmd/
├── internal/
└── ...
```

The taskboard application discovers/configures this project and provides:

```text
                 Taskboard
                     │
          ┌──────────┴──────────┐
          │                     │
       Browser                MCP
          │                     │
       Human                Coding Agent
          │                     │
          └──────────┬──────────┘
                     │
                Shared domain
                     │
          ┌──────────┴──────────┐
          │                     │
      taskboard.json       Project files
                              │
                       docs / source / git
```

The HTTP API and MCP server must share the same domain/service layer. Do not implement separate task-management logic for MCP and HTTP.

---

# 3. Recommended Technology

## Backend

Go.

Prefer the standard library where reasonable.

Recommended:

- `net/http`
- `encoding/json`
- `os`
- `path/filepath`
- `embed`
- `context`
- `database/sql` if SQLite is used

An HTTP framework is not required.

## Frontend

Use a lightweight web frontend.

React + TypeScript + Vite is acceptable.

Alternatively, server-rendered HTML/HTMX is acceptable if it significantly reduces complexity.

The UI should prioritize functionality and visual clarity over framework complexity.

## Storage

Start with the simplest viable storage model.

The project task file should be JSON.

SQLite may be introduced for application metadata, history, comments, analytics, etc., but it should not be required merely to read/write the basic task definitions.

Do not introduce PostgreSQL, Redis, or another server database.

## MCP

Use the official Go MCP SDK.

MCP should initially use stdio transport so coding agents can launch the MCP server locally.

---

# 4. Project Configuration

The application needs a way to discover projects.

For the first implementation, use a central configuration file.

Example:

```yaml
projects:
  - name: DanceNote
    path: /projects/dance-note

  - name: Job Tracker
    path: /projects/job-tracker
```

The exact configuration format can be chosen by the implementer. YAML is preferred for human readability.

Docker should be able to mount the configured project root.

Example conceptual Docker configuration:

```yaml
services:
  taskboard:
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./projects:/projects
```

The host-side port must bind specifically to `127.0.0.1`.

The application must not intentionally expose itself to the LAN or Internet.

---

# 5. Task File

Each project should contain:

```text
taskboard.json
```

Example:

```json
{
  "version": 1,
  "project": "DanceNote",
  "description": "AI-powered ballroom dance coaching platform",

  "tickets": [
    {
      "id": "DN-42",
      "title": "Implement video processing worker",
      "description": "Implement the worker that processes uploaded dance videos.",

      "type": "task",
      "status": "in_progress",
      "priority": "high",

      "parent": "DN-30",

      "depends_on": [
        "DN-38",
        "DN-40"
      ],

      "acceptance_criteria": [
        {
          "id": "DN-42-ac1",
          "text": "Worker claims pending processing jobs",
          "complete": true
        },
        {
          "id": "DN-42-ac2",
          "text": "Job status changes to processing",
          "complete": true
        },
        {
          "id": "DN-42-ac3",
          "text": "Successful processing marks the job complete",
          "complete": false
        }
      ],

      "constraints": [
        "Processing must be idempotent",
        "Do not introduce Redis"
      ],

      "tags": [
        "backend",
        "video"
      ],

      "related_docs": [
        "docs/video-processing.md"
      ],

      "related_code": [
        "server/internal/video/",
        "server/internal/jobs/"
      ]
    }
  ]
}
```

The exact Go structs may differ internally, but the JSON format should remain straightforward and human-editable.

---

# 6. Ticket Model

Tickets should support:

### Required

- `id`
- `title`
- `status`
- `priority`
- `type`

### Recommended

- `description`
- `parent`
- `depends_on`
- `acceptance_criteria`
- `constraints`
- `tags`
- `related_docs`
- `related_code`

## Ticket types

Initially support:

```text
epic
story
task
bug
subtask
```

## Statuses

Initially support:

```text
backlog
ready
in_progress
blocked
review
done
cancelled
```

The UI should allow these statuses to be represented as board columns.

## Priorities

Initially:

```text
low
medium
high
critical
```

---

# 7. Acceptance Criteria

Acceptance criteria are important because they make tasks measurable for both humans and coding agents.

Example:

```json
"acceptance_criteria": [
  {
    "id": "DN-42-ac1",
    "text": "Worker claims pending processing jobs",
    "complete": true
  },
  {
    "id": "DN-42-ac2",
    "text": "Failed jobs record an error",
    "complete": false
  }
]
```

The UI should display these as checkboxes.

Progress should be calculable:

```text
Acceptance criteria: 3 / 5
```

The system should not automatically mark criteria complete merely because a coding agent claims they are complete.

An agent may update the criteria, but the change should be explicit.

---

# 8. Project Manifest

In addition to `taskboard.json`, projects may contain project-level metadata.

Possible future file:

```text
taskboard.project.json
```

Example:

```json
{
  "name": "DanceNote",
  "description": "AI-powered ballroom dance coaching platform",

  "stack": [
    "Go",
    "PostgreSQL",
    "React"
  ],

  "docs": [
    "README.md",
    "docs/architecture.md"
  ],

  "commands": {
    "test": "go test ./...",
    "build": "go build ./...",
    "dev": "docker compose up"
  }
}
```

This is useful to MCP clients because an agent can request project context without having to infer the entire project from the filesystem.

This file is optional for v1.

---

# 9. Documentation

Markdown documentation should be treated as a first-class project resource.

The application should be able to:

1. Discover Markdown files.
2. Display them in the UI.
3. Associate documentation with tickets.
4. Allow MCP clients to retrieve documentation.

Example:

```json
"related_docs": [
  "docs/video-processing.md"
]
```

The UI should render Markdown rather than showing raw Markdown when viewing a document.

Do not implement a full documentation-management system.

The filesystem remains authoritative.

---

# 10. TODO Discovery

A later feature should scan project Markdown files for TODO markers.

Example:

```markdown
<!-- TODO: Add retry handling to video processing -->
```

or:

```markdown
TODO: Add retry handling to video processing
```

The system may surface these as potential tickets.

Do **not** automatically create tickets from arbitrary TODO comments in v1.

Potential future behavior:

```text
Documentation TODO
        │
        ▼
Potential ticket
        │
        ▼
Human/agent approves
        │
        ▼
Ticket created
```

---

# 11. Filesystem Watching

The application should eventually watch project directories for changes.

Important changes include:

```text
taskboard.json
Markdown documentation
project configuration
```

When `taskboard.json` changes externally:

1. Reload it.
2. Validate it.
3. Update in-memory state.
4. Notify connected clients if practical.

The application must not overwrite external changes silently.

Filesystem watching can be deferred until after the basic application works.

---

# 12. Web UI

The UI should have three primary views.

## Project Dashboard

Example:

```text
DanceNote

████████████████░░░░ 78%

18 / 23 complete

Ready        2
In Progress  2
Blocked      1
Review       0

High priority remaining: 2
```

Also show:

- recent tickets
- blocked tickets
- ready-to-work tickets
- project documentation
- major epics

---

# 13. Kanban Board

Primary ticket view:

```text
BACKLOG       READY        IN PROGRESS       REVIEW       DONE

┌─────────┐   ┌─────────┐  ┌─────────────┐   ┌────────┐
│ DN-51   │   │ DN-55   │  │ DN-42       │   │ DN-38  │
│ ...     │   │ ...     │  │ Video worker│   │ ...    │
└─────────┘   └─────────┘  └─────────────┘   └────────┘
```

Tickets should display at minimum:

- ID
- title
- priority
- type
- tags
- acceptance criteria progress

Drag-and-drop status changes are desirable but not mandatory for v1.

---

# 14. Ticket View

A ticket page should show:

```text
DN-42
Implement video processing worker

HIGH
IN PROGRESS

Description
...

Acceptance Criteria
☑ Claim pending jobs
☑ Update job status
☐ Handle failures
☐ Retry safely

Dependencies
✓ DN-38 Job schema
✓ DN-40 Upload pipeline

Related documentation
📄 docs/video-processing.md

Related code
server/internal/video/
server/internal/jobs/
```

The ticket page should make dependencies and completion state visually obvious.

---

# 15. List View

Provide a table/list view for projects with many tickets.

Columns:

```text
ID
Title
Status
Priority
Type
Progress
Tags
```

Filtering should eventually support:

- status
- priority
- type
- tag
- text search
- blocked
- parent/epic

---

# 16. Metrics

The application should calculate basic metrics from ticket data.

At minimum:

```text
Total tickets
Completed tickets
Open tickets
Blocked tickets
Ready tickets
In-progress tickets
Completion percentage
Acceptance-criteria completion
```

For an epic/project:

```text
23 total
18 complete
2 in progress
2 ready
1 blocked

78% complete
```

Avoid pretending these metrics are precise estimates of real-world project completion. They are indicators based on the task model.

---

# 17. MCP Server

Create a separate executable:

```text
taskboard-mcp
```

It should communicate over stdio.

The MCP server should use the same project/task services as the HTTP server.

Conceptually:

```text
             taskboard domain
                    │
         ┌──────────┴──────────┐
         │                     │
     HTTP handlers         MCP handlers
         │                     │
     Browser               Coding agent
```

Do not duplicate business logic.

---

# 18. MCP Tools

Initial tools should be small and focused.

## `list_projects`

Returns configured projects.

## `get_project`

Returns project metadata.

## `get_project_context`

Returns concise context useful to a coding agent.

Example:

```json
{
  "project": "DanceNote",
  "description": "...",
  "stack": ["Go", "PostgreSQL", "React"],
  "active_tickets": [...],
  "blocked_tickets": [...],
  "documentation": [...]
}
```

## `list_tickets`

Support filters where practical:

```text
status
priority
type
tag
parent
```

## `get_ticket`

Returns complete ticket information.

## `search_tickets`

Search titles/descriptions/tags.

## `create_ticket`

Create a ticket.

Required:

```text
title
```

Optional:

```text
description
type
status
priority
parent
dependencies
acceptance criteria
tags
related docs
related code
```

## `update_ticket`

Allow modification of ticket fields.

Updates must be explicit.

---

# 19. MCP Resources

Expose project information as MCP resources where supported.

Useful resource concepts:

```text
project://<project>
project://<project>/context
project://<project>/ticket/<ticket-id>
project://<project>/doc/<path>
```

The exact URI scheme may be adjusted to conform to MCP conventions.

The important point is that an agent should be able to retrieve:

- project context
- ticket context
- associated documentation

without needing to manually reconstruct it from many tool calls.

---

# 20. Agent-Oriented Behavior

The system should make it easy for an agent to perform this workflow:

```text
1. Identify current project
2. Get project context
3. Find relevant existing tickets
4. Read relevant ticket
5. Read associated documentation
6. Work on code
7. Update ticket status
8. Update acceptance criteria when appropriate
9. Create follow-up tickets for discovered work
```

Example:

```text
Agent:
  search_tickets("video processing")

Taskboard:
  DN-42 Implement video processing worker

Agent:
  get_ticket("DN-42")

Taskboard:
  ticket + acceptance criteria + dependencies + docs + code

Agent:
  read associated documentation

Agent:
  implements feature

Agent:
  update_ticket(
      "DN-42",
      status="review"
  )

Agent:
  create_ticket(
      title="Add retry handling to video processing",
      parent="DN-42",
      priority="medium"
  )
```

The MCP interface should make this workflow natural.

---

# 21. Git Integration

Git integration is desirable because this is a coding-project tool.

Do not make Git a prerequisite for the core application.

Eventually provide read-only project information such as:

```text
current branch
working tree status
recent commits
```

Potential future ticket information:

```text
DN-42

Branch:
feature/video-worker

Recent commits:
8fa91c2 Add worker skeleton
3c82ab1 Add processing job model
```

Do not implement automatic ticket-to-commit association in v1 unless it is trivial.

---

# 22. Local-Only Security Model

The application is intended for personal/local use.

Nevertheless:

- HTTP server should only be exposed through localhost.
- Docker port mapping should bind to `127.0.0.1`.
- MCP stdio should not require network access.
- Do not add authentication in v1.
- Do not add cloud synchronization.
- Do not send project contents to external services.
- Do not add telemetry.

Be careful when exposing filesystem paths through the API/MCP interface.

Paths should be constrained to configured project roots.

A ticket or MCP request must not be able to arbitrarily read:

```text
/etc/passwd
~/.ssh/*
```

or unrelated filesystem locations.

---

# 23. Data Validation

Validate `taskboard.json` when loading.

At minimum:

- required fields exist
- ticket IDs are unique
- referenced parent tickets exist
- dependencies reference existing tickets
- related documentation paths stay within the project
- related code paths stay within the project
- statuses are valid
- priorities are valid
- types are valid

Return useful validation errors.

Do not silently discard malformed tickets.

---

# 24. Persistence Rules

The project task file is human-editable.

Therefore:

- Preserve unknown JSON fields where practical.
- Avoid destructive rewrites.
- Write atomically.
- Do not overwrite a newer version of the file because of stale in-memory state.
- Keep JSON reasonably formatted/readable.

An update should preferably:

```text
write temporary file
       ↓
fsync if appropriate
       ↓
rename over original
```

rather than writing directly over the existing file.

---

# 25. Suggested Package Structure

A reasonable starting structure:

```text
taskboard/
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── mcp/
│       └── main.go
│
├── internal/
│   ├── project/
│   ├── ticket/
│   ├── document/
│   ├── storage/
│   ├── config/
│   └── git/
│
├── web/
│   ├── ...
│
├── testdata/
│   └── sample-project/
│       ├── taskboard.json
│       ├── README.md
│       └── docs/
│
├── Dockerfile
├── compose.yaml
└── README.md
```

The implementer may change this structure if there is a strong reason.

Avoid premature abstraction.

---

# 26. Implementation Phases

## Phase 1 — Domain model

Implement:

- project configuration
- taskboard JSON parsing
- ticket model
- validation
- ticket CRUD in memory
- JSON persistence

No UI required yet.

Acceptance test:

```text
Load project
→ parse taskboard.json
→ list tickets
→ modify ticket
→ save
→ reload
→ data is preserved
```

---

## Phase 2 — HTTP API

Implement endpoints for:

```text
GET    /api/projects
GET    /api/projects/:id
GET    /api/projects/:id/tickets
GET    /api/projects/:id/tickets/:ticket
POST   /api/projects/:id/tickets
PATCH  /api/projects/:id/tickets/:ticket
GET    /api/projects/:id/docs/...
```

Use JSON responses.

Keep handlers thin.

---

## Phase 3 — UI

Implement:

1. Project dashboard
2. Kanban board
3. Ticket detail
4. List view
5. Markdown document view

Focus on:

- readable typography
- clear hierarchy
- useful spacing
- obvious status/priority
- attractive cards
- useful progress indicators

Avoid trying to reproduce Jira's UI.

---

## Phase 4 — MCP

Implement:

```text
list_projects
get_project
get_project_context
list_tickets
get_ticket
search_tickets
create_ticket
update_ticket
```

Then add project/ticket/document resources.

Test MCP independently from the browser.

---

## Phase 5 — Filesystem integration

Add:

- filesystem watching
- automatic reload
- document discovery
- safe path handling

---

## Phase 6 — Git and metrics

Add:

- Git status
- current branch
- recent commits
- richer project metrics

These should remain read-only initially.

---

# 27. Non-Goals

Do NOT implement these unless specifically requested later:

- User accounts
- Authentication
- Multi-user collaboration
- Cloud hosting
- SaaS functionality
- Email
- Slack integration
- Jira synchronization
- GitHub synchronization
- Complex permissions
- Sprint planning
- Time tracking
- Story points
- Burndown charts
- AI-generated task decomposition
- Automatic code modification
- Automatic ticket completion based on Git commits

The system should remain small.

---

# 28. Design Philosophy

Prefer:

```text
simple > configurable
local > cloud
filesystem > proprietary database
explicit > magical
developer workflow > Jira compatibility
small API > giant API
```

The application should feel like a **developer tool**, not an enterprise project-management system.

The most important user experience is:

```text
Open project
      ↓
Immediately see what needs to be done
      ↓
Understand what is blocked
      ↓
Understand what is currently being worked on
      ↓
See measurable progress
      ↓
Open the relevant documentation
      ↓
Have a coding agent access exactly the same context
```

The MCP interface should make the taskboard useful to coding agents without making the human UI worse.

---

# 29. Definition of Done for v1

A v1 implementation is complete when:

- [ ] A project can be configured from a local directory.
- [ ] `taskboard.json` can be loaded and validated.
- [ ] Tickets can be viewed in a Kanban board.
- [ ] Tickets can be viewed in a list.
- [ ] Tickets have detail pages.
- [ ] Ticket status can be changed.
- [ ] Tickets can be created.
- [ ] Tickets can be edited.
- [ ] Acceptance criteria are displayed and editable.
- [ ] Basic project progress is displayed.
- [ ] Markdown documentation can be viewed.
- [ ] Tickets can reference documentation.
- [ ] Application runs entirely locally.
- [ ] Docker deployment works.
- [ ] Docker only exposes the application on localhost.
- [ ] MCP server can be launched locally.
- [ ] MCP can list/read projects.
- [ ] MCP can search/read tickets.
- [ ] MCP can create/update tickets.
- [ ] MCP and HTTP use the same domain/service logic.
- [ ] Project-root filesystem access is constrained.
- [ ] Basic automated tests exist for task loading, validation, CRUD, and persistence.

# 30. First Implementation Instruction

Do not attempt to implement the entire system at once.

Start with **Phase 1 only**.

Before adding the frontend or MCP server, establish a clean, tested domain/service layer for:

```text
Project
Ticket
AcceptanceCriterion
Taskboard
```

and operations:

```text
LoadProject
ListTickets
GetTicket
CreateTicket
UpdateTicket
DeleteTicket
SaveProject
ValidateProject
```

Once those work reliably, build the HTTP API on top of them. MCP should subsequently use the same services rather than bypassing them.