import { useEffect, useState } from "react";
import { NavLink } from "react-router-dom";

const STORAGE_KEY = "atlas.navRail.expanded";

function readStoredExpanded(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

// Left nav rail. Collapses to icon-only (52px) or expands to show the
// wordmark and link labels (180px). State persists across reloads.
export function NavRail() {
  const [expanded, setExpanded] = useState(readStoredExpanded);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, expanded ? "1" : "0");
    } catch {
      // Storage may be unavailable (private mode); expansion just won't persist.
    }
  }, [expanded]);

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `flex h-9 items-center gap-2.5 rounded-md px-2.5 text-[11px] transition-colors motion-reduce:transition-none ${
      isActive ? "bg-elevated text-accent" : "text-text-muted"
    } ${expanded ? "w-full justify-start" : "w-9 justify-center"}`;

  return (
    <nav
      className={`flex flex-shrink-0 flex-col gap-1 border-r border-border bg-surface pt-4 transition-[width] duration-150 motion-reduce:transition-none ${
        expanded ? "w-[180px] items-stretch px-2.5" : "w-[52px] items-center"
      }`}
    >
      <div className={`mb-5 flex items-center gap-2 ${expanded ? "px-0.5" : ""}`}>
        <img src="/logo-mark.png" alt="Atlas" className="h-7 w-7 flex-shrink-0" />
        {expanded && <span className="text-sm font-semibold tracking-wide">Atlas</span>}
      </div>

      <NavLink to="/traces" title="Traces" className={linkClass}>
        {expanded ? "Traces" : "TR"}
      </NavLink>
      <NavLink to="/discovery" title="Discovery" className={linkClass}>
        {expanded ? "Discovery" : "DS"}
      </NavLink>

      <div className="flex-1" />
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        title={expanded ? "Collapse sidebar" : "Expand sidebar"}
        aria-label={expanded ? "Collapse sidebar" : "Expand sidebar"}
        className={`mb-3 flex h-9 items-center rounded-md text-text-faint hover:bg-elevated hover:text-text-muted ${
          expanded ? "w-full justify-start px-2.5 gap-2.5" : "w-9 justify-center"
        }`}
      >
        <span aria-hidden="true">{expanded ? "←" : "→"}</span>
        {expanded && <span className="text-[11px]">Collapse</span>}
      </button>
    </nav>
  );
}
