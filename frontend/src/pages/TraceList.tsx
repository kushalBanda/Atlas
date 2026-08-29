import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTraces } from "../api/traces";
import { formatDurationNano } from "../lib/duration";
import { deriveVerdictState, type VerdictState } from "../lib/verdict";
import { computeTraceStats } from "../lib/stats";
import { FilterBar, type RootCauseFilter } from "../components/FilterBar";
import { Pagination } from "../components/Pagination";
import { TableSkeleton } from "../components/TableSkeleton";
import { ErrorState } from "../components/ErrorState";
import { StatsStrip } from "../components/StatsStrip";

const STATUS_LABEL: Record<VerdictState, string> = {
  found: "Root cause found",
  ok: "OK",
  open: "Open",
};

const STATUS_CLASS: Record<VerdictState, string> = {
  found: "bg-error-dim text-[#f3a2a5]",
  ok: "bg-[#1f2b1e] text-[#a2d18a]",
  open: "bg-elevated text-text-muted",
};

const COLUMNS = ["Trace ID", "First seen", "Duration", "Status", "Root cause reason", "Self-time"];

// Client-side page size. Pagination has no backend cursor (see
// docs/plans/atlas-frontend/02-architecture.md), so the batch fetched by
// useTraces is capped at BATCH_LIMIT and sliced here.
const PAGE_SIZE = 25;
const BATCH_LIMIT = 200;

function rootCauseToFilterValue(rootCause: RootCauseFilter): boolean | undefined {
  if (rootCause === "found") return true;
  if (rootCause === "not-found") return false;
  return undefined;
}

export function TraceList() {
  const navigate = useNavigate();
  const [sinceMs, setSinceMs] = useState(60 * 60 * 1000); // default 1h
  const [rootCause, setRootCause] = useState<RootCauseFilter>("any");
  const [page, setPage] = useState(1);

  const { data, isPending, isError, refetch } = useTraces({
    sinceMs,
    hasRootCause: rootCauseToFilterValue(rootCause),
    limit: BATCH_LIMIT,
  });

  const pageCount = data ? Math.max(1, Math.ceil(data.traces.length / PAGE_SIZE)) : 1;
  const pageRows = data ? data.traces.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE) : [];
  const stats = useMemo(() => (data ? computeTraceStats(data.traces) : null), [data]);

  function handleFilterChange(fn: () => void) {
    fn();
    setPage(1); // any filter change resets to page 1
  }

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <header className="flex items-baseline gap-2.5 border-b border-border px-5 py-3.5">
        <h1 className="m-0 text-sm font-semibold tracking-wide">Traces</h1>
        {data && (
          <span className="text-xs text-text-faint">{data.traces.length} traces</span>
        )}
      </header>

      {stats && stats.count > 0 && <StatsStrip stats={stats} />}

      <FilterBar
        sinceMs={sinceMs}
        onSinceChange={(ms) => handleFilterChange(() => setSinceMs(ms))}
        rootCause={rootCause}
        onRootCauseChange={(value) => handleFilterChange(() => setRootCause(value))}
      />

      {isPending && <TableSkeleton columns={COLUMNS.length} />}
      {isError && (
        <ErrorState message="Could not load traces. Check that atlas-server is running." onRetry={refetch} />
      )}
      {data && data.traces.length === 0 && (
        <div className="px-5 py-4 text-sm text-text-faint">
          No traces yet. Traces appear once Atlas receives OTLP data.
        </div>
      )}

      {data && data.traces.length > 0 && (
        <>
          <table className="w-full border-collapse">
            <thead>
              <tr>
                {COLUMNS.map((col) => (
                  <th
                    key={col}
                    className="border-b border-border px-5 py-2.5 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint"
                  >
                    {col}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {pageRows.map((trace) => {
                const status = deriveVerdictState(trace);
                return (
                  <tr
                    key={trace.TraceID}
                    className="cursor-pointer border-b border-border hover:bg-surface"
                    onClick={() => navigate(`/traces/${trace.TraceID}`)}
                  >
                    <td className="px-5 py-2.5 font-plex-mono text-text-muted">{trace.TraceID}</td>
                    <td className="px-5 py-2.5 font-plex-mono">
                      {new Date(trace.FirstSeen).toLocaleTimeString()}
                    </td>
                    <td className="px-5 py-2.5 font-plex-mono">
                      {formatDurationNano(trace.duration_nano)}
                    </td>
                    <td className="px-5 py-2.5">
                      <span className={`inline-flex items-center gap-1.5 rounded px-2 py-0.5 text-xs ${STATUS_CLASS[status]}`}>
                        {STATUS_LABEL[status]}
                      </span>
                    </td>
                    <td className="px-5 py-2.5 text-text-muted">{trace.Reason ?? "-"}</td>
                    <td className="px-5 py-2.5 font-plex-mono text-text-faint">
                      {trace.SelfTimePct !== null ? `${Math.round(trace.SelfTimePct * 100)}%` : "-"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          <div className="flex-1" />
          <Pagination
            page={page}
            pageCount={pageCount}
            pageSize={PAGE_SIZE}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => Math.min(pageCount, p + 1))}
          />
        </>
      )}
    </main>
  );
}
