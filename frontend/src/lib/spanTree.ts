import type { Span } from "../api/types";

export interface SpanNode {
  span: Span;
  children: SpanNode[];
  depth: number;
  selfTimeNano: number;
}

// Self-time = span duration minus each direct child's own duration, clamped
// at 0. Mirrors pkg/rootcause/heuristic.go's selfTime exactly (naive
// subtraction, no overlap merge) so the bar matches the verdict math.
function selfTimeNanoOf(span: Span, children: Span[]): number {
  let self = span.duration_nano;
  for (const child of children) self -= child.duration_nano;
  return Math.max(self, 0);
}

// Roots = ParentSpanID === "" OR the parent isn't in this span list at all
// (an orphaned reference is treated as a root, not dropped or crashed on).
export function buildSpanTree(spans: Span[]): SpanNode[] {
  const bySpanID = new Map<string, Span>();
  for (const span of spans) bySpanID.set(span.SpanID, span);

  const childrenOf = new Map<string, Span[]>();
  const roots: Span[] = [];

  for (const span of spans) {
    const hasParent = span.ParentSpanID !== "" && bySpanID.has(span.ParentSpanID);
    if (!hasParent) {
      roots.push(span);
      continue;
    }
    const siblings = childrenOf.get(span.ParentSpanID) ?? [];
    siblings.push(span);
    childrenOf.set(span.ParentSpanID, siblings);
  }

  function build(span: Span, depth: number): SpanNode {
    const directChildren = childrenOf.get(span.SpanID) ?? [];
    const children = directChildren
      .sort((a, b) => a.StartTime.localeCompare(b.StartTime))
      .map((child) => build(child, depth + 1));
    return { span, children, depth, selfTimeNano: selfTimeNanoOf(span, directChildren) };
  }

  return roots
    .sort((a, b) => a.StartTime.localeCompare(b.StartTime))
    .map((span) => build(span, 0));
}

// Depth-first flatten, shared by the tree sidebar and timeline columns so
// row N in one is always row N in the other, no separate scroll sync.
export function flattenSpanTree(nodes: SpanNode[]): SpanNode[] {
  const out: SpanNode[] = [];
  function visit(node: SpanNode) {
    out.push(node);
    for (const child of node.children) visit(child);
  }
  for (const node of nodes) visit(node);
  return out;
}
