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
