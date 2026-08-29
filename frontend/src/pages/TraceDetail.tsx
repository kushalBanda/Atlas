import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTrace } from "../api/traces";
import { Waterfall } from "../components/waterfall/Waterfall";
import { VerdictLine } from "../components/VerdictLine";
import { SpanDetailPanel } from "../components/SpanDetailPanel";
import { ErrorState } from "../components/ErrorState";

export function TraceDetail() {
  const { traceId = "" } = useParams<{ traceId: string }>();
  const { data, isPending, isError, error, refetch } = useTrace(traceId);
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);

  if (isPending) {
    return (
      <main className="flex flex-1 items-center justify-center text-sm text-text-faint">
        Loading trace.
      </main>
    );
  }

  if (isError) {
    // 404 means the trace does not exist, retrying does not help; any
    // other status is transient, offer retry (Gate 2's error contract).
    if (error.status === 404) {
      return (
        <main className="flex flex-1 flex-col items-center justify-center gap-1 text-sm text-text-faint">
          <span>Trace not found.</span>
          <Link to="/traces" className="text-accent hover:underline">
            Back to traces
          </Link>
        </main>
      );
    }
    return (
      <main className="flex flex-1 flex-col">
        <ErrorState message="Could not load trace." onRetry={refetch} />
      </main>
    );
  }

  const selectedSpan = data.spans.find((s) => s.SpanID === selectedSpanId) ?? null;

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <header className="flex items-center gap-3 border-b border-border px-5 py-3">
        <span className="text-sm font-medium text-text-muted">
          Trace <span className="font-plex-mono text-text-primary">{data.trace.TraceID}</span>
        </span>
      </header>

      <VerdictLine trace={data.trace} spans={data.spans} onJumpToSpan={setSelectedSpanId} />

      <Waterfall spans={data.spans} selectedSpanId={selectedSpanId} onSelectSpan={setSelectedSpanId} />

      {selectedSpan && (
        <SpanDetailPanel span={selectedSpan} onClose={() => setSelectedSpanId(null)} />
      )}
    </main>
  );
}
