import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTickets } from '../hooks/useTickets'
import { getGit, listDocs } from '../api'
import type { GitInfo, Ticket } from '../types'

function MiniList({ project, tickets, emptyLabel }: { project: string; tickets: Ticket[]; emptyLabel: string }) {
  if (tickets.length === 0) {
    return <div className="empty-note">{emptyLabel}</div>
  }
  return (
    <div>
      {tickets.map((t) => (
        <Link
          key={t.id}
          to={`/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(t.id)}`}
          className="mini-ticket-row"
        >
          <span className="mini-ticket-id">{t.id}</span>
          <span className="mini-ticket-title">{t.title}</span>
        </Link>
      ))}
    </div>
  )
}

export function DashboardPage() {
  const { project = '' } = useParams<{ project: string }>()
  const { tickets, error } = useTickets(project)
  const [docs, setDocs] = useState<string[]>([])
  const [git, setGit] = useState<GitInfo | null>(null)

  useEffect(() => {
    setDocs([])
    setGit(null)
    listDocs(project).then(setDocs).catch(() => setDocs([]))
    getGit(project).then(setGit).catch(() => setGit(null))
  }, [project])

  if (error) return <div className="page"><div className="error-banner">{error}</div></div>
  if (!tickets) return <div className="page"><div className="loading">Loading…</div></div>

  const total = tickets.length
  const done = tickets.filter((t) => t.status === 'done').length
  const pct = total > 0 ? Math.round((done / total) * 100) : 0

  const counts: Record<string, number> = {}
  for (const t of tickets) counts[t.status] = (counts[t.status] ?? 0) + 1

  const highRemaining = tickets.filter(
    (t) => (t.priority === 'high' || t.priority === 'critical') && t.status !== 'done' && t.status !== 'cancelled',
  ).length

  const blocked = tickets.filter((t) => t.status === 'blocked')
  const ready = tickets.filter((t) => t.status === 'ready')
  const epics = tickets.filter((t) => t.type === 'epic' && t.status !== 'done')
  const recent = [...tickets].slice(-6).reverse()

  return (
    <div className="page">
      <div className="page-header">
        <h1>{project}</h1>
      </div>

      <div className="metrics-grid">
        <div className="card metric-tile">
          <div className="metric-value">
            {done} / {total}
          </div>
          <div className="metric-label">Complete</div>
        </div>
        <div className="card metric-tile">
          <div className="metric-value">{counts.ready ?? 0}</div>
          <div className="metric-label">Ready</div>
        </div>
        <div className="card metric-tile">
          <div className="metric-value">{counts.in_progress ?? 0}</div>
          <div className="metric-label">In progress</div>
        </div>
        <div className="card metric-tile">
          <div className="metric-value">{counts.blocked ?? 0}</div>
          <div className="metric-label">Blocked</div>
        </div>
        <div className="card metric-tile">
          <div className="metric-value">{counts.review ?? 0}</div>
          <div className="metric-label">Review</div>
        </div>
        <div className="card metric-tile">
          <div className="metric-value">{highRemaining}</div>
          <div className="metric-label">High priority remaining</div>
        </div>
      </div>

      <div className="card card-pad" style={{ marginBottom: 24 }}>
        <div className="progress-bar" style={{ marginBottom: 8 }}>
          <div className="progress-bar-fill" style={{ width: `${pct}%` }} />
        </div>
        <div className="page-subtitle">{pct}% complete</div>
      </div>

      <div className="dashboard-columns">
        <div className="card card-pad">
          <h2 className="section-title">Recent tickets</h2>
          <MiniList project={project} tickets={recent} emptyLabel="No tickets yet." />
        </div>
        <div className="card card-pad">
          <h2 className="section-title">Blocked</h2>
          <MiniList project={project} tickets={blocked} emptyLabel="Nothing blocked." />
        </div>
        <div className="card card-pad">
          <h2 className="section-title">Ready to work</h2>
          <MiniList project={project} tickets={ready} emptyLabel="Nothing ready." />
        </div>
        <div className="card card-pad">
          <h2 className="section-title">Major epics</h2>
          <MiniList project={project} tickets={epics} emptyLabel="No epics." />
        </div>
        {git?.is_repo && (
          <div className="card card-pad">
            <h2 className="section-title">Git</h2>
            <div className="field-row">
              <span className="field-label">Branch</span>
              <span>{git.branch}</span>
            </div>
            <div className="field-row">
              <span className="field-label">Working tree</span>
              <span>{git.clean ? 'clean' : `${git.changed_files} changed`}</span>
            </div>
            {(git.recent_commits ?? []).slice(0, 5).map((c) => (
              <div className="mini-ticket-row" key={c.hash}>
                <span className="mini-ticket-id">{c.short_hash}</span>
                <span className="mini-ticket-title">{c.subject}</span>
              </div>
            ))}
            {(git.recent_commits ?? []).length === 0 && (
              <div className="empty-note">No commits yet.</div>
            )}
          </div>
        )}
        <div className="card card-pad" style={{ gridColumn: '1 / -1' }}>
          <h2 className="section-title">Documentation</h2>
          {docs.length === 0 ? (
            <div className="empty-note">No Markdown documentation found in this project.</div>
          ) : (
            docs.map((d) => (
              <Link
                key={d}
                to={`/projects/${encodeURIComponent(project)}/docs/${d}`}
                className="mini-ticket-row"
              >
                <span className="mini-ticket-title">{d}</span>
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
