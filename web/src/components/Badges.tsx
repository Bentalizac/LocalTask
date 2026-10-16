import type { CSSProperties } from 'react'
import type { TicketPriority, TicketStatus, TicketType } from '../types'

export function StatusBadge({ status }: { status: TicketStatus }) {
  const style = { '--status-color': `var(--status-${status})` } as CSSProperties
  return (
    <span className="badge badge-status" style={style}>
      {status.replace('_', ' ')}
    </span>
  )
}

export function PriorityBadge({ priority }: { priority: TicketPriority }) {
  const style = { '--priority-color': `var(--priority-${priority})` } as CSSProperties
  return (
    <span className="badge badge-priority" style={style}>
      {priority}
    </span>
  )
}

export function TypeBadge({ type }: { type: TicketType }) {
  return <span className="badge badge-type">{type}</span>
}

// agentHue gives each agent name a stable color so the same agent is
// recognizable across cards and the activity feed.
// FNV-1a, so names differing only in a trailing digit ("worker-1",
// "worker-2") still land far apart on the color wheel.
function agentHue(name: string): number {
  let h = 0x811c9dc5
  for (const ch of name) h = Math.imul(h ^ ch.charCodeAt(0), 16777619)
  return (h >>> 0) % 360
}

export function AssigneeChip({ name }: { name: string }) {
  const style = { '--agent-hue': agentHue(name) } as CSSProperties
  return (
    <span className="assignee-chip" style={style} title={`Assigned to ${name}`}>
      <span className="assignee-dot" />
      {name}
    </span>
  )
}
