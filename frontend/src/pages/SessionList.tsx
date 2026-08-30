import { Clock, Coins, User, Workflow } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useSessions } from "../api/runs";
import { TableSkeleton } from "../components/TableSkeleton";
import { ErrorState } from "../components/ErrorState";
import { Pill } from "../components/Pill";

export function SessionList() {
  const navigate = useNavigate();
  const { data, isPending, isError, refetch } = useSessions();

  return (
    <main className="flex min-w-0 flex-1 flex-col">
      <header className="flex items-baseline gap-2.5 border-b border-border px-5 py-3.5">
        <h1 className="m-0 text-sm font-semibold tracking-wide">Sessions</h1>
        {data && <span className="text-xs text-text-faint">{data.sessions.length} sessions</span>}
      </header>

      {isPending && <TableSkeleton columns={4} />}
      {isError && (
        <ErrorState message="Could not load sessions. Check that atlas-server is running." onRetry={refetch} />
      )}
      {data && data.sessions.length === 0 && (
        <div className="px-5 py-4 text-sm text-text-faint">
          No sessions yet. Tag spans with session.id to see them here.
        </div>
      )}

      {data && data.sessions.length > 0 && (
        <ul className="flex flex-col">
          {data.sessions.map((session) => (
            <li key={session.SessionID}>
              <button
                type="button"
                onClick={() => navigate(`/runs?session_id=${encodeURIComponent(session.SessionID)}`)}
                className={`flex w-full items-center gap-3 border-b border-border px-5 py-3 text-left transition-colors hover:bg-surface motion-reduce:transition-none
                  ${session.ErrorCount > 0 ? "border-l-2 border-l-[#f3a2a5]/60" : "border-l-2 border-l-transparent"}`}
              >
                <Clock className="h-4 w-4 shrink-0 text-accent" strokeWidth={2.25} />

                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate font-plex-mono text-sm font-medium text-text-primary">
                      {session.SessionID}
                    </span>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-2">
                    {session.UserID && (
                      <Pill>
                        <User className="h-3 w-3" strokeWidth={2.25} />
                        {session.UserID}
                      </Pill>
                    )}
                    <Pill>
                      <Workflow className="h-3 w-3" strokeWidth={2.25} />
                      {session.RunCount} runs
                      {session.ErrorCount > 0 && (
                        <span className="text-[#f3a2a5]"> · {session.ErrorCount} errors</span>
                      )}
                    </Pill>
                    <Pill>
                      <Coins className="h-3 w-3" strokeWidth={2.25} />${session.Cost.toFixed(4)}
                    </Pill>
                  </div>
                </div>

                <span className="shrink-0 font-plex-mono text-[11px] text-text-faint">
                  {new Date(session.LastSeen).toLocaleString()}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
