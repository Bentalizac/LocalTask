import { useCallback, useEffect, useState } from 'react'
import { createTicket, listTickets, updateTicket, type CreateTicketInput } from '../api'
import type { Ticket, TicketStatus } from '../types'

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

  async function setStatus(id: string, status: TicketStatus) {
    const prev = tickets
    setTickets((cur) => cur?.map((t) => (t.id === id ? { ...t, status } : t)) ?? cur)
    try {
      await updateTicket(project, id, { status })
    } catch (err) {
      setTickets(prev)
      setError(err instanceof Error ? err.message : 'Failed to update ticket')
    }
  }

  async function create(input: CreateTicketInput) {
    const created = await createTicket(project, input)
    setTickets((cur) => (cur ? [...cur, created] : [created]))
  }

  return { tickets, error, reload, setStatus, create }
}
