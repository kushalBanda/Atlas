import { useMemo, useState } from "react";
import type { Span } from "../../api/types";
import { buildCallGraph, type CallNode } from "../../lib/callGraph";
import { formatDurationNano } from "../../lib/duration";
import { colorForService } from "../../lib/serviceColor";
import { GraphCanvas } from "../canvas/GraphCanvas";

const COL_WIDTH = 240;
const ROW_HEIGHT = 56;
const NODE_W = 190;
const NODE_H = 40;
const PAD = 30;

export interface ServiceMapProps {
  spans: Span[];
  selectedSpanId?: string | null;
  onSelectSpan: (spanId: string) => void;
}

// Layered call graph, one node per (service, operation) pair, columns by
// call depth (not by service). A single-service agent trace still fans out
// into many depth layers under that one service — a service-level map would
// collapse the whole trace to one circle, which is exactly the gap this
// replaces. Repeated calls to the same operation collapse into one node
// with a call count, so a loop of 7 identical calls reads as one node, not
// 7 overlapping boxes.
export function ServiceMap({ spans, selectedSpanId, onSelectSpan }: ServiceMapProps) {
  const [hovered, setHovered] = useState<string | null>(null);
  const { nodes, edges } = useMemo(() => buildCallGraph(spans), [spans]);

  const { positions, width, height, columns } = useMemo(() => {
    const byDepth = new Map<number, CallNode[]>();
    for (const node of nodes) {
      const list = byDepth.get(node.depth) ?? [];
      list.push(node);
      byDepth.set(node.depth, list);
    }
    for (const list of byDepth.values()) list.sort((a, b) => b.callCount - a.callCount);

    const maxDepth = nodes.reduce((m, n) => Math.max(m, n.depth), 0);
    const maxRows = [...byDepth.values()].reduce((m, list) => Math.max(m, list.length), 1);

    const pos = new Map<string, { x: number; y: number }>();
    for (const [depth, list] of byDepth) {
      list.forEach((node, i) => {
        pos.set(node.key, {
          x: PAD + depth * COL_WIDTH,
          y: PAD + i * ROW_HEIGHT + (maxRows - list.length) * (ROW_HEIGHT / 2),
        });
      });
    }

    return {
      positions: pos,
      width: PAD * 2 + (maxDepth + 1) * COL_WIDTH,
      height: PAD * 2 + maxRows * ROW_HEIGHT,
      columns: maxDepth + 1,
    };
  }, [nodes]);

  if (nodes.length === 0) {
    return (
      <div className="m-5 mt-3.5 flex flex-1 items-center justify-center rounded-md border border-border text-xs text-text-faint">
        No spans.
      </div>
    );
  }

  const hoveredNode = hovered ? nodes.find((n) => n.key === hovered) : null;
  const hoveredEdge = hovered ? edges.find((e) => `${e.from}->${e.to}` === hovered) : null;

  return (
    <div className="m-5 mt-3.5 flex flex-1 flex-col overflow-hidden rounded-md border border-border">
      <div className="flex-shrink-0 border-b border-border bg-surface px-3 py-1.5 text-xs text-text-faint">
        {hoveredNode
          ? `${hoveredNode.service} · ${hoveredNode.name} — ${hoveredNode.callCount} call${hoveredNode.callCount === 1 ? "" : "s"}, ${formatDurationNano(hoveredNode.totalDurationNano)} total${hoveredNode.errorCount > 0 ? `, ${hoveredNode.errorCount} error` : ""}`
          : hoveredEdge
            ? `${hoveredEdge.count} call${hoveredEdge.count === 1 ? "" : "s"}${hoveredEdge.errorCount > 0 ? `, ${hoveredEdge.errorCount} error` : ""}`
            : `${nodes.length} operation${nodes.length === 1 ? "" : "s"} across ${columns} layer${columns === 1 ? "" : "s"}`}
      </div>
      <GraphCanvas contentWidth={width} contentHeight={height}>
        <svg width={width} height={height} className="block">
          <defs>
            <marker id="cg-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
              <path d="M0,0 L10,5 L0,10 z" fill="var(--color-text-faint)" />
            </marker>
            <marker id="cg-arrow-error" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
              <path d="M0,0 L10,5 L0,10 z" fill="var(--color-error)" />
            </marker>
          </defs>

          {edges.map((edge) => {
            const from = positions.get(edge.from);
            const to = positions.get(edge.to);
            if (!from || !to) return null;
            const hasError = edge.errorCount > 0;
            const key = `${edge.from}->${edge.to}`;
            const x1 = from.x + NODE_W;
            const y1 = from.y + NODE_H / 2;
            const x2 = to.x - 6;
            const y2 = to.y + NODE_H / 2;
            const midX = (x1 + x2) / 2;

            return (
              <g key={key} onMouseEnter={() => setHovered(key)} onMouseLeave={() => setHovered(null)}>
                <path
                  d={`M ${x1} ${y1} C ${midX} ${y1}, ${midX} ${y2}, ${x2} ${y2}`}
                  fill="none"
                  stroke={hasError ? "var(--color-error)" : "var(--color-text-faint)"}
                  strokeWidth={hasError ? 2 : 1.25}
                  markerEnd={hasError ? "url(#cg-arrow-error)" : "url(#cg-arrow)"}
                  opacity={hovered && hovered !== key ? 0.25 : 1}
                />
                {edge.count > 1 && (
                  <text x={midX} y={(y1 + y2) / 2 - 4} textAnchor="middle" fontSize="9" fill="var(--color-text-muted)">
                    {edge.count}x
                  </text>
                )}
              </g>
            );
          })}

          {nodes.map((node) => {
            const pos = positions.get(node.key);
            if (!pos) return null;
            const hasError = node.errorCount > 0;
            const dim = Boolean(hovered) && hovered !== node.key && !edges.some(
              (e) => `${e.from}->${e.to}` === hovered && (e.from === node.key || e.to === node.key),
            );
            const selected = Boolean(selectedSpanId) && node.spanIds.includes(selectedSpanId!);
            return (
              <g
                key={node.key}
                className="cursor-pointer"
                onMouseEnter={() => setHovered(node.key)}
                onMouseLeave={() => setHovered(null)}
                onClick={() => onSelectSpan(node.spanIds[0])}
                opacity={dim ? 0.35 : 1}
              >
                <rect
                  x={pos.x}
                  y={pos.y}
                  width={NODE_W}
                  height={NODE_H}
                  rx={5}
                  fill={selected ? "var(--color-accent-dim)" : "var(--color-surface)"}
                  stroke={hasError ? "var(--color-error)" : selected ? "var(--color-accent)" : colorForService(node.service)}
                  strokeWidth={hasError || selected ? 2 : 1.5}
                />
                <circle cx={pos.x + 12} cy={pos.y + NODE_H / 2} r={4} fill={colorForService(node.service)} />
                <text x={pos.x + 22} y={pos.y + 16} fontSize="10" fill="var(--color-text-faint)">
                  {node.service}
                </text>
                <text
                  x={pos.x + 22}
                  y={pos.y + 30}
                  fontSize="11"
                  fill="var(--color-text-primary)"
                  className="overflow-hidden"
                >
                  {node.name.length > 22 ? `${node.name.slice(0, 21)}…` : node.name}
                </text>
                {node.callCount > 1 && (
                  <text x={pos.x + NODE_W - 8} y={pos.y + 16} textAnchor="end" fontSize="9" fill="var(--color-accent)">
                    ×{node.callCount}
                  </text>
                )}
                <title>
                  {node.service} · {node.name}: {node.callCount} calls, {formatDurationNano(node.totalDurationNano)} total
                  {hasError ? `, ${node.errorCount} error` : ""}
                </title>
              </g>
            );
          })}
        </svg>
      </GraphCanvas>
    </div>
  );
}
