import type { Span } from "../api/types";
import { buildSpanTree, flattenSpanTree } from "./spanTree";

export interface CallNode {
  key: string; // `${service}::${name}`
  service: string;
  name: string;
  callCount: number;
  errorCount: number;
  totalDurationNano: number;
  depth: number; // shallowest depth this operation was seen at
  spanIds: string[]; // every span instance aggregated into this node
}

export interface CallEdge {
  from: string; // parent node key
  to: string; // child node key
  count: number;
  errorCount: number;
}

function keyOf(span: Span): string {
  return `${span.ServiceName}::${span.Name}`;
}

// Aggregates every span occurrence into one node per (service, operation
// name) pair, so a call made 7 times collapses to one node with callCount:7
// instead of 7 separate boxes. Depth comes from the real span tree
// (buildSpanTree), so an operation called from multiple places is placed at
// its shallowest occurrence — this is what surfaces the "layers under one
// service" a same-service call tree has, which a same-service-only Waterfall
// glance can hide once nodes fan out.
export function buildCallGraph(spans: Span[]): { nodes: CallNode[]; edges: CallEdge[] } {
  const tree = flattenSpanTree(buildSpanTree(spans));
  const bySpanID = new Map<string, Span>();
  for (const s of spans) bySpanID.set(s.SpanID, s);

  const nodeByKey = new Map<string, CallNode>();
  const edgeByKey = new Map<string, CallEdge>();

  for (const { span, depth } of tree) {
    const k = keyOf(span);
    let node = nodeByKey.get(k);
    if (!node) {
      node = {
        key: k,
        service: span.ServiceName,
        name: span.Name,
        callCount: 0,
        errorCount: 0,
        totalDurationNano: 0,
        depth,
        spanIds: [],
      };
      nodeByKey.set(k, node);
    }
    node.callCount += 1;
    node.totalDurationNano += span.duration_nano;
    node.depth = Math.min(node.depth, depth);
    node.spanIds.push(span.SpanID);
    if (span.StatusCode === "error") node.errorCount += 1;

    const parent = span.ParentSpanID ? bySpanID.get(span.ParentSpanID) : undefined;
    if (parent) {
      const edgeKey = `${keyOf(parent)}->${k}`;
      let edge = edgeByKey.get(edgeKey);
      if (!edge) {
        edge = { from: keyOf(parent), to: k, count: 0, errorCount: 0 };
        edgeByKey.set(edgeKey, edge);
      }
      edge.count += 1;
      if (span.StatusCode === "error") edge.errorCount += 1;
    }
  }

  return { nodes: [...nodeByKey.values()], edges: [...edgeByKey.values()] };
}
