export interface ErrorStateProps {
  message: string;
  onRetry?: () => void;
}

// Scoped to the failed panel, never the whole page (Gate 1, design doc
// section 6): callers render this inside their own <main>, not App-level.
export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 py-10 text-sm text-text-faint">
      <span>{message}</span>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="rounded-md border border-border bg-surface px-3 py-1.5 text-xs text-text-muted hover:border-accent hover:text-accent"
        >
          Retry
        </button>
      )}
    </div>
  );
}
