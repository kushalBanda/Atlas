import type { ReactNode } from "react";

// Fixed pixel widths, single source shared by header and row. This is what
// Waterfall's two-column header (420px + flex-1) already does implicitly;
// SpanTable's native <table> did not (browser auto-layout sizes header and
// body cells independently once content differs, which is what produced
// the overlapping column labels). A column list declared once here and
// consumed by both GridHeaderRow and GridRow can't drift out of sync.
export interface GridColumn<K extends string = string> {
  key: K;
  label: string;
  width: number; // px
  align?: "left" | "right";
}

export interface GridHeaderRowProps<K extends string> {
  columns: GridColumn<K>[];
  sortKey?: K;
  sortDir?: "asc" | "desc";
  onSort?: (key: K) => void;
  className?: string;
}

export function GridHeaderRow<K extends string>({
  columns,
  sortKey,
  sortDir,
  onSort,
  className,
}: GridHeaderRowProps<K>) {
  return (
    <div
      className={`flex flex-shrink-0 border-b border-border bg-surface text-[11px] uppercase tracking-wide text-text-faint ${className ?? ""}`}
    >
      {columns.map((col) => (
        <div
          key={col.key}
          className={`flex-shrink-0 overflow-hidden px-3 py-1.5 ${col.align === "right" ? "text-right" : "text-left"} ${
            onSort ? "cursor-pointer select-none hover:text-text-primary" : ""
          }`}
          style={{ width: col.width }}
          onClick={() => onSort?.(col.key)}
        >
          {col.label}
          {sortKey === col.key ? (sortDir === "asc" ? " ↑" : " ↓") : ""}
        </div>
      ))}
    </div>
  );
}

export interface GridRowProps<K extends string> {
  columns: GridColumn<K>[];
  cells: Record<K, ReactNode>;
  className?: string;
  onClick?: () => void;
  dataSpanId?: string;
}

export function GridRow<K extends string>({ columns, cells, className, onClick, dataSpanId }: GridRowProps<K>) {
  return (
    <div
      data-span-id={dataSpanId}
      className={`flex border-b border-border text-xs ${onClick ? "cursor-pointer hover:bg-elevated" : ""} ${className ?? ""}`}
      onClick={onClick}
    >
      {columns.map((col) => (
        <div
          key={col.key}
          className={`flex-shrink-0 overflow-hidden text-ellipsis whitespace-nowrap px-3 py-1.5 ${col.align === "right" ? "text-right" : ""}`}
          style={{ width: col.width }}
        >
          {cells[col.key]}
        </div>
      ))}
    </div>
  );
}
