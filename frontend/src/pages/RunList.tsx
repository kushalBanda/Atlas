import { Bot, Coins, Layers, Sparkles } from "lucide-react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useRuns } from "../api/runs";
import { TableSkeleton } from "../components/TableSkeleton";
import { ErrorState } from "../components/ErrorState";
import { Pill, PillDot } from "../components/Pill";

export function RunList() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const sessionId = searchParams.get("session_id") ?? undefined;
  const { data, isPending, isError, refetch } = useRuns({ sessionId });

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <header className="flex items-baseline gap-2.5 border-b border-border px-5 py-3.5">
        <h1 className="m-0 text-sm font-semibold tracking-wide">Agent runs</h1>
        {sessionId && (
          <span className="font-plex-mono text-xs text-text-faint">in session {sessionId}</span>
        )}
        {data && <span className="text-xs text-text-faint">{data.runs.length} runs</span>}
      </header>

      {isPending && <TableSkeleton columns={4} />}
      {isError && (
        <ErrorState message="Could not load runs. Check that atlas-server is running." onRetry={refetch} />
      )}
      {data && data.runs.length === 0 && (
        <div className="px-5 py-4 text-sm text-text-faint">
          No agent runs yet. Tag spans with agent.run.id to see them here.
        </div>
      )}

      {data && data.runs.length > 0 && (
        <ul className="flex flex-col">
          {data.runs.map((run) => (
            <li key={run.RunID}>
              <button
                type="button"
                onClick={() => navigate(`/runs/${run.RunID}`)}
                className={`flex w-full items-center gap-3 border-b border-border px-5 py-3 text-left transition-colors hover:bg-surface motion-reduce:transition-none
                  ${run.ErrorCount > 0 ? "border-l-2 border-l-[#f3a2a5]/60" : "border-l-2 border-l-transparent"}`}
              >
                <Bot className="h-4 w-4 shrink-0 text-violet-400" strokeWidth={2.25} />

                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-sm font-medium text-text-primary">
                      {run.AgentName ?? "unnamed agent"}
                    </span>
                    <span className="truncate font-plex-mono text-[11px] text-text-faint">{run.RunID}</span>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-2">
                    <Pill>
                      <Layers className="h-3 w-3" strokeWidth={2.25} />
                      {run.SpanCount} steps
                      {run.ErrorCount > 0 && (
                        <>
                          <PillDot />
                          <span className="text-[#f3a2a5]">{run.ErrorCount} errors</span>
                        </>
                      )}
                    </Pill>
                    <Pill>
                      <Sparkles className="h-3 w-3" strokeWidth={2.25} />
                      {run.PromptTokens + run.CompletionTokens} tok
                    </Pill>
                    <Pill>
                      <Coins className="h-3 w-3" strokeWidth={2.25} />${run.Cost.toFixed(4)}
                    </Pill>
                    {run.SessionID && (
                      <span className="font-plex-mono text-[11px] text-text-faint">{run.SessionID}</span>
                    )}
                  </div>
                </div>

                <span className="shrink-0 font-plex-mono text-[11px] text-text-faint">
                  {new Date(run.LastSeen).toLocaleString()}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
