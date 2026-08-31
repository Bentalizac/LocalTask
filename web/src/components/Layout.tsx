import { NavLink, Outlet, useParams } from 'react-router-dom'

export function Layout() {
  const { project } = useParams<{ project: string }>()
  const base = `/projects/${encodeURIComponent(project ?? '')}`

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
      </header>
      <Outlet />
    </div>
  )
}
