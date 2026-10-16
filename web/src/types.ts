export type TicketType = 'epic' | 'story' | 'task' | 'bug' | 'subtask'

export type TicketStatus =
  | 'backlog'
  | 'ready'
  | 'in_progress'
  | 'blocked'
  | 'review'
  | 'done'
  | 'cancelled'

export type TicketPriority = 'low' | 'medium' | 'high' | 'critical'

export const TICKET_TYPES: TicketType[] = ['epic', 'story', 'task', 'bug', 'subtask']
export const TICKET_STATUSES: TicketStatus[] = [
  'backlog',
  'ready',
  'in_progress',
  'blocked',
  'review',
  'done',
  'cancelled',
]
export const TICKET_PRIORITIES: TicketPriority[] = ['low', 'medium', 'high', 'critical']

// The columns shown on the Kanban board (spec §13). Cancelled tickets are
// intentionally left off the board and only appear in the List view.
export const BOARD_COLUMNS: TicketStatus[] = [
  'backlog',
  'ready',
  'in_progress',
  'blocked',
  'review',
  'done',
]

export interface AcceptanceCriterion {
  id: string
  text: string
  complete: boolean
}

export interface Ticket {
  id: string
  title: string
  description?: string
  type: TicketType
  status: TicketStatus
  priority: TicketPriority
  parent?: string
  depends_on?: string[]
  acceptance_criteria?: AcceptanceCriterion[]
  constraints?: string[]
  tags?: string[]
  related_docs?: string[]
  related_code?: string[]
  assignee?: string
  updated_at?: string
  activity?: ActivityEntry[]
}

export type ActivityKind = 'created' | 'status' | 'assignee' | 'note'

export interface ActivityEntry {
  at: string
  author?: string
  kind: ActivityKind
  text: string
}

export interface ProjectMetrics {
  total: number
  backlog: number
  ready: number
  in_progress: number
  blocked: number
  review: number
  done: number
  cancelled: number
  completion_percent: number
  acceptance_criteria_complete: number
  acceptance_criteria_total: number
}

export interface ProjectSummary {
  name: string
  description?: string
  version: number
  metrics: ProjectMetrics
}

export interface Doc {
  path: string
  content: string
}

export interface BrowseEntry {
  name: string
  path: string
  has_taskboard: boolean
}

export interface BrowseListing {
  path: string
  parent?: string
  has_taskboard: boolean
  entries: BrowseEntry[]
}

export interface GitCommit {
  hash: string
  short_hash: string
  subject: string
  author: string
  date: string
}

export interface GitInfo {
  is_repo: boolean
  branch?: string
  clean: boolean
  changed_files: number
  recent_commits?: GitCommit[]
}

export interface ApiErrorBody {
  error: string
  issues?: string[]
}

// Where a project's taskboard.json lives: outside the project (the
// default, so LocalTask writes nothing into the repo) or in its root.
export type ProjectStorage = 'external' | 'repo'
