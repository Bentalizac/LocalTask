import { useState, type CSSProperties } from 'react'
import { useParams } from 'react-router-dom'
import { useTickets } from '../hooks/useTickets'
import { TicketCard } from '../components/TicketCard'
import { NewTicketModal } from '../components/NewTicketModal'
import { BOARD_COLUMNS } from '../types'

export function BoardPage() {
  const { project = '' } = useParams<{ project: string }>()
  const { tickets, error, setStatus, create } = useTickets(project)
  const [showNew, setShowNew] = useState(false)

  if (error) return <div className="page"><div className="error-banner">{error}</div></div>
  if (!tickets) return <div className="page"><div className="loading">Loading…</div></div>

  return (
    <div className="page">
      <div className="page-header">
        <h1>Board</h1>
        <button className="btn btn-primary" onClick={() => setShowNew(true)}>
          + New ticket
        </button>
      </div>

      <div className="board">
        {BOARD_COLUMNS.map((status) => {
          const columnTickets = tickets.filter((t) => t.status === status)
          const style = { '--status-color': `var(--status-${status})` } as CSSProperties
          return (
            <div className="board-column" key={status}>
              <div className="board-column-header" style={style}>
                <span className="board-column-title">{status.replace('_', ' ')}</span>
                <span className="board-column-count">{columnTickets.length}</span>
              </div>
              <div className="board-column-cards">
                {columnTickets.map((t) => (
                  <TicketCard key={t.id} project={project} ticket={t} onStatusChange={setStatus} />
                ))}
              </div>
            </div>
          )
        })}
      </div>

      {showNew && (
        <NewTicketModal onClose={() => setShowNew(false)} onCreate={create} />
      )}
    </div>
  )
}
