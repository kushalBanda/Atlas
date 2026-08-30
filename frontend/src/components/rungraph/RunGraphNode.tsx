import { Repeat2 } from "lucide-react";
import type { LaidOutNode } from "../../lib/runGraph";
import { StepKindBadge } from "./StepKindBadge";

export interface RunGraphNodeProps {
  node: LaidOutNode;
  repeatCount?: number;
  selected: boolean;
  onSelect: (spanId: string) => void;
}

export function RunGraphNode({ node, repeatCount, selected, onSelect }: RunGraphNodeProps) {
  const isError = node.status_code === "error";

  return (
    <button
      type="button"
      onClick={() => onSelect(node.span_id)}
      className={`group h-[84px] w-48 shrink-0 overflow-hidden rounded-md border bg-surface px-3 py-2.5 text-left text-xs transition-colors motion-reduce:transition-none
        ${isError ? "border-[#f3a2a5]/50" : "border-border"}
        ${selected ? "ring-1 ring-accent" : "hover:border-accent/60 hover:bg-elevated"}`}
    >
      <div className="flex items-center justify-between gap-2">
        <StepKindBadge kind={node.step_kind} />
        {repeatCount ? (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-sm bg-elevated px-1.5 py-0.5 text-[10px] font-semibold text-accent">
            <Repeat2 className="h-3 w-3" strokeWidth={2.5} />
            {repeatCount}×
          </span>
        ) : null}
      </div>
      <div className="mt-1.5 truncate text-[13px] font-medium text-text-primary" title={node.name}>
        {node.name}
      </div>
      <div className="mt-1 flex items-center gap-1.5 truncate font-plex-mono text-[10px] text-text-faint">
        {node.agent_name ? <span className="truncate">{node.agent_name}</span> : null}
        {node.agent_name ? <span>·</span> : null}
        <span className={isError ? "text-[#f3a2a5]" : ""}>
          {(node.duration_nano / 1_000_000).toFixed(1)} ms
        </span>
      </div>
    </button>
  );
}
