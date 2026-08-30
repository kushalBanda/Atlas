import type { GraphEdge, GraphNode, RunGraph } from '../api/types'

/** A graph node with its computed grid position. */
export interface LaidOutNode extends GraphNode {
  /** Row: longest path from any root. */
  depth: number
  /** Column within the row, assigned left to right in node order. */
  column: number
}

export interface RunLayout {
  nodes: LaidOutNode[]
  edges: GraphEdge[]
  /** Widest row's node count. */
  width: number
  /** Number of rows. */
  depth: number
}

/**
 * layoutRunGraph assigns each node a depth and a column, so a run graph can
 * be drawn as a layered grid. Depth is the longest path from a root, so a
 * node never sits above one of its parents.
 *
 * The backend graph is normally a DAG, but a malformed or hand-written span
 * set could contain a cycle. The traversal caps each node's visit count, so
 * a cycle degrades to a stable layout instead of hanging the browser.
 */
export function layoutRunGraph(graph: RunGraph): RunLayout {
  if (graph.nodes.length === 0) {
    return { nodes: [], edges: [], width: 0, depth: 0 }
  }

  const children = new Map<string, string[]>()
  const hasParent = new Set<string>()
  for (const e of graph.edges) {
    if (!children.has(e.from)) children.set(e.from, [])
    children.get(e.from)!.push(e.to)
    hasParent.add(e.to)
  }

  const depths = new Map<string, number>()
  const queue: Array<{ id: string; depth: number }> = graph.nodes
    .filter((n) => !hasParent.has(n.span_id))
    .map((n) => ({ id: n.span_id, depth: 0 }))

  // A node in a cycle has no root, so seed any node still unreached.
  if (queue.length === 0) {
    queue.push({ id: graph.nodes[0].span_id, depth: 0 })
  }

  const maxVisits = graph.nodes.length + 1
  const visits = new Map<string, number>()
  while (queue.length > 0) {
    const { id, depth } = queue.shift()!
    const seen = (visits.get(id) ?? 0) + 1
    visits.set(id, seen)
    if (seen > maxVisits) continue

    const current = depths.get(id)
    if (current !== undefined && current >= depth) continue
    depths.set(id, depth)

    for (const child of children.get(id) ?? []) {
      queue.push({ id: child, depth: depth + 1 })
    }
  }

  const perDepth = new Map<number, number>()
  const nodes: LaidOutNode[] = graph.nodes.map((n) => {
    const depth = depths.get(n.span_id) ?? 0
    const column = perDepth.get(depth) ?? 0
    perDepth.set(depth, column + 1)
    return { ...n, depth, column }
  })

  return {
    nodes,
    edges: graph.edges,
    width: Math.max(...perDepth.values()),
    depth: perDepth.size,
  }
}
