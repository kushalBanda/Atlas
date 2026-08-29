import { useMemo, useRef, useState } from "react";
import type { Span } from "../../api/types";
import { buildSpanTree, flattenSpanTree } from "../../lib/spanTree";
import { formatDurationNano } from "../../lib/duration";
import { SpanRow } from "./SpanRow";
import { TimelineBar } from "./TimelineBar";

const TICK_COUNT = 5;

export interface WaterfallProps {
  spans: Span[];
  selectedSpanId?: string | null;
  onSelectSpan: (spanId: string) => void;
}

export function Waterfall({ spans, selectedSpanId, onSelectSpan }: WaterfallProps) {
  const [filter, setFilter] = useState("");
  const timelineRef = useRef<HTMLDivElement>(null);
  const [cursor, setCursor] = useState<{ x: number; pct: number } | null>(null);

  // Memoized on the spans array reference so selecting a span (parent
  // re-render) doesn't re-walk the tree every time (Gate 3, perf note).
  const rows = useMemo(() => flattenSpanTree(buildSpanTree(spans)), [spans]);

  const filteredRows = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter(
      (node) =>
        node.span.Name.toLowerCase().includes(q) || node.span.ServiceName.toLowerCase().includes(q),
    );
  }, [rows, filter]);

  const { traceStartMs, traceDurationMs } = useMemo(() => {
    if (spans.length === 0) return { traceStartMs: 0, traceDurationMs: 0 };
    const starts = spans.map((s) => new Date(s.StartTime).getTime());
    const ends = spans.map((s) => new Date(s.EndTime).getTime());
    const min = Math.min(...starts);
    const max = Math.max(...ends);
    return { traceStartMs: min, traceDurationMs: Math.max(max - min, 1) };
  }, [spans]);

  return (
    <div className="m-5 mt-3.5 flex flex-1 flex-col overflow-hidden rounded-md border border-border">
      <div className="flex-shrink-0 border-b border-border bg-surface px-3 py-1.5">
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter spans by name or service..."
          className="w-full max-w-xs bg-transparent text-xs text-text-primary placeholder:text-text-faint outline-none"
        />
      </div>

      <div className="flex flex-shrink-0">
        <div className="w-[420px] flex-shrink-0 border-b border-r border-border bg-canvas px-3 py-1.5 text-[11px] uppercase tracking-wide text-text-faint">
          Span
        </div>
        <div className="relative flex h-[27px] flex-1 border-b border-border pr-3 text-[11px] text-text-faint">
          {Array.from({ length: TICK_COUNT }, (_, i) => {
            const pct = (i / (TICK_COUNT - 1)) * 100;
            const tickNanos = ((traceDurationMs * i) / (TICK_COUNT - 1)) * 1_000_000;
            const isFirst = i === 0;
            const isLast = i === TICK_COUNT - 1;
            return (
              <span
                key={i}
                className="absolute top-1/2 whitespace-nowrap"
                style={{
                  left: `${pct}%`,
                  transform: `translate(${isFirst ? "0" : isLast ? "-100%" : "-50%"}, -50%)`,
                }}
              >
                {tickNanos === 0 ? "0ms" : formatDurationNano(tickNanos)}
              </span>
            );
          })}
        </div>
      </div>

      <div className="flex flex-1 overflow-y-auto overflow-x-hidden">
        <div className="flex w-[420px] flex-shrink-0 flex-col border-r border-border bg-surface">
          {filteredRows.map((node) => (
            <SpanRow
              key={node.span.SpanID}
              node={node}
              selected={node.span.SpanID === selectedSpanId}
              onClick={onSelectSpan}
            />
          ))}
          {filteredRows.length === 0 && (
            <div className="px-3 py-3 text-xs text-text-faint">No spans match "{filter}"</div>
          )}
        </div>
        <div
          ref={timelineRef}
          className="relative flex-1 bg-canvas pr-3"
          onMouseMove={(e) => {
            const rect = timelineRef.current?.getBoundingClientRect();
            if (!rect || rect.width === 0) return;
            const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width));
            setCursor({ x, pct: x / rect.width });
          }}
          onMouseLeave={() => setCursor(null)}
        >
          <div className="pointer-events-none absolute inset-0">
            {Array.from({ length: TICK_COUNT }, (_, i) => (
              <div
                key={i}
                className="absolute top-0 bottom-0 border-l border-border/60"
                style={{ left: `${(i / (TICK_COUNT - 1)) * 100}%` }}
              />
            ))}
          </div>
          {cursor && (
            <div
              className="pointer-events-none absolute top-0 bottom-0 z-10 border-l border-accent/70"
              style={{ left: cursor.x }}
            >
              <span
                className={`absolute -top-0.5 whitespace-nowrap rounded-sm bg-elevated px-1 py-0.5 font-plex-mono text-[10px] text-accent ${
                  cursor.pct > 0.9 ? "right-1" : "left-1"
                }`}
              >
                {formatDurationNano(cursor.pct * traceDurationMs * 1_000_000)}
              </span>
            </div>
          )}
          {filteredRows.map((node) => (
            <TimelineBar
              key={node.span.SpanID}
              span={node.span}
              traceStartMs={traceStartMs}
              traceDurationMs={traceDurationMs}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
