import { describe, expect, it } from 'vitest'
import { layoutRunGraph } from './runGraph'
import type { RunGraph } from '../api/types'

function node(span_id: string, name = 'step') {
  return {
    span_id,
    trace_id: 't1',
    name,
    step_kind: 'tool',
    agent_name: 'researcher',
    service_name: 'svc',
    status_code: 'ok',
    start_time: '2026-08-29T12:00:00.000000000Z',
    duration_nano: 1_000_000,
    repeat_group: null,
  }
}

describe('layoutRunGraph', () => {
  it('assigns depth 0 to roots and increments down each edge', () => {
    const graph: RunGraph = {
      run_id: 'run-a',
      nodes: [node('s1'), node('s2'), node('s3')],
      edges: [
        { from: 's1', to: 's2', cross_trace: false },
        { from: 's2', to: 's3', cross_trace: false },
      ],
      repeats: [],
    }

    const layout = layoutRunGraph(graph)

    expect(layout.nodes.find((n) => n.span_id === 's1')?.depth).toBe(0)
    expect(layout.nodes.find((n) => n.span_id === 's2')?.depth).toBe(1)
    expect(layout.nodes.find((n) => n.span_id === 's3')?.depth).toBe(2)
  })

  it('places siblings at the same depth with different columns', () => {
    const graph: RunGraph = {
      run_id: 'run-a',
      nodes: [node('s1'), node('s2'), node('s3')],
      edges: [
        { from: 's1', to: 's2', cross_trace: false },
        { from: 's1', to: 's3', cross_trace: false },
      ],
      repeats: [],
    }

    const layout = layoutRunGraph(graph)
    const s2 = layout.nodes.find((n) => n.span_id === 's2')!
    const s3 = layout.nodes.find((n) => n.span_id === 's3')!

    expect(s2.depth).toBe(1)
    expect(s3.depth).toBe(1)
    expect(s2.column).not.toBe(s3.column)
  })

  it('treats a disconnected component as another root', () => {
    const graph: RunGraph = {
      run_id: 'run-a',
      nodes: [node('s1'), node('s2')],
      edges: [],
      repeats: [],
    }

    const layout = layoutRunGraph(graph)

    expect(layout.nodes.every((n) => n.depth === 0)).toBe(true)
    expect(layout.width).toBe(2)
  })

  it('does not loop forever on a cyclic edge set', () => {
    const graph: RunGraph = {
      run_id: 'run-a',
      nodes: [node('s1'), node('s2')],
      edges: [
        { from: 's1', to: 's2', cross_trace: false },
        { from: 's2', to: 's1', cross_trace: false },
      ],
      repeats: [],
    }

    const layout = layoutRunGraph(graph)

    expect(layout.nodes).toHaveLength(2)
  })

  it('returns an empty layout for an empty graph', () => {
    const layout = layoutRunGraph({ run_id: 'run-a', nodes: [], edges: [], repeats: [] })

    expect(layout.nodes).toHaveLength(0)
    expect(layout.width).toBe(0)
    expect(layout.depth).toBe(0)
  })
})
