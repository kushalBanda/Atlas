import type { SpanNode } from "../../lib/spanTree";

const HEIGHT = 20;

export interface MinimapProps {
  rows: SpanNode[];
  traceStartMs: number;
  traceDurationMs: number;
  onJump: (spanId: string) => void;
}

// Full-trace overview strip, unaffected by the row filter, so a long trace
// or an error cluster stays visible even while the row list below is
// filtered down. Click jumps the row list to that span.
export function Minimap({ rows, traceStartMs, traceDurationMs, onJump }: MinimapProps) {
  if (rows.length === 0) return null;

  return (
    <div
      className="relative flex-shrink-0 cursor-pointer border-b border-border bg-surface"
      style={{ height: HEIGHT }}
      title="Trace overview, click a mark to jump"
    >
      {rows.map((node) => {
        const startMs = new Date(node.span.StartTime).getTime();
        const leftPct = traceDurationMs > 0 ? ((startMs - traceStartMs) / traceDurationMs) * 100 : 0;
        const widthPct =
          traceDurationMs > 0 ? (node.span.duration_nano / 1_000_000 / traceDurationMs) * 100 : 0;
        const isError = node.span.StatusCode === "error";
        return (
          <div
            key={node.span.SpanID}
            className={`absolute top-1 h-2 rounded-[1px] ${isError ? "bg-error" : "bg-[#6ba3d6]/70"}`}
            style={{ left: `${leftPct}%`, width: `${Math.max(widthPct, 0.3)}%` }}
            onClick={(e) => {
              e.stopPropagation();
              onJump(node.span.SpanID);
            }}
          />
        );
      })}
    </div>
  );
}
