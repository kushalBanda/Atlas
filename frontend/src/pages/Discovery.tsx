import type { DiscoveryTarget } from "../api/types";
import { useDiscoveryTargets } from "../api/discovery";
import { TableSkeleton } from "../components/TableSkeleton";
import { ErrorState } from "../components/ErrorState";

function TargetTable({ targets, showRule }: { targets: DiscoveryTarget[]; showRule: boolean }) {
  return (
    <table className="w-full overflow-hidden rounded-md border border-border">
      <thead>
        <tr>
          <th className="w-40 border-b border-border bg-surface px-3.5 py-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Host</th>
          <th className="w-20 border-b border-border bg-surface px-3.5 py-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Port</th>
          <th className="border-b border-border bg-surface px-3.5 py-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Process / image</th>
          {showRule && (
            <th className="border-b border-border bg-surface px-3.5 py-2 text-left text-[11px] font-medium uppercase tracking-wide text-text-faint">Receiver config</th>
          )}
        </tr>
      </thead>
      <tbody>
        {targets.map((target, i) => (
          <tr key={`${target.Host}:${target.Port}:${i}`} className="border-b border-border last:border-none hover:bg-surface">
            <td className="px-3.5 py-2.5 font-plex-mono text-text-primary">{target.Host}</td>
            <td className="px-3.5 py-2.5 font-plex-mono text-text-muted">{target.Port}</td>
            <td className="px-3.5 py-2.5 font-plex-mono">
              {target.ProcessOrImage}
              {target.ResourceAttributes &&
                Object.entries(target.ResourceAttributes).map(([k, v]) => (
                  <span key={k} className="ml-1 inline-block rounded bg-elevated px-1.5 py-0.5 text-[10px] text-text-faint">
                    {k}={v}
                  </span>
                ))}
            </td>
            {showRule && (
              <td className="px-3.5 py-2.5">
                {target.MatchedRule && (
                  <span className="rounded border border-border bg-elevated px-2 py-0.5 font-plex-mono text-[11.5px] text-accent">
                    {target.MatchedRule.ReceiverConfig}
                  </span>
                )}
              </td>
            )}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function Discovery() {
  const { data, isPending, isError, refetch } = useDiscoveryTargets();
  // Go's json.Marshal encodes a nil slice as null, not []. A discoverer
  // that found nothing (or errored, see RunAll) leaves matched/unrecognized
  // null, not empty, so normalize before touching .length or .map.
  const matched = data?.matched ?? [];
  const unrecognized = data?.unrecognized ?? [];

  return (
    <main className="flex-1 px-5 pb-5">
      <header className="flex items-baseline gap-2.5 border-b border-border py-3.5">
        <h1 className="m-0 text-sm font-semibold tracking-wide">Discovery</h1>
      </header>

      {isPending && <TableSkeleton columns={3} />}
      {isError && (
        <ErrorState message="Could not load discovery targets. Check that atlas-server is running." onRetry={refetch} />
      )}

      {data && (
        <>
          <h2 className="mb-2 mt-5 flex items-center gap-2 text-[11px] font-medium uppercase tracking-wide text-text-faint">
            Recognized
            <span className="rounded-full bg-elevated px-2 py-0.5 text-[11px] text-text-muted">{matched.length}</span>
          </h2>
          {matched.length === 0 ? (
            <div className="py-2 text-sm text-text-faint">No recognized services found yet.</div>
          ) : (
            <TargetTable targets={matched} showRule />
          )}

          <h2 className="mb-2 mt-5 flex items-center gap-2 text-[11px] font-medium uppercase tracking-wide text-text-faint">
            Unrecognized
            <span className="rounded-full bg-elevated px-2 py-0.5 text-[11px] text-text-muted">{unrecognized.length}</span>
          </h2>
          {unrecognized.length === 0 ? (
            <div className="py-2 text-sm text-text-faint">No unrecognized services found.</div>
          ) : (
            <TargetTable targets={unrecognized} showRule={false} />
          )}
        </>
      )}
    </main>
  );
}
