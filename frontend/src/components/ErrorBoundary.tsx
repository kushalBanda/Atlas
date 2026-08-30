import { Component, type ErrorInfo, type ReactNode } from "react";

export interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

// Last line of defense against an uncaught render/event exception taking
// down the whole app. Without this, React unmounts everything on an
// uncaught error — including the nav rail — leaving a blank page with no
// way back except a hard reload. Scoped around the routed page content
// only (see App.tsx), so a crash in one page still leaves navigation
// usable, the same guarantee the app already gives a page whose data
// fetch merely fails (see ErrorState).
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Unhandled error in page content", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <main className="flex flex-1 flex-col items-center justify-center gap-3 text-sm text-text-faint">
          <span>Something went wrong rendering this page.</span>
          <button
            type="button"
            onClick={() => this.setState({ error: null })}
            className="rounded-md border border-border bg-surface px-3 py-1.5 text-xs text-text-muted hover:border-accent hover:text-accent"
          >
            Try again
          </button>
        </main>
      );
    }
    return this.props.children;
  }
}
