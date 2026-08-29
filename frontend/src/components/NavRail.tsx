import { NavLink } from "react-router-dom";

// Left nav rail. Fixed width, always shows the wordmark and link labels.
export function NavRail() {
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `flex h-9 w-full items-center gap-2.5 rounded-md px-2.5 text-[13px] font-medium tracking-wide transition-colors motion-reduce:transition-none ${
      isActive ? "bg-elevated text-accent" : "text-text-muted"
    }`;

  return (
    <nav className="flex w-[180px] flex-shrink-0 flex-col items-stretch gap-1 border-r border-border bg-surface px-2.5 pt-3">
      <div className="mb-5 flex items-center gap-2 px-0.5">
        <img src="/logo-mark.png" alt="Atlas" className="h-7 w-7 flex-shrink-0" />
        <span className="text-sm font-semibold tracking-wide">Atlas</span>
      </div>

      <NavLink to="/traces" title="Traces" className={linkClass}>
        Traces
      </NavLink>
      <NavLink to="/discovery" title="Discovery" className={linkClass}>
        Discovery
      </NavLink>

      <div className="flex-1" />
    </nav>
  );
}
