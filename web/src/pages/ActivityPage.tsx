import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTickets } from '../hooks/useTickets'
import { AssigneeChip, StatusBadge } from '../components/Badges'
import { isStale, relativeTime } from '../time'
import type { ActivityEntry, Ticket } from '../types'

const FEED_LIMIT = 80

// useNow re-renders every 30s so relative timestamps keep moving even when
// polling finds nothing new.
function useNow() {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 30_000)
    return () => window.clearInterval(id)
  }, [])
  return now
}

function ticketPath(project: string, id: string) {
  return `/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(id)}`
}

// ActivityPage is a single-column "what are the agents doing" view, laid
// out to stay readable in a narrow side pane: who holds what right now,
// what's ready to be picked up, and a live feed of every ticket's activity.
export function ActivityPage() {
  const { project = '' } = useParams<{ project: string }>()
  const { tickets, error } = useTickets(project)
  const now = useNow()

  if (error) return <div className="page"><div className="error-banner">{error}</div></div>
  if (!tickets) return <div className="page"><div className="loading">Loading…</div></div>

  const active = tickets.filter((t) => t.status === 'in_progress' || t.status === 'review')
  const byAgent = new Map<string, Ticket[]>()
  const unclaimed: Ticket[] = []
  for (const t of active) {
    if (!t.assignee) {
      unclaimed.push(t)
      continue
    }
    byAgent.set(t.assignee, [...(byAgent.get(t.assignee) ?? []), t])
  }
  const agents = [...byAgent.entries()].sort(([a], [b]) => a.localeCompare(b))

  const doneIds = new Set(tickets.filter((t) => t.status === 'done' || t.status === 'cancelled').map((t) => t.id))
  const ready = tickets.filter(
    (t) => t.status === 'ready' && !t.assignee && (t.depends_on ?? []).every((d) => doneIds.has(d)),
  )

  // Timestamps are second-resolution, so break ties by position in the
  // ticket's log (which is append-only) to keep same-second entries in order.
  const feed: { entry: ActivityEntry; ticket: Ticket; seq: number }[] = []
  for (const t of tickets) (t.activity ?? []).forEach((entry, seq) => feed.push({ entry, ticket: t, seq }))
  feed.sort((a, b) => b.entry.at.localeCompare(a.entry.at) || b.seq - a.seq)

  return (
    <div className="page page-activity">
      <div className="page-header">
        <h1>Activity</h1>
        <span className="live-indicator" title="Refreshes every few seconds">
          <span className="live-dot" /> Live
        </span>
      </div>

      <section className="activity-section">
        <h2 className="section-title">Working now</h2>
        {agents.length === 0 && unclaimed.length === 0 && (
          <div className="empty-note">Nothing in progress.</div>
        )}
        {agents.map(([agent, list]) => (
          <div className="agent-group" key={agent}>
            <AssigneeChip name={agent} />
            {list.map((t) => (
              <ActiveTicketRow key={t.id} project={project} ticket={t} now={now} />
            ))}
          </div>
        ))}
        {unclaimed.length > 0 && (
          <div className="agent-group">
            <span className="empty-note">Unassigned</span>
            {unclaimed.map((t) => (
              <ActiveTicketRow key={t.id} project={project} ticket={t} now={now} />
            ))}
          </div>
        )}
      </section>

      <section className="activity-section">
        <h2 className="section-title">Ready to pick up ({ready.length})</h2>
        {ready.length === 0 ? (
          <div className="empty-note">No unblocked, unassigned ready tickets.</div>
        ) : (
          ready.slice(0, 8).map((t) => (
            <Link key={t.id} to={ticketPath(project, t.id)} className="mini-ticket-row">
              <span className="mini-ticket-id">{t.id}</span>
              <span className="mini-ticket-title">{t.title}</span>
            </Link>
          ))
        )}
      </section>

      <section className="activity-section">
        <h2 className="section-title">Recent activity</h2>
        {feed.length === 0 ? (
          <div className="empty-note">
            No activity recorded yet. Agents' claims, status changes and notes will appear here.
          </div>
        ) : (
          <ol className="activity-feed">
            {feed.slice(0, FEED_LIMIT).map(({ entry, ticket }, i) => (
              <li key={`${ticket.id}-${entry.at}-${i}`} className={`activity-item kind-${entry.kind}`}>
                <div className="activity-item-head">
                  {entry.author ? <AssigneeChip name={entry.author} /> : <span className="empty-note">unknown</span>}
                  <Link to={ticketPath(project, ticket.id)} className="activity-ticket" title={ticket.title}>
                    {ticket.id}
                  </Link>
                  <span className="activity-time" title={entry.at}>
                    {relativeTime(entry.at, now)}
                  </span>
                </div>
                <div className="activity-text">{entry.text}</div>
              </li>
            ))}
          </ol>
        )}
      </section>
    </div>
  )
}

function ActiveTicketRow({ project, ticket, now }: { project: string; ticket: Ticket; now: number }) {
  const stale = ticket.status === 'in_progress' && isStale(ticket.updated_at, now)
  const lastNote = [...(ticket.activity ?? [])].reverse().find((e) => e.kind === 'note')
  return (
    <Link to={ticketPath(project, ticket.id)} className="active-ticket">
      <div className="active-ticket-head">
        <span className="mini-ticket-id">{ticket.id}</span>
        <StatusBadge status={ticket.status} />
        <span className={`activity-time${stale ? ' stale' : ''}`} title={stale ? 'No updates for 30+ minutes' : ticket.updated_at}>
          {stale && '⚠ '}
          {relativeTime(ticket.updated_at, now)}
        </span>
      </div>
      <div className="active-ticket-title">{ticket.title}</div>
      {lastNote && <div className="active-ticket-note">{lastNote.text}</div>}
    </Link>
  )
}
