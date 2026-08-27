# Future: Atlas v1 (backend)

Items scoped out of the current gate docs — noted now so they aren't lost, revisited once the base structure (ingest, storage, plugin registry, query API) is built and running.

## 1. Trace view flexibility

Waterfall is the v1 rendering choice (matches the ecosystem default — Jaeger, SigNoz, HyperDX, Phoenix all lead with it). But the underlying data — a span tree with `trace_id`/`span_id`/`parent_span_id` — doesn't lock us into it.

Keep the query API (`GET /traces/{trace_id}`) returning the raw span tree, not a waterfall-shaped payload, so other views can be added later without a data-model change:

- **Flame graph** — same spans, stacked instead of indented. Better for spotting which span type dominates across many traces.
- **Gantt chart** — same shape as waterfall, more scheduling-style detail.
- **Service map / DAG** — aggregated across many traces, not one request. "What talks to what," not "what happened here."
- **Span list/table** — flat, sortable, loses the tree shape but scans fast across many traces.

No decision needed now. Just a constraint on the query API: return structure, not a pre-rendered view.

## 2. Root-cause scorer — deep design deferred

The v1 heuristic (self-time threshold + earliest-error, scored at trace-close, written once as a verdict) is locked in `03-program-design.md` as the starting point, not the final word. Once ingest/storage/plugin/query are running and Atlas has real trace data flowing, revisit:

- Whether the 30% self-time threshold default holds up against real traces (explicitly unvalidated per the HLD).
- Whether trace-only signal scope (no metrics/logs correlation) is still the right v1 cut once there's something to look at.
- Whether the rule-based heuristic needs refinement before the ML-based future-scope work even becomes relevant.

Don't touch `pkg/rootcause`'s existing spec until there's a running base to validate against — this is a "come back with data" deferral, not an open design question to resolve on paper.
