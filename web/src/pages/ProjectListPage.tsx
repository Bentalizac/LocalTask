import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { listProjects } from '../api'
import { AddProjectModal } from '../components/AddProjectModal'
import type { ProjectSummary } from '../types'

export function ProjectListPage() {
  const [projects, setProjects] = useState<ProjectSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const navigate = useNavigate()

  function reload() {
    listProjects()
      .then(setProjects)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load projects'))
  }

  useEffect(reload, [])

  function handleAdded(projectName: string) {
    setShowAdd(false)
    navigate(`/projects/${encodeURIComponent(projectName)}`)
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <span className="topbar-brand">LocalTask</span>
      </header>
      <div className="page">
        <div className="page-header">
          <h1>Projects</h1>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>
            + Add Project
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        {!projects && !error && <div className="loading">Loading…</div>}
        {projects && projects.length === 0 && (
          <div className="empty-note">
            No projects configured yet. Click "+ Add Project" to browse to one.
          </div>
        )}
        <div className="project-list">
          {projects?.map((p) => (
            <Link
              key={p.name}
              to={`/projects/${encodeURIComponent(p.name)}`}
              className="card project-list-item"
            >
              <div className="project-list-item-name">{p.name}</div>
              {p.description && <div className="project-list-item-desc">{p.description}</div>}
              <div className="project-list-item-desc">{p.metrics.total} tickets</div>
            </Link>
          ))}
        </div>
      </div>
      {showAdd && <AddProjectModal onClose={() => setShowAdd(false)} onAdded={handleAdded} />}
    </div>
  )
}
