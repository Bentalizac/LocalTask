import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { marked } from 'marked'
import { getDoc } from '../api'

export function DocumentPage() {
  const { project = '', '*': path = '' } = useParams<{ project: string; '*': string }>()
  const [html, setHtml] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setHtml(null)
    setError(null)
    getDoc(project, path)
      .then((doc) => setHtml(marked.parse(doc.content, { async: false })))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load document'))
  }, [project, path])

  return (
    <div className="page">
      <div className="page-header">
        <h1>{path}</h1>
      </div>
      {error && <div className="error-banner">{error}</div>}
      {!html && !error && <div className="loading">Loading…</div>}
      {html && (
        <div className="card markdown-body" dangerouslySetInnerHTML={{ __html: html }} />
      )}
    </div>
  )
}
