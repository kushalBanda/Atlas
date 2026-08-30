import { GitBranch } from "lucide-react";
import { layoutRunGraph } from "../../lib/runGraph";
import type { RunGraph as RunGraphData } from "../../api/types";
import { GraphCanvas } from "../canvas/GraphCanvas";
import { RunGraphNode } from "./RunGraphNode";

export interface RunGraphProps {
  graph: RunGraphData;
  selectedSpanId: string | null;
  onSelect: (spanId: string) => void;
}

// Pixel geometry the connectors are computed from. Nodes are absolutely
// positioned on this grid (not flex-flowed) specifically so every edge can
// be drawn as a real elbow line between two known points — a plain
// downward arrow can't show which parent feeds which child once a node
// fans out to more than one.
const NODE_WIDTH = 192; // w-48
const NODE_HEIGHT = 84;
const COL_GAP = 24;
const ROW_GAP = 48;

// Rendered inside the shared GraphCanvas (pan/zoom viewport, also used by
// ServiceMap) instead of a scrolling div — a run graph has no natural size
// limit, so a scrollbar stops being usable past a handful of steps.
export function RunGraph({ graph, selectedSpanId, onSelect }: RunGraphProps) {
  const layout = layoutRunGraph(graph);
  const repeatCount = new Map<string, number>();
  for (const r of graph.repeats) {
    for (const id of r.span_ids) repeatCount.set(id, r.count);
  }

  if (layout.nodes.length === 0) {
    return <p className="px-5 py-4 text-sm text-text-faint">This run has no steps.</p>;
  }

  const posBySpanId = new Map(
    layout.nodes.map((n) => [
      n.span_id,
      {
        x: n.column * (NODE_WIDTH + COL_GAP),
        y: n.depth * (NODE_HEIGHT + ROW_GAP),
      },
    ]),
  );
  const maxColumns = Math.max(...layout.nodes.map((n) => n.column)) + 1;
  const width = maxColumns * (NODE_WIDTH + COL_GAP) - COL_GAP;
  const height = layout.depth * (NODE_HEIGHT + ROW_GAP) - ROW_GAP;

  return (
    <GraphCanvas contentWidth={width} contentHeight={height}>
      <svg className="pointer-events-none absolute left-0 top-0 overflow-visible" width={width} height={height}>
        <defs>
          <marker id="run-graph-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M0,0 L8,4 L0,8 z" className="fill-border" />
          </marker>
          <marker id="run-graph-arrow-cross" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M0,0 L8,4 L0,8 z" className="fill-accent" />
          </marker>
        </defs>
        {graph.edges.map((edge) => {
          const from = posBySpanId.get(edge.from);
          const to = posBySpanId.get(edge.to);
          if (!from || !to) return null;
          const x1 = from.x + NODE_WIDTH / 2;
          const y1 = from.y + NODE_HEIGHT;
          const x2 = to.x + NODE_WIDTH / 2;
          const y2 = to.y;
          const midY = y1 + (y2 - y1) / 2;
          return (
            <path
              key={`${edge.from}-${edge.to}`}
              d={`M${x1},${y1} L${x1},${midY} L${x2},${midY} L${x2},${y2}`}
              fill="none"
              strokeWidth={1.5}
              strokeDasharray={edge.cross_trace ? "4 3" : undefined}
              className={edge.cross_trace ? "stroke-accent" : "stroke-border"}
              markerEnd={`url(#${edge.cross_trace ? "run-graph-arrow-cross" : "run-graph-arrow"})`}
            />
          );
        })}
      </svg>

      {graph.edges
        .filter((e) => e.cross_trace)
        .map((edge) => {
          const from = posBySpanId.get(edge.from);
          const to = posBySpanId.get(edge.to);
          if (!from || !to) return null;
          const x1 = from.x + NODE_WIDTH / 2;
          const y1 = from.y + NODE_HEIGHT;
          const x2 = to.x + NODE_WIDTH / 2;
          const y2 = to.y;
          const midY = y1 + (y2 - y1) / 2;
          return (
            <span
              key={`${edge.from}-${edge.to}-label`}
              className="absolute inline-flex -translate-x-1/2 -translate-y-1/2 items-center gap-1 whitespace-nowrap rounded-sm border border-border bg-canvas px-1.5 py-0.5 font-plex-mono text-[10px] uppercase tracking-wide text-accent"
              style={{ left: x1 + (x2 - x1) / 2, top: midY }}
            >
              <GitBranch className="h-3 w-3" strokeWidth={2.25} />
              handoff
            </span>
          );
        })}

      {layout.nodes.map((node) => {
        const pos = posBySpanId.get(node.span_id)!;
        return (
          <div key={node.span_id} className="absolute" style={{ left: pos.x, top: pos.y }}>
            <RunGraphNode
              node={node}
              repeatCount={repeatCount.get(node.span_id)}
              selected={node.span_id === selectedSpanId}
              onSelect={onSelect}
            />
          </div>
        );
      })}
    </GraphCanvas>
  );
}
