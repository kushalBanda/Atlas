import type { Span } from "../../api/types";
import { formatDurationNano } from "../../lib/duration";

export interface TimelineBarProps {
  span: Span;
  traceStartMs: number;
  traceDurationMs: number;
  selfTimeNano: number;
  isRootCause: boolean;
  dim: boolean;
}

// Percent-position bar, matching SigNoz's waterfall model (see
// docs/plans/atlas-frontend/design/Design.md 4.2). A darker sub-bar,
// right-aligned within the span's own bar, marks self-time (span duration
// minus direct children's durations) so the eye lands on where a span
// actually spent time itself, not time it was just waiting on children.
export function TimelineBar({
  span,
  traceStartMs,
  traceDurationMs,
  selfTimeNano,
  isRootCause,
  dim,
}: TimelineBarProps) {
  const startMs = new Date(span.StartTime).getTime();
  const leftPct = traceDurationMs > 0 ? ((startMs - traceStartMs) / traceDurationMs) * 100 : 0;
  const widthPct = traceDurationMs > 0 ? (span.duration_nano / 1_000_000 / traceDurationMs) * 100 : 0;
  const isError = span.StatusCode === "error";
  // A label placed outside the bar (left: 100% + 6px) runs off the visible
  // area once the bar itself sits near the trace's right edge, and the
  // ancestor's overflow-x-hidden then clips it silently. Past this
  // threshold, flip the label inside the bar, right-aligned, instead.
  const runsToEdge = leftPct + widthPct > 88;
  const selfTimePctOfBar = span.duration_nano > 0 ? (selfTimeNano / span.duration_nano) * 100 : 0;

  return (
    <div className="relative h-[30px] border-b border-border">
      <div
        className={`absolute top-1.5 h-[18px] rounded-sm ${isError ? "bg-error" : "bg-[#6ba3d6]"} ${dim ? "opacity-30" : ""} ${isRootCause ? "ring-1 ring-accent" : ""}`}
        style={{ left: `${leftPct}%`, width: `${Math.max(widthPct, 0.5)}%` }}
      >
        {selfTimePctOfBar > 0 && (
          <div
            className={`absolute right-0 top-0 h-full rounded-sm ${isError ? "bg-error/70 brightness-75" : "bg-[#3d6d99]"}`}
            style={{ width: `${selfTimePctOfBar}%` }}
          />
        )}
        {isError &&
          (runsToEdge ? (
            <span className="absolute right-1 top-0 whitespace-nowrap font-plex-mono text-[11px] text-white/90">
              {formatDurationNano(span.duration_nano)}
            </span>
          ) : (
            <span className="absolute left-[calc(100%+6px)] top-0 whitespace-nowrap font-plex-mono text-[11px] text-text-faint">
              {formatDurationNano(span.duration_nano)}
            </span>
          ))}
      </div>
    </div>
  );
}
