# Atlas: High-Level Design and System Design

Date: 2026-08-25
Status: Approved
Supersedes: none (extends `2026-08-24-language-choice-design.md` and `2026-08-24-folder-structure-design.md`)
Related: `docs/notes/atlas-technical-differentiation-candidates.md`, `docs/notes/observability-platform-alternatives.md`

## Context

Follow-up to the folder-structure spec. That spec fixed the layout assuming ClickHouse + OTel Collector, general observability only, multi-user-shaped. This spec revises the product direction after a long brainstorming session and locks the system design that follows from it. Where this spec conflicts with the earlier two, this spec wins.

Reference repos used throughout: `docs/resources/{signoz,hyperdx,openobserve,netdata,langfuse,phoenix}`.

## Product framing

Atlas is a single-user, developer-first observability platform. It is a learning project with real product ambition — not a lightweight clone of SigNoz or Langfuse. Convenience features (single binary, auto-discovery) are the on-ramp, not the product's reason to exist. The reason to exist is the differentiation section below.

## Scope

One binary covers both:
- **General observability core** — OTel-native logs, metrics, traces.
- **LLM/agent observability** — prompts, tool calls, multi-agent traces — built as a plugin on the same core, not a second tool.

No multi-tenancy, no team/org model, no auth system in v1. Single user, single deployment.

## Storage: DuckDB, embedded

Atlas embeds DuckDB directly in the Go binary. This replaces the earlier ClickHouse-only decision from the folder-structure spec.

- DuckDB is columnar/OLAP, so trace/metric/log aggregate queries stay fast at the target scale (single user, single machine, up to tens of GB).
- It is a library, not a server process — no container, no network hop, no separate lifecycle to manage. This is what makes the binary-first install path genuinely zero-dependency.
- Data lives in one file. One binary + one file is the whole deployable unit.
- ClickHouse is not ruled out permanently: the storage layer sits behind an interface (`pkg/storage`) so a ClickHouse-backed implementation can be swapped in later for users who outgrow single-node DuckDB. It is not built in this pass.

## Distribution

Multiple install paths, all supported:
- `docker-compose up` (familiar, matches the reference tools).
- `brew install atlas` (or equivalent package manager).
- Raw binary download.

The bar for all of them: the **binary-first path must run with zero external dependencies** — no ClickHouse, no Redis, no Postgres. This is a hard requirement, not a nice-to-have, because it is the thing that makes "developer-first" true rather than aspirational.

## Setup: auto-discovery over manual config

Atlas does not build custom collectors. It uses the existing OTel Collector receiver ecosystem — the same choice SigNoz, HyperDX, and OpenObserve already made, confirmed by reading their `deploy/`/`docker/` configs in `docs/resources/`. Writing custom collectors would duplicate work the OTel community has already done and contradicts the "reuse, don't rebuild" principle carried over from the language-choice spec.

What Atlas adds is the setup experience. Manually running `atlas collect <tool>` per service, once for every service in a stack, is redundant and does not scale to a real multi-service environment — this was flagged directly in review and the design changed as a result. The primary flow is **auto-discovery**, modeled on netdata's service-discovery architecture (`docs/resources/netdata/src/go/plugin/go.d/discovery/sdext/discoverer/`):

1. **Process/port scan** (build first) — read the host's listening-socket table, match `(port, process name)` against a curated rule set (e.g. port 5432 + `postgres` → Postgres receiver config), same approach as netdata's `net_listeners` discoverer. Works with no Docker and no Kubernetes; the widest-reaching, lowest-dependency discoverer, so it goes first.
2. **Docker discovery** (build second) — query the Docker API, match container image name to a receiver, one target per exposed port, mirroring netdata's `dockersd`.
3. **Kubernetes discovery** (build third) — query the K8s API for pods/containers, mirroring netdata's `k8ssd`. This stage is also what attaches pod/node/namespace resource attributes to spans, which the root-cause and pod-level trace-flow features depend on.

Each discoverer is a separate implementation behind one interface, added in this order without blocking the others. `atlas collect <tool>` remains available as a manual override for the case discovery misses, but is not the primary path.

## Plugin architecture

An in-process module registry, adapted from netdata's `go.d.plugin` model but running in-process rather than as a separate OS process (netdata's plugins run out-of-process over a pipe; Atlas's single-binary goal rules that out).

