import type { Span } from "../../api/types";
import { formatDurationNano } from "../../lib/duration";

export interface TimelineBarProps {
  span: Span;
  traceStartMs: number;
  traceDurationMs: number;
}

// Percent-position bar, matching SigNoz's waterfall model (see
// docs/plans/atlas-frontend/design/Design.md 4.2).
export function TimelineBar({ span, traceStartMs, traceDurationMs }: TimelineBarProps) {
  const startMs = new Date(span.StartTime).getTime();
  const leftPct = traceDurationMs > 0 ? ((startMs - traceStartMs) / traceDurationMs) * 100 : 0;
  const widthPct = traceDurationMs > 0 ? (span.duration_nano / 1_000_000 / traceDurationMs) * 100 : 0;
  const isError = span.StatusCode === "error";

  return (
    <div className="relative h-[30px] border-b border-border">
      <div
        className={`absolute top-1.5 h-[18px] rounded-sm ${isError ? "bg-error" : "bg-[#6ba3d6]"}`}
        style={{ left: `${leftPct}%`, width: `${Math.max(widthPct, 0.5)}%` }}
      >
        {isError && (
          <span className="absolute left-[calc(100%+6px)] top-0 whitespace-nowrap font-plex-mono text-[11px] text-text-faint">
            {formatDurationNano(span.duration_nano)}
          </span>
        )}
      </div>
    </div>
  );
}
