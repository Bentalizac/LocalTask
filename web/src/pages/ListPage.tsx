import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTickets } from '../hooks/useTickets'
import { PriorityBadge, StatusBadge, TypeBadge } from '../components/Badges'
import { TICKET_PRIORITIES, TICKET_STATUSES, TICKET_TYPES } from '../types'

export function ListPage() {
  const { project = '' } = useParams<{ project: string }>()
  const { tickets, error } = useTickets(project)
  const navigate = useNavigate()

  const [status, setStatus] = useState('')
  const [priority, setPriority] = useState('')
  const [type, setType] = useState('')
  const [tag, setTag] = useState('')
  const [search, setSearch] = useState('')
  const [parent, setParent] = useState('')

  const parents = useMemo(
    () => Array.from(new Set((tickets ?? []).map((t) => t.parent).filter((p): p is string => !!p))),
    [tickets],
  )

  const filtered = useMemo(() => {
    if (!tickets) return []
    const q = search.trim().toLowerCase()
    return tickets.filter((t) => {
      if (status && t.status !== status) return false
      if (priority && t.priority !== priority) return false
      if (type && t.type !== type) return false
      if (parent && t.parent !== parent) return false
      if (tag && !(t.tags ?? []).includes(tag)) return false
      if (q && !(t.title.toLowerCase().includes(q) || t.id.toLowerCase().includes(q) || (t.description ?? '').toLowerCase().includes(q))) {
        return false
      }
      return true
    })
  }, [tickets, status, priority, type, parent, tag, search])

  const allTags = useMemo(
    () => Array.from(new Set((tickets ?? []).flatMap((t) => t.tags ?? []))),
    [tickets],
  )

  if (error) return <div className="page"><div className="error-banner">{error}</div></div>
  if (!tickets) return <div className="page"><div className="loading">Loading…</div></div>

  return (
    <div className="page">
      <div className="page-header">
        <h1>All tickets</h1>
      </div>

      <div className="filters-bar">
        <input
          type="search"
          placeholder="Search title, id, description…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">All statuses</option>
          {TICKET_STATUSES.map((s) => (
            <option key={s} value={s}>
              {s.replace('_', ' ')}
            </option>
          ))}
        </select>
        <select value={priority} onChange={(e) => setPriority(e.target.value)}>
          <option value="">All priorities</option>
          {TICKET_PRIORITIES.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
        <select value={type} onChange={(e) => setType(e.target.value)}>
          <option value="">All types</option>
          {TICKET_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        {allTags.length > 0 && (
          <select value={tag} onChange={(e) => setTag(e.target.value)}>
            <option value="">All tags</option>
            {allTags.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        )}
        {parents.length > 0 && (
          <select value={parent} onChange={(e) => setParent(e.target.value)}>
            <option value="">All parents</option>
            {parents.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        )}
      </div>

      <table className="ticket-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Title</th>
            <th>Status</th>
            <th>Priority</th>
            <th>Type</th>
            <th>Progress</th>
            <th>Tags</th>
          </tr>
        </thead>
        <tbody>
          {filtered.map((t) => {
            const criteria = t.acceptance_criteria ?? []
            const complete = criteria.filter((c) => c.complete).length
            return (
              <tr
                key={t.id}
                onClick={() =>
                  navigate(`/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(t.id)}`)
                }
              >
                <td className="ticket-table-id">{t.id}</td>
                <td>{t.title}</td>
                <td>
                  <StatusBadge status={t.status} />
                </td>
                <td>
                  <PriorityBadge priority={t.priority} />
                </td>
                <td>
                  <TypeBadge type={t.type} />
                </td>
                <td>{criteria.length > 0 ? `${complete}/${criteria.length}` : '—'}</td>
                <td>{(t.tags ?? []).join(', ') || '—'}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
      {filtered.length === 0 && <div className="empty-note">No tickets match these filters.</div>}
    </div>
  )
}
