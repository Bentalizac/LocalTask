import { useState } from 'react'
import { Link } from 'react-router-dom'
import { PriorityBadge, TypeBadge } from './Badges'
import type { Ticket, TicketStatus } from '../types'
import { BOARD_COLUMNS } from '../types'

interface Props {
  project: string
  ticket: Ticket
  onStatusChange: (id: string, status: TicketStatus) => void
}

export function TicketCard({ project, ticket, onStatusChange }: Props) {
  const [isDragging, setIsDragging] = useState(false)
  const criteria = ticket.acceptance_criteria ?? []
  const complete = criteria.filter((c) => c.complete).length

  return (
    <div
      className={`card ticket-card${isDragging ? ' dragging' : ''}`}
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData('text/plain', ticket.id)
        e.dataTransfer.effectAllowed = 'move'
        setIsDragging(true)
      }}
      onDragEnd={() => setIsDragging(false)}
    >
      <div className="ticket-card-top">
        <span>{ticket.id}</span>
      </div>
      <Link
        draggable={false}
        to={`/projects/${encodeURIComponent(project)}/tickets/${encodeURIComponent(ticket.id)}`}
      >
        <div className="ticket-card-title">{ticket.title}</div>
      </Link>
      <div className="ticket-card-badges">
        <PriorityBadge priority={ticket.priority} />
        <TypeBadge type={ticket.type} />
        {(ticket.tags ?? []).map((tag) => (
          <span className="tag" key={tag}>
            {tag}
          </span>
        ))}
      </div>
      <div className="ticket-card-footer">
        {criteria.length > 0 ? (
          <span className="ac-progress">
            AC: {complete}/{criteria.length}
          </span>
        ) : (
          <span />
        )}
        <select
          className="status-select"
          value={ticket.status}
          onChange={(e) => onStatusChange(ticket.id, e.target.value as TicketStatus)}
          onClick={(e) => e.stopPropagation()}
        >
          {BOARD_COLUMNS.concat(
            ticket.status === 'cancelled' ? ['cancelled'] : [],
          ).map((s) => (
            <option key={s} value={s}>
              {s.replace('_', ' ')}
            </option>
          ))}
        </select>
      </div>
    </div>
  )
}
