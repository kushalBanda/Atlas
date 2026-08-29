import { useMemo, useState } from "react";
import type { Span } from "../../api/types";
import { buildSpanTree, flattenSpanTree, type SpanNode } from "../../lib/spanTree";
import { formatDurationNano } from "../../lib/duration";
import { colorForService } from "../../lib/serviceColor";

const ROW_HEIGHT = 24;
const MIN_LABEL_WIDTH_PCT = 4;

export interface FlameGraphProps {
  spans: Span[];
  selectedSpanId?: string | null;
  rootCauseSpanId?: string | null;
  onSelectSpan: (spanId: string) => void;
}

// Icicle-style flame graph: x = time (same math as Waterfall's TimelineBar),
// y = call depth. Time position is preserved (unlike a profiler flame graph,
// which merges same-named frames and drops the time axis) because Atlas
// traces are multi-service and shallow, and losing time order loses the
// "what happened when" read that matters for root-cause work.
export function FlameGraph({ spans, selectedSpanId, rootCauseSpanId, onSelectSpan }: FlameGraphProps) {
  const [hovered, setHovered] = useState<string | null>(null);

  const rows = useMemo(() => flattenSpanTree(buildSpanTree(spans)), [spans]);
  const maxDepth = useMemo(() => rows.reduce((max, n) => Math.max(max, n.depth), 0), [rows]);

  const { traceStartMs, traceDurationMs } = useMemo(() => {
    if (spans.length === 0) return { traceStartMs: 0, traceDurationMs: 0 };
    const starts = spans.map((s) => new Date(s.StartTime).getTime());
    const ends = spans.map((s) => new Date(s.EndTime).getTime());
    const min = Math.min(...starts);
    const max = Math.max(...ends);
    return { traceStartMs: min, traceDurationMs: Math.max(max - min, 1) };
  }, [spans]);

  const hoveredNode = hovered ? rows.find((n) => n.span.SpanID === hovered) : null;

  return (
    <div className="m-5 mt-3.5 flex flex-1 flex-col overflow-hidden rounded-md border border-border">
      <div className="flex-shrink-0 border-b border-border bg-surface px-3 py-1.5 text-xs text-text-faint">
        {hoveredNode
          ? `${hoveredNode.span.ServiceName} · ${hoveredNode.span.Name} — ${formatDurationNano(hoveredNode.span.duration_nano)}`
          : "Hover a frame for details. Width = duration, depth = call nesting."}
      </div>
      <div className="relative flex-1 overflow-auto bg-canvas p-3">
        <div
          className="relative"
          style={{ height: (maxDepth + 1) * ROW_HEIGHT, minWidth: "100%" }}
        >
          {rows.map((node) => (
            <FlameFrame
              key={node.span.SpanID}
              node={node}
              traceStartMs={traceStartMs}
              traceDurationMs={traceDurationMs}
              selected={node.span.SpanID === selectedSpanId}
              isRootCause={node.span.SpanID === rootCauseSpanId}
              dim={Boolean(rootCauseSpanId) && node.span.SpanID !== rootCauseSpanId}
              onClick={onSelectSpan}
              onHover={setHovered}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

interface FlameFrameProps {
  node: SpanNode;
  traceStartMs: number;
  traceDurationMs: number;
  selected: boolean;
  isRootCause: boolean;
  dim: boolean;
  onClick: (spanId: string) => void;
  onHover: (spanId: string | null) => void;
}

function FlameFrame({
  node,
  traceStartMs,
  traceDurationMs,
  selected,
  isRootCause,
  dim,
  onClick,
  onHover,
}: FlameFrameProps) {
  const { span, depth } = node;
  const startMs = new Date(span.StartTime).getTime();
  const leftPct = traceDurationMs > 0 ? ((startMs - traceStartMs) / traceDurationMs) * 100 : 0;
  const widthPct = traceDurationMs > 0 ? (span.duration_nano / 1_000_000 / traceDurationMs) * 100 : 0;
  const isError = span.StatusCode === "error";
  const bg = isError ? "var(--color-error)" : colorForService(span.ServiceName);

  return (
    <div
      className={`absolute cursor-pointer overflow-hidden text-ellipsis whitespace-nowrap rounded-[2px] border border-canvas px-1 text-[10px] leading-[22px] text-white/90 ${
        dim ? "opacity-30" : ""
      } ${selected ? "ring-2 ring-accent" : isRootCause ? "ring-1 ring-accent" : ""}`}
      style={{
        left: `${leftPct}%`,
        width: `${Math.max(widthPct, 0.3)}%`,
        top: depth * ROW_HEIGHT,
        height: ROW_HEIGHT - 2,
        backgroundColor: bg,
      }}
      onClick={() => onClick(span.SpanID)}
      onMouseEnter={() => onHover(span.SpanID)}
      onMouseLeave={() => onHover(null)}
    >
      {widthPct > MIN_LABEL_WIDTH_PCT ? `${span.ServiceName} · ${span.Name}` : ""}
    </div>
  );
}
