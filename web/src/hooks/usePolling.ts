import { useEffect, useRef } from 'react'

// How often live views re-fetch. Agents write through MCP in other
// processes, so polling is how the browser finds out about their changes.
export const POLL_INTERVAL_MS = 3000

// usePolling calls fn every POLL_INTERVAL_MS while the tab is visible, and
// immediately when it becomes visible again. fn may change between renders;
// the latest one is always used.
export function usePolling(fn: () => void, enabled = true) {
  const fnRef = useRef(fn)
  useEffect(() => {
    fnRef.current = fn
  })

  useEffect(() => {
    if (!enabled) return
    const tick = () => {
      if (!document.hidden) fnRef.current()
    }
    const id = window.setInterval(tick, POLL_INTERVAL_MS)
    document.addEventListener('visibilitychange', tick)
    return () => {
      window.clearInterval(id)
      document.removeEventListener('visibilitychange', tick)
    }
  }, [enabled])
}
