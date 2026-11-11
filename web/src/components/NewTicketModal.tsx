import { useState, type FormEvent } from 'react'
import { TICKET_PRIORITIES, TICKET_STATUSES, TICKET_TYPES } from '../types'
import type { CreateTicketInput } from '../api'
import { parseJiraKeys } from '../jira'

interface Props {
  onClose: () => void
  onCreate: (input: CreateTicketInput) => Promise<void>
}

export function NewTicketModal({ onClose, onCreate }: Props) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [type, setType] = useState('task')
  const [status, setStatus] = useState('backlog')
  const [priority, setPriority] = useState('medium')
  const [tags, setTags] = useState('')
  const [jira, setJira] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!title.trim()) {
      setError('Title is required')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await onCreate({
        title: title.trim(),
        description: description.trim() || undefined,
        type: type as CreateTicketInput['type'],
        status: status as CreateTicketInput['status'],
        priority: priority as CreateTicketInput['priority'],
        tags: tags
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
        jira: parseJiraKeys(jira),
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create ticket')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>New ticket</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-row">
            <label htmlFor="title">Title</label>
            <input id="title" value={title} onChange={(e) => setTitle(e.target.value)} autoFocus />
          </div>
          <div className="form-row">
            <label htmlFor="description">Description</label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="form-row">
            <label htmlFor="type">Type</label>
            <select id="type" value={type} onChange={(e) => setType(e.target.value)}>
              {TICKET_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </div>
          <div className="form-row">
            <label htmlFor="status">Status</label>
            <select id="status" value={status} onChange={(e) => setStatus(e.target.value)}>
              {TICKET_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s.replace('_', ' ')}
                </option>
              ))}
            </select>
          </div>
          <div className="form-row">
            <label htmlFor="priority">Priority</label>
            <select id="priority" value={priority} onChange={(e) => setPriority(e.target.value)}>
              {TICKET_PRIORITIES.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <div className="form-row">
            <label htmlFor="tags">Tags (comma separated)</label>
            <input id="tags" value={tags} onChange={(e) => setTags(e.target.value)} />
          </div>
          <div className="form-row">
            <label htmlFor="jira">Jira issues</label>
            <input id="jira" value={jira} onChange={(e) => setJira(e.target.value)} placeholder="PROJ-123, PROJ-150" />
          </div>
          {error && <div className="error-banner">{error}</div>}
          <div className="form-actions">
            <button type="button" className="btn" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={submitting}>
              {submitting ? 'Creating…' : 'Create ticket'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
