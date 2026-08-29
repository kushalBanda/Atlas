import { useNavigate } from "react-router-dom";
import { useStats } from "../api/stats";
import { useTraces } from "../api/traces";
import { formatDurationNano } from "../lib/duration";
import { ErrorState } from "../components/ErrorState";

function formatCost(n: number): string {
  return `$${n.toFixed(n < 1 ? 4 : 2)}`;
}

function StatBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-2xl font-semibold tabular-nums">{value}</div>
      <div className="text-xs text-text-faint">{label}</div>
    </div>
  );
}

export function Home() {
  const navigate = useNavigate();
  const { data: stats, isPending: statsPending, isError: statsError, refetch: refetchStats } = useStats();
  const { data: recentTraces, isPending: tracesPending } = useTraces({ limit: 6 });

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <header className="border-b border-border px-5 py-3.5">
        <h1 className="m-0 text-sm font-semibold tracking-wide">Home</h1>
      </header>

      <div className="px-5 py-5">
        <div className="mb-6 rounded-md border border-border bg-surface px-5 py-4">
          <h2 className="m-0 text-base font-semibold">
            One pipeline for server code and AI calls.
          </h2>
          <p className="mt-1.5 whitespace-nowrap text-[13px] text-text-muted">
            Atlas ingests OTLP from any service, HTTP handlers, database calls, LLM prompts, and finds the root cause automatically: every span, every model call, one trace.
          </p>
        </div>

        {statsError && (
          <ErrorState message="Could not load stats. Check that atlas-server is running." onRetry={refetchStats} />
        )}

        {!statsError && (
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-md border border-border bg-surface px-5 py-4">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="m-0 text-[13px] font-medium text-text-muted">Traces</h3>
              </div>
              {statsPending ? (
                <div className="h-8 w-16 animate-pulse rounded bg-elevated" />
              ) : (
                <StatBlock label="Total traces tracked" value={String(stats?.TotalTraces ?? 0)} />
              )}
            </div>

            <div className="rounded-md border border-border bg-surface px-5 py-4">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="m-0 text-[13px] font-medium text-text-muted">Issues found</h3>
              </div>
              {statsPending ? (
                <div className="h-8 w-16 animate-pulse rounded bg-elevated" />
              ) : (
                <StatBlock
                  label="Traces with a root cause"
                  value={String(stats?.TracesWithRootCause ?? 0)}
                />
              )}
            </div>

            <div className="rounded-md border border-border bg-surface px-5 py-4">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="m-0 text-[13px] font-medium text-text-muted">Model cost</h3>
              </div>
              {statsPending ? (
                <div className="h-8 w-16 animate-pulse rounded bg-elevated" />
              ) : (
                <StatBlock label="Total LLM cost tracked" value={formatCost(stats?.LLM.TotalCost ?? 0)} />
              )}
            </div>
          </div>
        )}

        {!statsError && stats && stats.LLM.Models && stats.LLM.Models.length > 0 && (
          <div className="mt-4 rounded-md border border-border bg-surface px-5 py-4">
            <h3 className="m-0 mb-3 text-[13px] font-medium text-text-muted">Model usage</h3>
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr>
                  <th className="border-b border-border pb-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Model</th>
                  <th className="border-b border-border pb-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Calls</th>
                  <th className="border-b border-border pb-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Tokens</th>
                  <th className="border-b border-border pb-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Cost</th>
                </tr>
              </thead>
              <tbody>
                {stats.LLM.Models.map((m) => (
                  <tr key={m.Model} className="border-b border-border last:border-none">
                    <td className="py-2 font-plex-mono">{m.Model}</td>
                    <td className="py-2 font-plex-mono text-text-muted">{m.Calls}</td>
                    <td className="py-2 font-plex-mono text-text-muted">
                      {m.PromptTokens + m.CompletionTokens}
                    </td>
                    <td className="py-2 font-plex-mono text-text-muted">{formatCost(m.Cost)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="mt-4 rounded-md border border-border bg-surface px-5 py-4">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="m-0 text-[13px] font-medium text-text-muted">Recent traces</h3>
            <button
              type="button"
              onClick={() => navigate("/traces")}
              className="rounded-md border border-border px-2.5 py-1 text-xs text-text-muted hover:border-accent hover:text-accent"
            >
              Open Tracing
            </button>
          </div>

          {tracesPending && <div className="h-24 animate-pulse rounded bg-elevated" />}
          {!tracesPending && (!recentTraces || recentTraces.traces.length === 0) && (
            <div className="py-4 text-sm text-text-faint">
              No traces yet. Traces appear once Atlas receives OTLP data.
            </div>
          )}
          {!tracesPending && recentTraces && recentTraces.traces.length > 0 && (
            <table className="w-full border-collapse text-[13px]">
              <tbody>
                {recentTraces.traces.map((t) => (
                  <tr
                    key={t.TraceID}
                    className="cursor-pointer border-b border-border last:border-none hover:bg-elevated"
                    onClick={() => navigate(`/traces/${t.TraceID}`)}
                  >
                    <td className="py-2 font-plex-mono text-text-muted">{t.TraceID}</td>
                    <td className="py-2 font-plex-mono text-text-faint">
                      {new Date(t.FirstSeen).toLocaleTimeString()}
                    </td>
                    <td className="py-2 font-plex-mono text-text-faint">
                      {formatDurationNano(t.duration_nano)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <div className="mt-4">
          <button
            type="button"
            onClick={() => navigate("/discovery")}
            className="rounded-md border border-border bg-surface px-3 py-1.5 text-xs text-text-muted hover:border-accent hover:text-accent"
          >
            Run Discovery
          </button>
        </div>
      </div>
    </main>
  );
}
