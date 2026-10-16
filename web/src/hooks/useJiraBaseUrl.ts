import { useEffect, useState } from 'react'
import { getProject } from '../api'

// Returns the Jira site a project's Jira keys link to, or undefined when
// none is configured (keys then render unlinked). The fetched value is kept
// with the project it belongs to, so switching projects never shows the
// previous project's site.
export function useJiraBaseUrl(project: string): string | undefined {
  const [fetched, setFetched] = useState<{ project: string; url?: string } | null>(null)

  useEffect(() => {
    let current = true
    getProject(project)
      .then((p) => current && setFetched({ project, url: p.jira_base_url }))
      .catch(() => current && setFetched({ project }))
    return () => {
      current = false
    }
  }, [project])

  return fetched?.project === project ? fetched.url : undefined
}
