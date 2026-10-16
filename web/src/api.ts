import type { ApiErrorBody, BrowseListing, Doc, GitInfo, ProjectStorage, ProjectSummary, Ticket } from './types'

export class ApiError extends Error {
  status: number
  issues?: string[]

  constructor(status: number, body: ApiErrorBody) {
    super(body.error)
    this.status = status
    this.issues = body.issues
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({ error: res.statusText }))) as ApiErrorBody
    throw new ApiError(res.status, body)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export function listProjects(): Promise<ProjectSummary[]> {
  return request('/api/projects')
}

export function browseDirectory(path?: string): Promise<BrowseListing> {
  const qs = path ? `?path=${encodeURIComponent(path)}` : ''
  return request(`/api/browse${qs}`)
}

export function addProject(name: string, path: string, storage: ProjectStorage): Promise<ProjectSummary> {
  return request('/api/projects', {
    method: 'POST',
    body: JSON.stringify({ name, path, storage }),
  })
}

export function getProject(project: string): Promise<ProjectSummary> {
  return request(`/api/projects/${encodeURIComponent(project)}`)
}

export function listTickets(project: string, filters?: { parent?: string }): Promise<Ticket[]> {
  const qs = filters?.parent ? `?parent=${encodeURIComponent(filters.parent)}` : ''
  return request(`/api/projects/${encodeURIComponent(project)}/tickets${qs}`)
}

export function getTicket(project: string, ticketId: string): Promise<Ticket> {
  return request(
    `/api/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(ticketId)}`,
  )
}

export type CreateTicketInput = Partial<Ticket> & { title: string }

export function createTicket(project: string, input: CreateTicketInput): Promise<Ticket> {
  return request(`/api/projects/${encodeURIComponent(project)}/tickets`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export type TicketPatch = Partial<Omit<Ticket, 'id' | 'activity' | 'updated_at'>> & {
  // Appended to the ticket's activity log rather than stored as a field.
  note?: string
}

export function updateTicket(
  project: string,
  ticketId: string,
  patch: TicketPatch,
): Promise<Ticket> {
  return request(
    `/api/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(ticketId)}`,
    { method: 'PATCH', body: JSON.stringify(patch) },
  )
}

export function getGit(project: string): Promise<GitInfo> {
  return request(`/api/projects/${encodeURIComponent(project)}/git`)
}

export function listDocs(project: string): Promise<string[]> {
  return request(`/api/projects/${encodeURIComponent(project)}/docs`)
}

export function getDoc(project: string, path: string): Promise<Doc> {
  return request(
    `/api/projects/${encodeURIComponent(project)}/docs/${path.split('/').map(encodeURIComponent).join('/')}`,
  )
}
