// Pure layout for the Graph page: groups a project's tickets into feature
// clusters and sections, then places each section's tickets in layers by
// their dependencies on each other. Ported from the roadmap mod's flowchart
// (featureOrder, depths, groupsOf, layout). No React here.

import type { Ticket } from '../types'

export const HGAP = 12
export const VGAP = 28
const MIN_NODE_W = 170
const MAX_NODE_W = 280
const MAX_PER_ROW = 4
// Node heights in CSS pixels. Keep in step with .graph-node in index.css.
const NODE_BASE_H = 74
const META_LINE_H = 22
// Average glyph advance of the 11px meta-line text, for wrapping estimates.
const CHAR_PX = 6.6
const NEEDS_LABEL_PX = 40
const ITEM_GAP_PX = 6

export interface Chip {
  id: string
  status: Ticket['status']
  met: boolean
}

export interface PlacedNode {
  ticket: Ticket
  x: number
  y: number
  w: number
  h: number
  decision: boolean
  // Dependencies drawn as chips: those outside this node's group, or hidden.
  chips: Chip[]
}

export interface Edge {
  from: string
  to: string
  met: boolean
  path: string
}

export interface Section {
  id: string
  title: string
}

export interface Group {
  section?: Section
  done: number
  total: number
  folded: boolean
  nodes: PlacedNode[]
  edges: Edge[]
  height: number
}

export interface Cluster {
  // The feature ticket's id, or '' for the "Unparented" cluster.
  id: string
  feature?: Ticket
  title: string
  done: number
  total: number
  // Titles of the features this one's tickets depend on.
  after: string[]
  folded: boolean
  groups: Group[]
}

export interface GraphOptions {
  hideDone?: boolean
}

const SECTION_PREFIX = 'section:'

// sectionOf reads a ticket's section from its first `section:<id>` tag.
// The title is the id with dashes as spaces, first letter capitalized.
export function sectionOf(ticket: Ticket): Section | undefined {
  const tag = (ticket.tags ?? []).find((t) => t.startsWith(SECTION_PREFIX))
  const id = tag?.slice(SECTION_PREFIX.length)
  if (!id) return undefined
  const words = id.replace(/-/g, ' ')
  return { id, title: words.charAt(0).toUpperCase() + words.slice(1) }
}

export function isDecision(ticket: Ticket): boolean {
  return (ticket.tags ?? []).includes('decision')
}

const isDone = (t: Ticket) => t.status === 'done'

// nodesPerRow picks how many nodes fit side by side in a group `width` wide.
export function nodesPerRow(width: number): { perRow: number; nodeW: number } {
  const perRow = Math.max(1, Math.min(MAX_PER_ROW, Math.floor((width + HGAP) / (MIN_NODE_W + HGAP))))
  const nodeW = Math.min(MAX_NODE_W, (width - (perRow - 1) * HGAP) / perRow)
  return { perRow, nodeW: Math.max(0, nodeW) }
}

// metaLines estimates how many lines a node's Jira keys and "needs" chips
// wrap to at width `w`, so the node is tall enough that nothing is clipped.
// The "needs" label and its chips wrap as one unit.
export function metaLines(ticket: Ticket, chips: Chip[], w: number): number {
  const room = w - 24
  const keys = (ticket.jira ?? []).map((k) => k.length * CHAR_PX + 16)
  const needs = chips.length
    ? NEEDS_LABEL_PX + chips.reduce((px, c) => px + (c.id.length + 2) * CHAR_PX + ITEM_GAP_PX, 0)
    : 0
  if (!keys.length && !needs) return 0
  let lines = 1
  let used = 0
  for (const px of keys) {
    if (used > 0 && used + px > room) {
      lines++
      used = 0
    }
    used += px + ITEM_GAP_PX
  }
  if (needs) {
    if (used > 0 && used + needs > room) lines++
    lines += Math.ceil(needs / room) - 1
  }
  return lines
}

// depths gives each ticket the length of its longest dependency path within
// `tickets`. A cycle counts as a root.
export function depths(tickets: Ticket[]): Map<string, number> {
  const byId = new Map(tickets.map((t) => [t.id, t]))
  const memo = new Map<string, number>()
  const visiting = new Set<string>()
  const depth = (id: string): number => {
    const known = memo.get(id)
    if (known !== undefined) return known
    if (visiting.has(id)) return 0
    visiting.add(id)
    const local = (byId.get(id)?.depends_on ?? []).filter((dep) => byId.has(dep))
    const d = local.length ? 1 + Math.max(...local.map(depth)) : 0
    visiting.delete(id)
    memo.set(id, d)
    return d
  }
  for (const t of tickets) depth(t.id)
  return memo
}

// featureOrder sorts feature ids so one whose tickets depend on another's
// comes after it. Ties and cycles keep the given (file) order.
export function featureOrder(featureIds: string[], featureOf: Map<string, string>, tickets: Ticket[]): string[] {
  const needs = new Map(featureIds.map((id) => [id, new Set<string>()]))
  for (const t of tickets) {
    const to = featureOf.get(t.id)
    if (to === undefined) continue
    for (const dep of t.depends_on ?? []) {
      const from = featureOf.get(dep)
      if (from !== undefined && from !== to) needs.get(to)?.add(from)
    }
  }
  const out: string[] = []
  const seen = new Set<string>()
  const visit = (id: string, path: Set<string>) => {
    if (seen.has(id) || path.has(id)) return
    path.add(id)
    for (const dep of needs.get(id) ?? []) visit(dep, path)
    path.delete(id)
    seen.add(id)
    out.push(id)
  }
  for (const id of featureIds) visit(id, new Set())
  return out
}