- A `pkg/plugin` interface exposes what a module needs: register its ingest schema, register its query/API routes, and read from the shared storage layer.
- Each module self-registers via its own `init()`, following the collector self-registration pattern seen in `docs/resources/netdata/src/go/plugin/go.d/collector/*`.
- The OTel core (`pkg/ingest`, `pkg/query` from the folder-structure spec) is implemented **as the first module through this same interface**, not special-cased. This is deliberate: if the core itself can't fit through the plugin seam, the seam is wrong.
- The LLM/agent module is the second module, proving the interface holds for a genuinely different schema (prompts, tool calls, agent-to-agent spans) without changes to the interface itself.

## Data model: the real differentiation (north star #1 + #5)

From `docs/notes/atlas-technical-differentiation-candidates.md`: no reference tool has a schema spanning both infra observability and LLM/agent observability. SigNoz/HyperDX/OpenObserve schemas are infra-only; Phoenix/Langfuse schemas are LLM-only and model single request/response trees, not multi-agent or tool-call chains.

Atlas's span model is unified: every signal — an HTTP call, a DB query, an LLM completion, a tool invocation, an agent handoff — is a span in the same trace tree, sharing the same `trace_id`/`span_id`/parent-child structure (W3C Trace Context propagation, the OTel standard — no reason to deviate from it). LLM/agent-specific fields (prompt, tool name, token count, cost) are additional attributes on a span, not a separate schema or a separate store. This is what lets a single trace show a gateway hop, a pod-to-pod call, and an agent's tool call in one waterfall.

K8s discovery (above) attaches pod/node/namespace as resource attributes on these same spans, so a full request flow — gateway → pod → pod → DB, or a coding agent's tool-call chain — renders as one trace regardless of whether the hops are infra or LLM calls.

## Root-cause correlation: the flagship differentiator (north star #2)

Distributed tracing across gateway/pods/services is table stakes — any OTel-based tool already does this once spans propagate `trace_id` correctly. The open problem, and the one worth owning, is **automatically pinpointing which span in that trace is the actual root cause**, instead of leaving a human to eyeball a waterfall. This is built in full now, not deferred to a later phase — scoped as follows:

**Signal scope (v1): trace-only.** Root-cause scoring reasons over span structure alone — duration, self-time (time not spent in child spans), and error status. Metrics and logs are not correlated in for this pass. Cross-signal correlation (folding in metrics/log anomalies) is real future value, but pulling it into v1 means solving two hard problems — a unified schema and cross-signal correlation — before either is proven standalone. Trace-only is enough to answer "which span actually broke this request," and is the harder-but-tractable slice.

**Algorithm (v1): rule-based/threshold heuristic, not ML.**
- Walk the trace tree once it closes.
- Flag any span with error status; the earliest error (by start time) is a primary candidate.
- Compute self-time for every span (span duration minus sum of child span durations); rank by self-time as a percentage of total trace duration.
- A span crossing a self-time threshold (default: >30% of total trace duration) is a secondary candidate.
- Surface the top-ranked candidate as `likely_root_cause_span_id`, with a human-readable `reason` string ("first error in trace" or "67% of total trace time spent in this span, no child calls").

This is deterministic and explainable — the UI can show *why* a span was flagged, not just that it was. It is buildable by a small team without a training corpus, which a statistical/ML approach would require.

**Future scope (explicitly out of v1, not designed further here): a trained ML model.** Once Atlas has accumulated enough real trace data, a learned model (baseline-aware — "this span is slow compared to its own history," anomaly scoring across historical spans of the same operation) can replace or augment the heuristic. This needs a data corpus Atlas won't have on day one, so it is roadmap, not v1.

**Compute timing: ingest-time, not query-time.** Root-cause scoring runs once, immediately after a trace closes (all its spans have arrived), and the result is stored as attributes on the trace row in DuckDB (`likely_root_cause_span_id`, `reason`, `self_time_pct`). Rationale: DuckDB is built for analytical reads, not per-request recompute; traces are immutable once closed, so recomputing the same heuristic every time a trace is viewed wastes work for no benefit. Storing the verdict also means trace-list views can filter/sort by "has a flagged root cause" without recomputing anything.

