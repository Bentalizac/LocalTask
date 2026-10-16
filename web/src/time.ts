// relativeTime renders an ISO timestamp as a compact "3m ago" string.
export function relativeTime(iso: string | undefined, now = Date.now()): string {
  if (!iso) return ''
  const secs = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000))
  if (secs < 45) return 'just now'
  const mins = Math.round(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}

// isStale flags in-progress work nobody has touched for a while — usually
// an agent that crashed or was stopped without releasing its ticket.
export const STALE_AFTER_MS = 30 * 60 * 1000

export function isStale(iso: string | undefined, now = Date.now()): boolean {
  return !!iso && now - new Date(iso).getTime() > STALE_AFTER_MS
}
