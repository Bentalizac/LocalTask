import { useLayoutEffect, useMemo, useRef, useState, type CSSProperties } from 'react'
import { flushSync } from 'react-dom'
import { Link, useParams } from 'react-router-dom'
import { useTickets } from '../hooks/useTickets'
import { useJiraBaseUrl } from '../hooks/useJiraBaseUrl'
import { AssigneeChip } from '../components/Badges'
import { JiraKeys } from '../components/JiraKeys'
import { layoutGraph, type Cluster, type Group, type PlacedNode } from '../graph/layout'
import type { Ticket, TicketStatus } from '../types'

const HIDE_DONE_KEY = 'localtask.graph.hideDone'
// A cluster's padding plus border, on each side. Keep in step with
// .graph-cluster in index.css.
const CLUSTER_INSET = 13

const GLYPH: Record<TicketStatus, string> = {
  backlog: '○',
  ready: '▸',
  in_progress: '●',
  blocked: '✗',
  review: '◐',
  done: '✓',
  cancelled: '–',
}

function readHideDone(): boolean {
  try {
    return localStorage.getItem(HIDE_DONE_KEY) === '1'
  } catch {
    return false
  }
}

function writeHideDone(on: boolean) {
  try {
    localStorage.setItem(HIDE_DONE_KEY, on ? '1' : '0')
  } catch {
    // Storage blocked (private window): the toggle still works for this visit.
  }
}

const statusStyle = (status: TicketStatus) => ({ '--status-color': `var(--status-${status})` }) as CSSProperties

