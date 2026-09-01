import { useState } from 'react'
import { NavLink, Outlet, useNavigate, useParams } from 'react-router-dom'
import { createTicket } from '../api'
import { NewTicketModal } from './NewTicketModal'

export function Layout() {
  const { project } = useParams<{ project: string }>()
  const base = `/projects/${encodeURIComponent(project ?? '')}`
  const [showNew, setShowNew] = useState(false)
  const navigate = useNavigate()

  return (
    <div className="app-shell">
      <header className="topbar">
        <NavLink to="/" className="topbar-brand" end>
          LocalTask
        </NavLink>
        <span className="topbar-project">{project}</span>
        <nav className="topbar-nav">
          <NavLink to={base} end>
            Dashboard
          </NavLink>
          <NavLink to={`${base}/board`}>Board</NavLink>
          <NavLink to={`${base}/list`}>List</NavLink>
        </nav>
        <button className="btn btn-primary btn-sm" onClick={() => setShowNew(true)}>
          + New ticket
        </button>
      </header>
      <Outlet />

      {showNew && project && (
        <NewTicketModal
          onClose={() => setShowNew(false)}
          onCreate={async (input) => {
            const created = await createTicket(project, input)
            navigate(`${base}/tickets/${encodeURIComponent(created.id)}`)
          }}
        />
      )}
    </div>
  )
}
