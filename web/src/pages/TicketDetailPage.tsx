import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getTicket, listTickets, updateTicket } from '../api'
import { AssigneeChip, PriorityBadge, StatusBadge, TypeBadge } from '../components/Badges'
import { usePolling } from '../hooks/usePolling'
import { useJiraBaseUrl } from '../hooks/useJiraBaseUrl'
import { JiraKeys } from '../components/JiraKeys'
import { parseJiraKeys } from '../jira'
import { relativeTime } from '../time'
import { TICKET_PRIORITIES, TICKET_STATUSES, TICKET_TYPES } from '../types'
import type { Ticket } from '../types'

export function TicketDetailPage() {
  const { project = '', ticketId = '' } = useParams<{ project: string; ticketId: string }>()
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<{
    title: string
    description: string
    tags: string
    jira: string
    constraints: string
  } | null>(null)
  const [saving, setSaving] = useState(false)
  const [children, setChildren] = useState<Ticket[]>([])
  const [note, setNote] = useState('')
  const [addingNote, setAddingNote] = useState(false)
  const jiraBaseUrl = useJiraBaseUrl(project)

  function load() {
    setError(null)
    getTicket(project, ticketId)
      .then(setTicket)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load ticket'))
  }

  useEffect(() => {
    setTicket(null)
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project, ticketId])

  // Keep the page live while an agent works the ticket. Paused while
  // editing so a refresh can't shift the form out from under the user.
  usePolling(() => {
    getTicket(project, ticketId)
      .then(setTicket)
      .catch(() => {})
  }, !editing)

  useEffect(() => {
    setChildren([])
    listTickets(project, { parent: ticketId })
      .then(setChildren)
      .catch(() => setChildren([]))
  }, [project, ticketId])

  async function patch(fields: Parameters<typeof updateTicket>[2]) {
    if (!ticket) return
    const updated = await updateTicket(project, ticket.id, fields)
    setTicket(updated)
  }

  async function toggleCriterion(index: number) {
    if (!ticket) return
    const criteria = (ticket.acceptance_criteria ?? []).map((c, i) =>
      i === index ? { ...c, complete: !c.complete } : c,
    )
    try {
      await patch({ acceptance_criteria: criteria })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update acceptance criteria')
    }
  }

  async function addNote() {
    const text = note.trim()
    if (!text) return
    setAddingNote(true)
    try {
      await patch({ note: text })
      setNote('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add note')
    } finally {
      setAddingNote(false)
    }
  }

  function startEditing() {
    if (!ticket) return
    setDraft({
      title: ticket.title,
      description: ticket.description ?? '',
      tags: (ticket.tags ?? []).join(', '),
      jira: (ticket.jira ?? []).join(', '),
      constraints: (ticket.constraints ?? []).join('\n'),
    })
    setEditing(true)
  }

  async function saveEditing() {
    if (!draft) return
    setSaving(true)
    setError(null)
    try {
      await patch({
        title: draft.title.trim(),
        description: draft.description.trim() || undefined,
        tags: draft.tags.split(',').map((t) => t.trim()).filter(Boolean),
        jira: parseJiraKeys(draft.jira),
        constraints: draft.constraints.split('\n').map((c) => c.trim()).filter(Boolean),
      })
      setEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save ticket')
    } finally {
      setSaving(false)
    }
  }

  if (error && !ticket) return <div className="page"><div className="error-banner">{error}</div></div>
  if (!ticket) return <div className="page"><div className="loading">Loading…</div></div>

  const criteria = ticket.acceptance_criteria ?? []
  const acComplete = criteria.filter((c) => c.complete).length

  return (
    <div className="page">
      {error && <div className="error-banner" style={{ marginBottom: 16 }}>{error}</div>}

      <div className="detail-header">
        <div>
          <div className="detail-id">{ticket.id}</div>
          {editing ? (
            <input
              className="title-input"
              value={draft?.title ?? ''}
              onChange={(e) => setDraft((d) => (d ? { ...d, title: e.target.value } : d))}
            />
          ) : (
            <h1 className="detail-title">{ticket.title}</h1>
          )}
          <div className="ticket-card-badges" style={{ marginTop: 8 }}>
            <StatusBadge status={ticket.status} />
            <PriorityBadge priority={ticket.priority} />
            <TypeBadge type={ticket.type} />
          </div>
        </div>
        {editing ? (
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn" onClick={() => setEditing(false)} disabled={saving}>
              Cancel
            </button>
            <button className="btn btn-primary" onClick={saveEditing} disabled={saving}>
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        ) : (
          <button className="btn" onClick={startEditing}>
            Edit
          </button>
        )}
      </div>

      <div className="detail-grid">
        <div>
          <div className="detail-section">
            <h2 className="section-title">Description</h2>
            {editing ? (
              <textarea
                className="description-input"
                value={draft?.description ?? ''}
                onChange={(e) => setDraft((d) => (d ? { ...d, description: e.target.value } : d))}
              />
            ) : (
              <p>{ticket.description || <span className="empty-note">No description.</span>}</p>
            )}
          </div>

          <div className="detail-section">
            <h2 className="section-title">
              Acceptance criteria {criteria.length > 0 && `(${acComplete}/${criteria.length})`}
            </h2>
            {criteria.length === 0 ? (
              <div className="empty-note">No acceptance criteria defined.</div>
            ) : (
              criteria.map((c, i) => (
                <label className={`ac-item ${c.complete ? 'complete' : ''}`} key={c.id || i}>
                  <input type="checkbox" checked={c.complete} onChange={() => toggleCriterion(i)} />
                  <span className="ac-text">{c.text}</span>
                </label>
              ))
            )}
          </div>

          <div className="detail-section">
            <h2 className="section-title">
              Children {children.length > 0 && `(${children.filter((c) => c.status === 'done').length}/${children.length} done)`}
            </h2>
            {children.length === 0 ? (
              <div className="empty-note">No child tickets.</div>
            ) : (
              children.map((child) => (
                <Link
                  key={child.id}
                  to={`/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(child.id)}`}
                  className="mini-ticket-row"
                >
                  <span className="mini-ticket-id">{child.id}</span>
                  <span className="mini-ticket-title">{child.title}</span>
                  <StatusBadge status={child.status} />
                </Link>
              ))
            )}
          </div>

          <div className="detail-section">
            <h2 className="section-title">Dependencies</h2>
            {(ticket.depends_on ?? []).length === 0 ? (
              <div className="empty-note">No dependencies.</div>
            ) : (
              (ticket.depends_on ?? []).map((depId) => (
                <DependencyRow key={depId} project={project} depId={depId} />
              ))
            )}
          </div>

          <div className="detail-section">
            <h2 className="section-title">Activity</h2>
            <div className="note-form">
              <textarea
                className="description-input"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Leave a note for the agent working this ticket…"
                rows={2}
              />
              <button className="btn btn-sm" onClick={addNote} disabled={addingNote || !note.trim()}>
                {addingNote ? 'Adding…' : 'Add note'}
              </button>
            </div>
            {(ticket.activity ?? []).length === 0 ? (
              <div className="empty-note">No activity yet.</div>
            ) : (
              <ol className="activity-feed">
                {[...(ticket.activity ?? [])].reverse().map((e, i) => (
                  <li key={`${e.at}-${i}`} className={`activity-item kind-${e.kind}`}>
                    <div className="activity-item-head">
                      {e.author ? <AssigneeChip name={e.author} /> : <span className="empty-note">unknown</span>}
                      <span className="activity-time" title={e.at}>
                        {relativeTime(e.at)}
                      </span>
                    </div>
                    <div className="activity-text">{e.text}</div>
                  </li>
                ))}
              </ol>
            )}
          </div>

          <div className="detail-section">
            <h2 className="section-title">Constraints</h2>
            {editing ? (
              <textarea
                className="description-input"
                value={draft?.constraints ?? ''}
                onChange={(e) => setDraft((d) => (d ? { ...d, constraints: e.target.value } : d))}
                placeholder="One constraint per line"
              />
            ) : (ticket.constraints ?? []).length === 0 ? (
              <div className="empty-note">No constraints.</div>
            ) : (
              <ul>
                {(ticket.constraints ?? []).map((c) => (
                  <li key={c}>{c}</li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div>
          <div className="card card-pad detail-section">
            <h2 className="section-title">Details</h2>
            <div className="field-row">
              <span className="field-label">Status</span>
              <select value={ticket.status} onChange={(e) => patch({ status: e.target.value as Ticket['status'] })}>
                {TICKET_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {s.replace('_', ' ')}
                  </option>
                ))}
              </select>
            </div>
            <div className="field-row">
              <span className="field-label">Priority</span>
              <select value={ticket.priority} onChange={(e) => patch({ priority: e.target.value as Ticket['priority'] })}>
                {TICKET_PRIORITIES.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div className="field-row">
              <span className="field-label">Type</span>
              <select value={ticket.type} onChange={(e) => patch({ type: e.target.value as Ticket['type'] })}>
                {TICKET_TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            <div className="field-row">
              <span className="field-label">Assignee</span>
              {ticket.assignee ? (
                <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <AssigneeChip name={ticket.assignee} />
                  <button className="btn btn-sm" onClick={() => patch({ assignee: '' })} title="Release this ticket">
                    Unassign
                  </button>
                </span>
              ) : (
                <span className="empty-note">—</span>
              )}
            </div>
            {ticket.updated_at && (
              <div className="field-row">
                <span className="field-label">Updated</span>
                <span title={ticket.updated_at}>{relativeTime(ticket.updated_at)}</span>
              </div>
            )}
            {ticket.parent && (
              <div className="field-row">
                <span className="field-label">Parent</span>
                <Link to={`/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(ticket.parent)}`}>
                  {ticket.parent}
                </Link>
              </div>
            )}
          </div>

          <div className="card card-pad detail-section">
            <h2 className="section-title">Tags</h2>
            {editing ? (
              <input
                className="title-input"
                value={draft?.tags ?? ''}
                onChange={(e) => setDraft((d) => (d ? { ...d, tags: e.target.value } : d))}
                placeholder="comma, separated, tags"
              />
            ) : (ticket.tags ?? []).length === 0 ? (
              <div className="empty-note">No tags.</div>
            ) : (
              <div className="ticket-card-badges">
                {(ticket.tags ?? []).map((t) => (
                  <span className="tag" key={t}>
                    {t}
                  </span>
                ))}
              </div>
            )}
          </div>

          <div className="card card-pad detail-section">
            <h2 className="section-title">Jira</h2>
            {editing ? (
              <input
                className="title-input"
                value={draft?.jira ?? ''}
                onChange={(e) => setDraft((d) => (d ? { ...d, jira: e.target.value } : d))}
                placeholder="PROJ-123, PROJ-150"
              />
            ) : (ticket.jira ?? []).length === 0 ? (
              <div className="empty-note">No linked Jira issues.</div>
            ) : (
              <div className="ticket-card-badges">
                <JiraKeys keys={ticket.jira} baseUrl={jiraBaseUrl} />
              </div>
            )}
          </div>

          <div className="card card-pad detail-section">
            <h2 className="section-title">Related documentation</h2>
            {(ticket.related_docs ?? []).length === 0 ? (
              <div className="empty-note">None.</div>
            ) : (
              (ticket.related_docs ?? []).map((d) => (
                <Link
                  key={d}
                  className="doc-link-item"
                  to={`/projects/${encodeURIComponent(project)}/docs/${d}`}
                  style={{ display: 'block' }}
                >
                  📄 {d}
                </Link>
              ))
            )}
          </div>

          <div className="card card-pad detail-section">
            <h2 className="section-title">Related code</h2>
            {(ticket.related_code ?? []).length === 0 ? (
              <div className="empty-note">None.</div>
            ) : (
              (ticket.related_code ?? []).map((c) => (
                <div className="code-path-item" key={c}>
                  {c}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function DependencyRow({ project, depId }: { project: string; depId: string }) {
  const [dep, setDep] = useState<Ticket | null>(null)
  useEffect(() => {
    getTicket(project, depId).then(setDep).catch(() => setDep(null))
  }, [project, depId])

  const done = dep?.status === 'done'
  return (
    <Link className="dep-item" to={`/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(depId)}`}>
      <span className={`dep-check ${done ? 'ok' : 'pending'}`}>{done ? '✓' : '○'}</span>
      <span>{depId}</span>
      {dep && <span className="empty-note">{dep.title}</span>}
    </Link>
  )
}