export function GraphPage() {
  const { project = '' } = useParams<{ project: string }>()
  const { tickets, error } = useTickets(project)
  const jiraBaseUrl = useJiraBaseUrl(project)
  const [hideDone, setHideDone] = useState(readHideDone)
  const [width, setWidth] = useState(0)
  const [hot, setHot] = useState<string | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)

  // Re-run the layout whenever the graph's width changes, including the
  // first observation. flushSync renders the new layout before the browser
  // paints, so a resize never shows a stale frame.
  const hasTickets = tickets !== null
  useLayoutEffect(() => {
    const el = rootRef.current
    if (!el) return
    const observer = new ResizeObserver(([entry]) => {
      const next = Math.floor(entry.contentRect.width)
      flushSync(() => setWidth(next))
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [hasTickets])

  const clusters = useMemo(
    () => (tickets && width ? layoutGraph(tickets, width - 2 * CLUSTER_INSET, { hideDone }) : []),
    [tickets, width, hideDone],
  )

  // The hovered ticket's dependencies and dependents.
  const related = useMemo(() => {
    const set = new Set<string>()
    if (!hot || !tickets) return set
    for (const t of tickets) {
      if (t.id === hot) for (const dep of t.depends_on ?? []) set.add(dep)
      if (t.depends_on?.includes(hot)) set.add(t.id)
    }
    return set
  }, [hot, tickets])

  if (error)
    return (
      <div className="page">
        <div className="error-banner">{error}</div>
      </div>
    )
  if (!tickets)
    return (
      <div className="page">
        <div className="loading">Loading…</div>
      </div>
    )

  const live = tickets.filter((t) => t.status !== 'cancelled' && !t.archived_at)
  const done = live.filter((t) => t.status === 'done').length
  const byId = new Map(tickets.map((t) => [t.id, t]))
  const base = `/projects/${encodeURIComponent(project)}`

  return (
    <div className="page page-graph">
      <div className="page-header">
        <div>
          <h1>Graph</h1>
          <div className="page-subtitle">
            {done}/{live.length} done
          </div>
        </div>
        <label className="graph-toggle">
          <input
            type="checkbox"
            checked={hideDone}
            onChange={(e) => {
              setHideDone(e.target.checked)
              writeHideDone(e.target.checked)
            }}
          />
          Hide done
        </label>
      </div>

      <svg className="graph-defs" aria-hidden="true">
        <defs>
          <marker id="graph-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto">
            <path d="M0,0 L8,4 L0,8 z" className="graph-arrow" />
          </marker>
          <marker id="graph-arrow-hot" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto">
            <path d="M0,0 L8,4 L0,8 z" className="graph-arrow hot" />
          </marker>
        </defs>
      </svg>

      <div className="graph" ref={rootRef}>
        {live.length === 0 && <div className="empty-state">No tickets yet.</div>}
        {clusters.map((cluster) => (
          <ClusterView
            key={cluster.id || '(unparented)'}
            cluster={cluster}
            base={base}
            jiraBaseUrl={jiraBaseUrl}
            byId={byId}
            hot={hot}
            related={related}
            onHover={setHot}
          />
        ))}
      </div>
    </div>
  )
}

interface ViewProps {
  base: string
  jiraBaseUrl?: string
  byId: Map<string, Ticket>
  hot: string | null
  related: Set<string>
  onHover: (id: string | null) => void
}

const ticketPath = (base: string, id: string) => `${base}/tickets/${encodeURIComponent(id)}`

function ClusterView({ cluster, ...view }: ViewProps & { cluster: Cluster }) {
  const { base, jiraBaseUrl } = view
  const title = cluster.feature ? (
    <Link to={ticketPath(base, cluster.feature.id)}>{cluster.title}</Link>
  ) : (
    cluster.title
  )

  if (cluster.folded)
    return (
      <div className="graph-fold graph-fold-cluster">
        <span className="graph-fold-title">✓ {title}</span>
        <span className="graph-count">
          {cluster.done}/{cluster.total}
        </span>
        <JiraKeys keys={cluster.feature?.jira} baseUrl={jiraBaseUrl} />
      </div>
    )

  const pct = cluster.total ? (100 * cluster.done) / cluster.total : 0
  return (
    <section className="graph-cluster">
      <header className="graph-cluster-header">
        <div className="graph-cluster-title-row">
          <h2 className="graph-cluster-title">{title}</h2>
          <span className="graph-count">
            {cluster.done}/{cluster.total}
          </span>
        </div>
        {(cluster.feature?.jira?.length || cluster.after.length > 0) && (
          <div className="graph-cluster-meta">
            <JiraKeys keys={cluster.feature?.jira} baseUrl={jiraBaseUrl} />
            {cluster.after.length > 0 && <span className="graph-after">after {cluster.after.join(', ')}</span>}
          </div>
        )}
        <div className="graph-bar">
          <div className="graph-bar-fill" style={{ width: `${pct}%` }} />
        </div>
      </header>
      {cluster.groups.map((group) => (
        <GroupView key={group.section?.id ?? '(none)'} group={group} {...view} />
      ))}
    </section>
  )
}

function GroupView({ group, ...view }: ViewProps & { group: Group }) {
  const { section } = group
  if (group.folded && section)
    return (
      <div className="graph-fold graph-fold-section">
        <span className="graph-fold-title">✓ {section.title}</span>
        <span className="graph-count">
          {group.done}/{group.total}
        </span>
      </div>
    )

  return (
    <div className="graph-group">
      {section && (
        <div className="graph-section-header">
          <h3>{section.title}</h3>
          <span className="graph-count">
            {group.done}/{group.total}
          </span>
        </div>
      )}
      {group.nodes.length > 0 && (
        <div className="graph-canvas" style={{ height: group.height }}>
          <svg className="graph-edges" width="100%" height={group.height} aria-hidden="true">
            {group.edges.map((edge) => {
              const isHot = view.hot === edge.from || view.hot === edge.to
              return (
                <path
                  key={`${edge.from}>${edge.to}`}
                  d={edge.path}
                  className={`graph-edge${edge.met ? ' met' : ''}${isHot ? ' hot' : ''}`}
                  markerEnd={`url(#${isHot ? 'graph-arrow-hot' : 'graph-arrow'})`}
                />
              )
            })}
          </svg>
          {group.nodes.map((node) => (
            <NodeView key={node.ticket.id} node={node} {...view} />
          ))}
        </div>
      )}
    </div>
  )
}

function tooltipOf(ticket: Ticket, byId: Map<string, Ticket>): string {
  const deps = (ticket.depends_on ?? []).map((dep) => {
    const status = byId.get(dep)?.status
    return `  ${status === 'done' ? '✓' : '○'} ${dep} ${status ? status.replace('_', ' ') : '(missing)'}`
  })
  return [
    `${ticket.id} · ${ticket.title}`,
    `${ticket.status.replace('_', ' ')}${ticket.assignee ? ` · ${ticket.assignee}` : ''}`,
    ...(deps.length ? ['depends on:', ...deps] : []),
  ].join('\n')
}

function NodeView({ node, base, jiraBaseUrl, byId, hot, related, onHover }: ViewProps & { node: PlacedNode }) {
  const { ticket } = node
  const state = hot === ticket.id ? ' is-hot' : related.has(ticket.id) ? ' is-related' : hot ? ' is-dim' : ''
  const style = {
    ...statusStyle(ticket.status),
    left: node.x,
    top: node.y,
    width: node.w,
    height: node.h,
  }
  return (
    <div
      className={`graph-node status-${ticket.status}${node.decision ? ' decision' : ''}${state}`}
      style={style}
      title={tooltipOf(ticket, byId)}
      onMouseEnter={() => onHover(ticket.id)}
      onMouseLeave={() => onHover(null)}
    >
      <div className="graph-node-top">
        <span className="graph-glyph">{GLYPH[ticket.status]}</span>
        <span className="graph-node-id">{ticket.id}</span>
        {node.decision && <span className="graph-node-kind">decision</span>}
        {ticket.assignee && <AssigneeChip name={ticket.assignee} />}
      </div>
      <Link className="graph-node-title" to={ticketPath(base, ticket.id)}>
        {ticket.title}
      </Link>
      {(node.chips.length > 0 || (ticket.jira?.length ?? 0) > 0) && (
        <div className="graph-node-meta">
          <JiraKeys keys={ticket.jira} baseUrl={jiraBaseUrl} />
          {node.chips.length > 0 && (
            <span className="graph-needs">
              needs
              {node.chips.map((chip) => (
                <span
                  key={chip.id}
                  className={`graph-chip ${chip.met ? 'met' : 'unmet'}`}
                  title={`${chip.id}: ${chip.status.replace('_', ' ')}`}
                >
                  {chip.met ? '✓' : '○'} {chip.id}
                </span>
              ))}
            </span>
          )}
        </div>
      )}
    </div>
  )
}
