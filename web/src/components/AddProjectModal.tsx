import { useEffect, useState, type FormEvent } from 'react'
import { addProject, browseDirectory } from '../api'
import type { BrowseListing } from '../types'

interface Props {
  onClose: () => void
  onAdded: (projectName: string) => void
}

export function AddProjectModal({ onClose, onAdded }: Props) {
  const [listing, setListing] = useState<BrowseListing | null>(null)
  const [pathInput, setPathInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [naming, setNaming] = useState(false)
  const [name, setName] = useState('')
  const [submitting, setSubmitting] = useState(false)

  function load(path?: string) {
    setLoading(true)
    setError(null)
    browseDirectory(path)
      .then((l) => {
        setListing(l)
        setPathInput(l.path)
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to browse'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function goTo(path: string) {
    setNaming(false)
    load(path)
  }

  function handlePathSubmit(e: FormEvent) {
    e.preventDefault()
    goTo(pathInput.trim())
  }

  function startNaming() {
    if (!listing) return
    const base = listing.path.split('/').filter(Boolean).pop() ?? listing.path
    setName(base)
    setNaming(true)
  }

  async function handleAdd(e: FormEvent) {
    e.preventDefault()
    if (!listing || !name.trim()) return
    setSubmitting(true)
    setError(null)
    try {
      const project = await addProject(name.trim(), listing.path)
      onAdded(project.name)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add project')
      setSubmitting(false)
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <h2>Add project</h2>

        {!naming && (
          <>
            <form className="browse-path-form" onSubmit={handlePathSubmit}>
              <input
                value={pathInput}
                onChange={(e) => setPathInput(e.target.value)}
                placeholder="/path/to/project"
                autoFocus
              />
              <button type="submit" className="btn btn-sm">
                Go
              </button>
            </form>

            {error && <div className="error-banner">{error}</div>}
            {loading && <div className="loading">Loading…</div>}

            {listing && !loading && (
              <>
                <div className="browse-list">
                  {listing.parent && (
                    <button className="browse-row" onClick={() => goTo(listing.parent!)}>
                      <span className="browse-row-name">.. (up)</span>
                    </button>
                  )}
                  {listing.entries.length === 0 && (
                    <div className="empty-note">No subfolders here.</div>
                  )}
                  {listing.entries.map((entry) => (
                    <button className="browse-row" key={entry.path} onClick={() => goTo(entry.path)}>
                      <span className="browse-row-name">📁 {entry.name}</span>
                      {entry.has_taskboard && <span className="tag">taskboard</span>}
                    </button>
                  ))}
                </div>

                <div className="browse-current">
                  <div>
                    <div className="field-label">Current folder</div>
                    <div className="browse-current-path">{listing.path}</div>
                  </div>
                  <button className="btn btn-primary btn-sm" onClick={startNaming}>
                    Use this folder
                  </button>
                </div>
              </>
            )}

            <div className="form-actions">
              <button type="button" className="btn" onClick={onClose}>
                Cancel
              </button>
            </div>
          </>
        )}

        {naming && listing && (
          <form onSubmit={handleAdd}>
            <div className="form-row">
              <label htmlFor="project-name">Project name</label>
              <input
                id="project-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
              />
            </div>
            <div className="page-subtitle" style={{ marginBottom: 12 }}>
              {listing.path}
              {listing.has_taskboard
                ? ' — already has a taskboard.json, will be added as-is.'
                : ' — no taskboard.json yet, an empty one will be created.'}
            </div>
            {error && <div className="error-banner">{error}</div>}
            <div className="form-actions">
              <button type="button" className="btn" onClick={() => setNaming(false)} disabled={submitting}>
                Back
              </button>
              <button type="submit" className="btn btn-primary" disabled={submitting || !name.trim()}>
                {submitting ? 'Adding…' : 'Add project'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