## Component overview

```
atlas-server (single Go binary)
├── pkg/plugin/           # module registry interface; all modules below implement it
│   ├── otelcore/         # first module: OTel ingest + query (logs/metrics/traces)
│   └── llmagent/         # second module: LLM/agent trace schema (prompts, tool calls)
├── pkg/discovery/        # process-scan, docker, k8s discoverers (same order as build plan)
├── pkg/rootcause/        # ingest-time trace-tree scoring (rule-based v1)
├── pkg/storage/          # DuckDB implementation behind a swappable interface
├── pkg/ingest/           # OTLP receive, shared across modules
├── pkg/query/            # HTTP/gRPC query API
└── pkg/api/              # routing, wires modules' routes in at startup
```

Data flow for a request through gateway → pod A → pod B:
1. Each hop's OTel SDK (or gateway's built-in OTel support) propagates `trace_id` via W3C Trace Context headers.
2. Each hop emits spans to Atlas's OTLP ingest endpoint (`pkg/ingest`), tagged with K8s resource attributes if the K8s discoverer is active.
3. `pkg/storage` writes spans to DuckDB as they arrive.
4. On trace close (no new spans for a configured window), `pkg/rootcause` runs the heuristic scoring pass and writes the verdict back onto the trace row.
5. `pkg/query` serves the trace waterfall plus the root-cause verdict to the UI in one call.

## Error handling and edge cases

- **Trace never closes** (a hop crashes mid-request, no final span arrives): a timeout-based close (configurable window, default a few seconds after last-seen span) forces scoring on whatever spans arrived, so a broken request still gets a verdict rather than hanging forever unscored.
- **Discovery finds an unknown port/process**: no match in the curated rule set means no collector job is started for it; it is surfaced as "unrecognized service" rather than silently ignored, so the user knows discovery saw something it didn't understand.
- **Self-time threshold false positives** (a legitimately slow but correct span, e.g. a batch job): the heuristic flags it as a *candidate*, with its reasoning shown — it is a suggestion, not an alarm, and the ML-based future scope is the intended fix for reducing false positives over time.

## Testing

- `pkg/rootcause`: unit tests over synthetic trace trees (linear chain, fan-out, error-in-child, no-error) verifying the heuristic picks the expected span and threshold math is correct.
- `pkg/discovery`: unit tests per discoverer against fixture data (fake `/proc`-style listener tables, fake Docker API responses) — no live Docker/K8s required for the test suite.
- `pkg/plugin`: a test module implementing the interface, verifying the registry wires ingest/query routes correctly, and that `otelcore` and `llmagent` don't collide on route or schema registration.
- Integration/e2e (top-level `tests/`, per the folder-structure spec): a multi-hop synthetic trace sent through real ingest, confirming the full pipeline (ingest → storage → root-cause scoring → query) produces the expected verdict end-to-end.

## What changes from the earlier two specs

- Storage: ClickHouse-primary → DuckDB-primary, ClickHouse as a future swappable backend.
- Setup: `deploy/otel-collector` config as the only path → auto-discovery (process/Docker/K8s) as primary, manual config as override.
- Scope: general-observability-only → general + LLM/agent, one plugin-based binary.
- The folder-structure spec's `pkg/<domain>` layout still holds; `pkg/plugin`, `pkg/discovery`, `pkg/rootcause` are new domains added under it, `otelcore` and `llmagent` live under `pkg/plugin/`.

## NOT in scope (this pass)

- Cross-signal (metrics/logs) correlation for root cause — trace-only for v1, noted as future.
- Trained ML model for root-cause scoring — explicitly future scope.
- Multi-user, auth, teams/orgs.
- ClickHouse backend implementation (interface only, no second implementation built now).
- CI/CD, distribution/build pipeline (still deferred per the folder-structure spec).

## Consequences

- All new backend code implements the plugin interface where it registers ingest or query behavior; nothing bypasses `pkg/plugin` to wire routes directly.
- Every span, regardless of source module, uses the same trace/span ID model and resource-attribute conventions — the LLM/agent module must not introduce a parallel schema.
- Root-cause scoring is a mandatory ingest-time step for every closed trace, not an opt-in feature — it's core to the product's identity, not an add-on.
