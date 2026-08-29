import type { Span, Trace } from "../api/types";

export interface VerdictLineProps {
  trace: Trace;
  spans: Span[];
  onJumpToSpan: (spanId: string) => void;
}

// The signature element (see docs/plans/atlas-frontend/mockups/trace-detail.html):
// Atlas prints its own diagnosis like a terminal, not a generic alert card.
export function VerdictLine({ trace, spans, onJumpToSpan }: VerdictLineProps) {
  if (!trace.ClosedAt) {
    return (
      <div className="mx-5 mt-4 rounded border border-border border-l-[3px] border-l-text-faint bg-surface px-4 py-2.5 font-plex-mono text-[12.5px] leading-relaxed">
        <span className="text-text-faint">&gt;</span> trace still open, no verdict yet
      </div>
    );
  }

  if (!trace.LikelyRootCauseSpanID) {
    return (
      <div className="mx-5 mt-4 rounded border border-border border-l-[3px] border-l-text-faint bg-surface px-4 py-2.5 font-plex-mono text-[12.5px] leading-relaxed">
        <span className="text-text-faint">&gt;</span> no clear root cause
      </div>
    );
  }

  const rootSpan = spans.find((s) => s.SpanID === trace.LikelyRootCauseSpanID);
  const isSelfTime = trace.Reason?.toLowerCase().includes("self-time") ?? false;

  return (
    <div
      className={`mx-5 mt-4 rounded border border-border bg-surface px-4 py-2.5 font-plex-mono text-[12.5px] leading-relaxed ${
        isSelfTime ? "border-l-[3px] border-l-accent" : "border-l-[3px] border-l-error"
      }`}
    >
      <div className="text-text-primary">
        <span className={isSelfTime ? "text-accent" : "text-error"}>&gt;</span> root cause:{" "}
        <span
          className="cursor-pointer text-text-primary underline decoration-border underline-offset-2 hover:decoration-error"
          onClick={() => onJumpToSpan(trace.LikelyRootCauseSpanID!)}
        >
          {rootSpan ? `${rootSpan.ServiceName} · ${rootSpan.Name}` : trace.LikelyRootCauseSpanID}
        </span>
      </div>
      <div className="pl-5 text-text-faint">
        {trace.Reason}
        {trace.SelfTimePct !== null ? `, ${Math.round(trace.SelfTimePct * 100)}% self time` : ""}
      </div>
    </div>
  );
}
