import { Bot, CircleDot, Link2, Search, Sparkles, Wrench } from "lucide-react";

// Icon + color per step kind, the way Langfuse's ItemBadge maps
// ObservationType to a lucide icon and a hue (docs/resources/langfuse
// web/src/components/ItemBadge.tsx). Colors are chosen to sit next to
// Atlas's own orange accent without competing with it.
const KIND_STYLE: Record<string, { icon: typeof Bot; className: string; label: string }> = {
  AGENT: { icon: Bot, className: "text-violet-400", label: "Agent" },
  agent: { icon: Bot, className: "text-violet-400", label: "Agent" },
  LLM: { icon: Sparkles, className: "text-fuchsia-400", label: "LLM" },
  llm: { icon: Sparkles, className: "text-fuchsia-400", label: "LLM" },
  TOOL: { icon: Wrench, className: "text-sky-400", label: "Tool" },
  tool: { icon: Wrench, className: "text-sky-400", label: "Tool" },
  CHAIN: { icon: Link2, className: "text-emerald-400", label: "Chain" },
  chain: { icon: Link2, className: "text-emerald-400", label: "Chain" },
  RETRIEVER: { icon: Search, className: "text-teal-400", label: "Retriever" },
  retriever: { icon: Search, className: "text-teal-400", label: "Retriever" },
};

export function StepKindBadge({ kind }: { kind: string }) {
  const style = KIND_STYLE[kind] ?? { icon: CircleDot, className: "text-text-faint", label: kind || "Step" };
  const Icon = style.icon;
  return (
    <span className={`inline-flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide ${style.className}`}>
      <Icon className="h-3 w-3" strokeWidth={2.25} />
      {style.label}
    </span>
  );
}
