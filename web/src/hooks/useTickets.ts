import { useCallback, useEffect, useRef, useState } from 'react'
import { createTicket, listTickets, updateTicket, type CreateTicketInput } from '../api'
import type { Ticket, TicketStatus } from '../types'
import { usePolling } from './usePolling'

export function useTickets(project: string) {
  const [tickets, setTickets] = useState<Ticket[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(() => {
    setError(null)
    listTickets(project)
      .then(setTickets)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load tickets'))
  }, [project])

  useEffect(() => {
    setTickets(null)
    reload()
  }, [reload])

  // Live refresh so agents' changes show up without a reload. Skipped while
  // a local write is in flight, so a poll that raced it can't briefly undo
  // an optimistic update; identical results are dropped to avoid re-renders.
  const pendingWrites = useRef(0)
  const lastJSON = useRef('')
  usePolling(() => {
    if (pendingWrites.current > 0) return
    listTickets(project)
      .then((next) => {
        if (pendingWrites.current > 0) return
        const json = JSON.stringify(next)
        if (json === lastJSON.current) return
        lastJSON.current = json
        setTickets(next)
        setError(null)
      })
      .catch(() => {
        // Transient (e.g. the server restarting during a rebuild); the next
        // tick retries, and the initial load already surfaces hard errors.
      })
  })

  async function setStatus(id: string, status: TicketStatus) {
    const current = tickets?.find((t) => t.id === id)
    if (!current || current.status === status) return

    const prev = tickets
    // Move it to the end of the list, not just update in place — since
    // column order is just array order, this makes a drag-and-drop land
    // at the bottom of the destination column instead of wherever the
    // ticket's original position happens to fall.
    setTickets((cur) => {
      if (!cur) return cur
      const rest = cur.filter((t) => t.id !== id)
      return [...rest, { ...current, status }]
    })
    pendingWrites.current++
    try {
      await updateTicket(project, id, { status })
    } catch (err) {
      setTickets(prev)
      setError(err instanceof Error ? err.message : 'Failed to update ticket')
    } finally {
      pendingWrites.current--
    }
  }

  async function create(input: CreateTicketInput) {
    const created = await createTicket(project, input)
    setTickets((cur) => (cur ? [...cur, created] : [created]))
  }

  return { tickets, error, reload, setStatus, create }
}
