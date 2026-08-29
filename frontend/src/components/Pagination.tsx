export interface PaginationProps {
  page: number; // 1-indexed
  pageCount: number;
  onPrev: () => void;
  onNext: () => void;
  pageSize: number;
}

// Client-side only: slices an already-fetched batch, never refetches.
// See docs/plans/atlas-frontend/02-architecture.md's pagination correction.
export function Pagination({ page, pageCount, onPrev, onNext, pageSize }: PaginationProps) {
  return (
    <div className="flex items-center gap-2.5 border-t border-border px-5 py-2.5 text-xs text-text-faint">
      <span>
        Rows per page <span className="font-plex-mono text-text-primary">{pageSize}</span>
      </span>
      <div className="flex-1" />
      <span>
        Page <span className="text-text-primary">{page}</span> of {pageCount}
      </span>
      <button
        type="button"
        disabled={page <= 1}
        onClick={onPrev}
        className="rounded-md border border-border bg-surface px-3 py-1.5 text-xs text-text-muted disabled:cursor-default disabled:opacity-50 enabled:hover:border-accent enabled:hover:text-accent"
      >
        Prev
      </button>
      <button
        type="button"
        disabled={page >= pageCount}
        onClick={onNext}
        className="rounded-md border border-border bg-surface px-3 py-1.5 text-xs text-text-muted disabled:cursor-default disabled:opacity-50 enabled:hover:border-accent enabled:hover:text-accent"
      >
        Next
      </button>
    </div>
  );
}
