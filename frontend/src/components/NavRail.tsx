import { NavLink } from "react-router-dom";

// Left nav rail. Fixed width, always shows the wordmark and link labels.
// Grouped like SigNoz/Langfuse: Home, then Discovery (setup action, not a
// data view, so it's outside Observability), then an "Observability"
// section for data views.
export function NavRail() {
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `flex h-8 w-full items-center gap-2.5 rounded-md px-2.5 text-[13px] font-medium tracking-wide transition-colors motion-reduce:transition-none ${
      isActive ? "bg-elevated text-accent" : "text-text-muted hover:text-text-primary"
    }`;

  return (
    <nav className="flex w-[180px] flex-shrink-0 flex-col items-stretch gap-1 border-r border-border bg-surface px-2.5 pt-3">
      <div className="mb-5 flex items-center gap-2 px-0.5">
        <img src="/logo-mark.png" alt="Atlas" className="h-7 w-7 flex-shrink-0" />
        <span className="text-sm font-semibold tracking-wide">Atlas</span>
      </div>

      <NavLink to="/" end title="Home" className={linkClass}>
        Home
      </NavLink>
      <NavLink to="/discovery" title="Discovery" className={linkClass}>
        Discovery
      </NavLink>

      <div className="mb-1 mt-4 px-2.5 text-[10.5px] font-semibold uppercase tracking-wider text-text-faint">
        Observability
      </div>
      <NavLink to="/traces" title="Tracing" className={linkClass}>
        Tracing
      </NavLink>

      <div className="flex-1" />
    </nav>
  );
}
