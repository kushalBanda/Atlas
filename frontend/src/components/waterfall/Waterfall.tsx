import { useMemo } from "react";
import type { Span } from "../../api/types";
import { buildSpanTree, flattenSpanTree } from "../../lib/spanTree";
import { SpanRow } from "./SpanRow";
import { TimelineBar } from "./TimelineBar";

export interface WaterfallProps {
  spans: Span[];
  selectedSpanId?: string | null;
  onSelectSpan: (spanId: string) => void;
}

export function Waterfall({ spans, selectedSpanId, onSelectSpan }: WaterfallProps) {
  // Memoized on the spans array reference so selecting a span (parent
  // re-render) doesn't re-walk the tree every time (Gate 3, perf note).
  const rows = useMemo(() => flattenSpanTree(buildSpanTree(spans)), [spans]);

  const { traceStartMs, traceDurationMs } = useMemo(() => {
    if (spans.length === 0) return { traceStartMs: 0, traceDurationMs: 0 };
    const starts = spans.map((s) => new Date(s.StartTime).getTime());
    const ends = spans.map((s) => new Date(s.EndTime).getTime());
    const min = Math.min(...starts);
    const max = Math.max(...ends);
    return { traceStartMs: min, traceDurationMs: Math.max(max - min, 1) };
  }, [spans]);

  return (
    <div className="m-5 mt-3.5 flex flex-1 overflow-hidden rounded-md border border-border">
      <div className="flex w-[420px] flex-shrink-0 flex-col overflow-y-auto border-r border-border bg-surface">
        <div className="border-b border-border bg-canvas px-3 py-1.5 text-[11px] uppercase tracking-wide text-text-faint">
          Span
        </div>
        {rows.map((node) => (
          <SpanRow
            key={node.span.SpanID}
            node={node}
            selected={node.span.SpanID === selectedSpanId}
            onClick={onSelectSpan}
          />
        ))}
      </div>
      <div className="relative flex-1 overflow-hidden bg-canvas">
        <div className="border-b border-border px-3 py-1.5 text-[11px] uppercase tracking-wide text-text-faint">
          0ms to {Math.round(traceDurationMs)}ms
        </div>
        {rows.map((node) => (
          <TimelineBar
            key={node.span.SpanID}
            span={node.span}
            traceStartMs={traceStartMs}
            traceDurationMs={traceDurationMs}
          />
        ))}
      </div>
    </div>
  );
}