// groupsOf splits a cluster's tickets by section. Unsectioned tickets lead;
// sections follow in the order their first ticket appears.
export function groupsOf(tickets: Ticket[]): { section?: Section; tickets: Ticket[] }[] {
  const loose = tickets.filter((t) => !sectionOf(t))
  const groups: { section?: Section; tickets: Ticket[] }[] = loose.length ? [{ tickets: loose }] : []
  const index = new Map<string, number>()
  for (const t of tickets) {
    const section = sectionOf(t)
    if (!section) continue
    const at = index.get(section.id)
    if (at === undefined) {
      index.set(section.id, groups.length)
      groups.push({ section, tickets: [t] })
    } else {
      groups[at].tickets.push(t)
    }
  }
  return groups
}

// placeGroup lays one group's shown tickets out in layers by their
// dependencies on each other, wrapping each layer into rows.
function placeGroup(shown: Ticket[], width: number, byId: Map<string, Ticket>) {
  const { perRow, nodeW } = nodesPerRow(width)
  const local = new Set(shown.map((t) => t.id))
  const depth = depths(shown)
  const layers: Ticket[][] = []
  for (const t of shown) (layers[depth.get(t.id) ?? 0] ??= []).push(t)

  const nodes: PlacedNode[] = []
  let y = 0
  for (const layer of layers.filter(Boolean)) {
    for (let i = 0; i < layer.length; i += perRow) {
      const row = layer.slice(i, i + perRow).map((ticket) => {
        const chips = (ticket.depends_on ?? [])
          .filter((dep) => byId.has(dep) && !local.has(dep))
          .map((dep) => {
            const status = byId.get(dep)!.status
            return { id: dep, status, met: status === 'done' }
          })
        const h = NODE_BASE_H + metaLines(ticket, chips, nodeW) * META_LINE_H
        return { ticket, chips, h }
      })
      const rowW = row.length * nodeW + (row.length - 1) * HGAP
      const rowH = Math.max(...row.map((r) => r.h))
      row.forEach((r, j) =>
        nodes.push({
          ticket: r.ticket,
          x: (width - rowW) / 2 + j * (nodeW + HGAP),
          y,
          w: nodeW,
          h: rowH,
          decision: isDecision(r.ticket),
          chips: r.chips,
        }),
      )
      y += rowH + VGAP
    }
  }

  const at = new Map(nodes.map((n) => [n.ticket.id, n]))
  const edges: Edge[] = nodes.flatMap((to) =>
    (to.ticket.depends_on ?? []).flatMap((dep) => {
      const from = at.get(dep)
      if (!from) return []
      const x1 = from.x + from.w / 2
      const y1 = from.y + from.h
      const x2 = to.x + to.w / 2
      const y2 = to.y
      const mid = (y1 + y2) / 2
      return [
        {
          from: dep,
          to: to.ticket.id,
          met: isDone(from.ticket),
          path: `M${x1},${y1} C${x1},${mid} ${x2},${mid} ${x2},${y2 - 2}`,
        },
      ]
    }),
  )
  return { nodes, edges, height: Math.max(0, y - VGAP) }
}

// layoutGraph turns a project's tickets into positioned clusters for a group
// area `width` CSS pixels wide. Cancelled tickets are left out. Archived
// tickets aren't drawn, but a dependency on one still shows as met, and an
// archived feature still heads its unarchived subtasks.
export function layoutGraph(all: Ticket[], width: number, opts: GraphOptions = {}): Cluster[] {
  const tickets = all.filter((t) => t.status !== 'cancelled')
  const byId = new Map(tickets.map((t) => [t.id, t]))
  const live = tickets.filter((t) => !t.archived_at)

  const parents = new Set(live.flatMap((t) => (t.parent && byId.has(t.parent) ? [t.parent] : [])))
  const features = tickets.filter((t) => parents.has(t.id) && t.type !== 'epic')
  const featureIds = new Set(features.map((f) => f.id))

  // Each ticket that isn't itself a feature belongs to its parent feature's
  // cluster, or to the "Unparented" cluster ('').
  const featureOf = new Map<string, string>()
  for (const t of live) {
    if (featureIds.has(t.id)) continue
    featureOf.set(t.id, t.parent && featureIds.has(t.parent) ? t.parent : '')
  }

  const order = featureOrder(
    features.map((f) => f.id),
    featureOf,
    live,
  )
  if ([...featureOf.values()].includes('')) order.push('')

  return order.map((fid) => {
    const feature = byId.get(fid)
    const members = live.filter((t) => featureOf.get(t.id) === fid)
    const after = [
      ...new Set(
        members.flatMap((t) =>
          (t.depends_on ?? []).flatMap((dep) => {
            const f = featureOf.get(dep)
            return f && f !== fid ? [f] : []
          }),
        ),
      ),
    ].map((f) => byId.get(f)?.title ?? f)
    const done = members.filter(isDone).length
    const folded = members.length > 0 && done === members.length

    const groups: Group[] = folded
      ? []
      : groupsOf(members).map(({ section, tickets: own }) => {
          const gdone = own.filter(isDone).length
          const gfolded = section !== undefined && gdone === own.length
          const shown = gfolded ? [] : opts.hideDone ? own.filter((t) => !isDone(t)) : own
          return { section, done: gdone, total: own.length, folded: gfolded, ...placeGroup(shown, width, byId) }
        })

    return {
      id: fid,
      feature,
      title: feature?.title ?? 'Unparented',
      done,
      total: members.length,
      after,
      folded,
      groups,
    }
  })
}
