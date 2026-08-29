export interface TableSkeletonProps {
  columns: number;
  rows?: number;
}

// Matches the final row shape (Gate 1, design doc section 6), not a
// generic spinner.
export function TableSkeleton({ columns, rows = 6 }: TableSkeletonProps) {
  return (
    <table className="w-full border-collapse" aria-label="Loading">
      <tbody>
        {Array.from({ length: rows }).map((_, r) => (
          <tr key={r} className="border-b border-border">
            {Array.from({ length: columns }).map((_, c) => (
              <td key={c} className="px-5 py-2.5">
                <div className="h-3 animate-pulse rounded bg-elevated" style={{ width: `${40 + ((r + c) % 4) * 15}%` }} />
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
