import { useMemo, useState } from "react";
import type { Span } from "../../api/types";
import { buildSpanTree, flattenSpanTree, type SpanNode } from "../../lib/spanTree";
import { formatDurationNano } from "../../lib/duration";
import { GridHeaderRow, GridRow, type GridColumn } from "./GridColumns";

type SortKey = "service" | "name" | "start" | "duration" | "self" | "cost";
type SortDir = "asc" | "desc";

export interface SpanTableProps {
  spans: Span[];
  selectedSpanId?: string | null;
  rootCauseSpanId?: string | null;
  onSelectSpan: (spanId: string) => void;
}

// Flat, sortable list, no tree nesting. Best for hunting the slowest span
// or the most expensive LLM call across a large trace, where the
// Waterfall's visual timeline is not the fastest path to an answer.
export function SpanTable({ spans, selectedSpanId, rootCauseSpanId, onSelectSpan }: SpanTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>("start");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const nodes = useMemo(() => flattenSpanTree(buildSpanTree(spans)), [spans]);
  const hasCost = useMemo(() => spans.some((s) => s.LLMCost != null), [spans]);
  const traceStartMs = useMemo(
    () => (spans.length === 0 ? 0 : Math.min(...spans.map((s) => new Date(s.StartTime).getTime()))),
    [spans],
  );

  const columns: GridColumn<SortKey>[] = useMemo(() => {
    const base: GridColumn<SortKey>[] = [
      { key: "service", label: "Service", width: 180 },
      { key: "name", label: "Span", width: 320 },
      { key: "start", label: "Start", width: 100, align: "right" },
      { key: "duration", label: "Duration", width: 100, align: "right" },
      { key: "self", label: "Self %", width: 80, align: "right" },
    ];
    if (hasCost) base.push({ key: "cost", label: "LLM Cost", width: 110, align: "right" });
    return base;
  }, [hasCost]);

  const rows = useMemo(() => {
    const withKeys = nodes.map((node) => ({
      node,
      startMs: new Date(node.span.StartTime).getTime() - traceStartMs,
      selfPct: node.span.duration_nano > 0 ? (node.selfTimeNano / node.span.duration_nano) * 100 : 0,
    }));
    const dir = sortDir === "asc" ? 1 : -1;
    withKeys.sort((a, b) => {
      switch (sortKey) {
        case "service":
          return dir * a.node.span.ServiceName.localeCompare(b.node.span.ServiceName);
        case "name":
          return dir * a.node.span.Name.localeCompare(b.node.span.Name);
        case "duration":
          return dir * (a.node.span.duration_nano - b.node.span.duration_nano);
        case "self":
          return dir * (a.selfPct - b.selfPct);
        case "cost":
          return dir * ((a.node.span.LLMCost ?? 0) - (b.node.span.LLMCost ?? 0));
        case "start":
        default:
          return dir * (a.startMs - b.startMs);
      }
    });
    return withKeys;
  }, [nodes, sortKey, sortDir, traceStartMs]);

  function toggleSort(key: SortKey) {
    if (key === sortKey) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  }

  return (
    <div className="m-5 mt-3.5 flex flex-1 flex-col overflow-hidden rounded-md border border-border">
      <div className="flex flex-1 flex-col overflow-auto">
        <GridHeaderRow columns={columns} sortKey={sortKey} sortDir={sortDir} onSort={toggleSort} className="sticky top-0 z-10" />
        {rows.map(({ node, startMs, selfPct }) => (
          <SpanTableRow
            key={node.span.SpanID}
            node={node}
            startMs={startMs}
            selfPct={selfPct}
            columns={columns}
            selected={node.span.SpanID === selectedSpanId}
            isRootCause={node.span.SpanID === rootCauseSpanId}
            onClick={onSelectSpan}
          />
        ))}
        {rows.length === 0 && <div className="px-3 py-3 text-xs text-text-faint">No spans.</div>}
      </div>
    </div>
  );
}

function SpanTableRow({
  node,
  startMs,
  selfPct,
  columns,
  selected,
  isRootCause,
  onClick,
}: {
  node: SpanNode;
  startMs: number;
  selfPct: number;
  columns: GridColumn<SortKey>[];
  selected: boolean;
  isRootCause: boolean;
  onClick: (spanId: string) => void;
}) {
  const { span } = node;
  const isError = span.StatusCode === "error";

  return (
    <GridRow
      columns={columns}
      dataSpanId={span.SpanID}
      className={selected ? "bg-accent-dim" : isError ? "bg-error-row" : ""}
      onClick={() => onClick(span.SpanID)}
      cells={{
        service: (
          <span className="flex items-center gap-1.5">
            <span
              className={`h-[7px] w-[7px] flex-shrink-0 rounded-full ${isError ? "bg-error" : "bg-[#6ba3d6]"} ${isRootCause ? "ring-1 ring-accent" : ""}`}
            />
            {span.ServiceName}
          </span>
        ),
        name: span.Name,
        start: <span className="font-plex-mono text-text-muted">+{formatDurationNano(Math.max(startMs, 0) * 1_000_000)}</span>,
        duration: <span className="font-plex-mono">{formatDurationNano(span.duration_nano)}</span>,
        self: <span className="font-plex-mono text-text-muted">{selfPct.toFixed(0)}%</span>,
        cost: (
          <span className="font-plex-mono text-text-muted">
            {span.LLMCost != null ? `$${span.LLMCost.toFixed(4)}` : "—"}
          </span>
        ),
      }}
    />
  );
}
