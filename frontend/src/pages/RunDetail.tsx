import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { AlertTriangle, Bot, Coins, Layers, Sparkles, TriangleAlert, User } from "lucide-react";
import { useRun } from "../api/runs";
import { RunGraph } from "../components/rungraph/RunGraph";
import { SpanDetailPanel } from "../components/SpanDetailPanel";
import { ErrorState } from "../components/ErrorState";
import { LinkPill, Pill, PillDot } from "../components/Pill";

export function RunDetail() {
  const { runId = "" } = useParams<{ runId: string }>();
  const { data, isPending, isError, error, refetch } = useRun(runId);
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);

  if (isPending) {
    return (
      <main className="flex flex-1 items-center justify-center text-sm text-text-faint">
        Loading run.
      </main>
    );
  }

  if (isError) {
    if (error.status === 404) {
      return (
        <main className="flex flex-1 flex-col items-center justify-center gap-1 text-sm text-text-faint">
          <span>Run not found.</span>
          <Link to="/runs" className="text-accent hover:underline">
            Back to runs
          </Link>
        </main>
      );
    }
    return (
      <main className="flex flex-1 flex-col">
        <ErrorState message="Could not load run." onRetry={refetch} />
      </main>
    );
  }

  const selectedSpan = data.spans.find((s) => s.SpanID === selectedSpanId) ?? null;
  const run = data.run;

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <header className="border-b border-border px-5 py-3.5">
        <div className="flex items-center gap-2">
          <Bot className="h-4 w-4 shrink-0 text-violet-400" strokeWidth={2.25} />
          <h1 className="m-0 text-sm font-semibold tracking-wide text-text-primary">
            {run?.AgentName ?? "Agent run"}
          </h1>
          <span className="font-plex-mono text-xs text-text-faint">{data.graph.run_id}</span>
        </div>

        {run ? (
          <div className="mt-2.5 flex flex-wrap items-center gap-2">
            {run.SessionID && (
              <LinkPill to={`/runs?session_id=${encodeURIComponent(run.SessionID)}`} title="View session">
                session <span className="text-text-primary">{run.SessionID}</span>
              </LinkPill>
            )}
            {run.UserID && (
              <Pill title="User">
                <User className="h-3 w-3" strokeWidth={2.25} />
                {run.UserID}
              </Pill>
            )}
            <Pill title="Steps">
              <Layers className="h-3 w-3" strokeWidth={2.25} />
              {run.SpanCount} <PillDot /> {" "}
              <span className={run.ErrorCount > 0 ? "text-[#f3a2a5]" : ""}>{run.ErrorCount} errors</span>
            </Pill>
            <Pill title="Tokens">
              <Sparkles className="h-3 w-3" strokeWidth={2.25} />
              {run.PromptTokens + run.CompletionTokens} tok
            </Pill>
            <Pill title="Cost">
              <Coins className="h-3 w-3" strokeWidth={2.25} />${run.Cost.toFixed(4)}
            </Pill>
          </div>
        ) : null}

        {data.graph.repeats.length > 0 ? (
          <div className="mt-3 flex items-center gap-2 rounded-md border border-amber-400/30 bg-amber-400/10 px-3 py-2 text-xs text-amber-300">
            <TriangleAlert className="h-4 w-4 shrink-0" strokeWidth={2.25} />
            <span>
              {data.graph.repeats.length} repeated step group
              {data.graph.repeats.length === 1 ? "" : "s"} detected — possible tool loop or retry storm.
            </span>
          </div>
        ) : null}

        {run && run.ErrorCount > 0 && data.graph.repeats.length === 0 ? (
          <div className="mt-3 flex items-center gap-2 rounded-md border border-[#f3a2a5]/30 bg-[#f3a2a5]/10 px-3 py-2 text-xs text-[#f3a2a5]">
            <AlertTriangle className="h-4 w-4 shrink-0" strokeWidth={2.25} />
            <span>
              {run.ErrorCount} step{run.ErrorCount === 1 ? "" : "s"} failed in this run.
            </span>
          </div>
        ) : null}
      </header>

      <RunGraph graph={data.graph} selectedSpanId={selectedSpanId} onSelect={setSelectedSpanId} />

      <SpanDetailPanel span={selectedSpan} onClose={() => setSelectedSpanId(null)} />
    </main>
  );
}
