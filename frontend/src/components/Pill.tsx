import { Link } from "react-router-dom";
import type { ReactNode } from "react";

// Small mono chip for metric/metadata rows, e.g. "4 steps · 2 errors · $0.02".
// Borrowed from Langfuse's ModernSessionHeaderPill (docs/resources/langfuse
// web/src/components/session/ModernSessionHeaderPill.tsx), reworked onto
// Atlas's own dark-charcoal/orange tokens instead of shadcn's palette.
const PILL_CLASS =
  "inline-flex h-[22px] shrink-0 items-center gap-1.5 whitespace-nowrap rounded-sm border border-border px-2 font-plex-mono text-[11px] leading-none text-text-muted";

export function Pill({ children, title }: { children: ReactNode; title?: string }) {
  return (
    <span title={title} className={PILL_CLASS}>
      {children}
    </span>
  );
}

export function LinkPill({ to, children, title }: { to: string; children: ReactNode; title?: string }) {
  return (
    <Link
      to={to}
      title={title}
      className={`${PILL_CLASS} group hover:border-accent hover:text-accent`}
    >
      {children}
    </Link>
  );
}

// Separator dot between pills in a metric row, matching Langfuse's ChipDot.
export function PillDot() {
  return <span className="text-text-faint">·</span>;
}
