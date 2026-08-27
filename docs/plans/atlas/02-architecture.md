# Architecture: Atlas v1 (backend)

## Fit

Greenfield. Only `cmd/atlas-server/main.go` exists (println stub). All packages below are new, built inside the folder layout already fixed by `2026-08-24-folder-structure-design.md`.

- `pkg/storage` — DuckDB implementation behind a storage interface.
- `pkg/ingest` — OTLP receive (HTTP/gRPC), shared by all plugin modules.
- `pkg/plugin` — module registry interface. `otelcore` and `llmagent` register through it.
- `pkg/discovery` — process-scan, Docker, K8s discoverers, in that build order.
- `pkg/rootcause` — ingest-time trace-tree scoring.
- `pkg/query` — query API (logs/metrics/traces + root-cause verdict).
- `pkg/api` — HTTP routing; wires each module's routes at startup.
- `pkg/config` — config loader (discovery rules, storage path, thresholds).
- `cmd/atlas-server` — entrypoint: load config, init storage, register modules, start ingest + API servers.



## Endpoints

- `POST /v1/traces` (OTLP/HTTP) : Span ingest, standard OTLP trace export format. Keeps the `/v1/` prefix here — this is the OTLP/HTTP protocol's fixed default export path (not Atlas's own API versioning), used as-is by every standard OTel SDK/Collector exporter without extra config (confirmed against signoz/hyperdx collector configs, which don't override it).
- `GET /traces/{trace_id}` : Trace waterfall + root-cause verdict, one call.
- `GET /traces?filter=...` : Trace list, filterable by has-root-cause-verdict, time range, service.
- `GET /logs`, `GET /metrics` : Basic query surface for the other two OTel signals (otelcore module), not the focus of v1 but required so the ingest path isn't trace-only.
- `GET /healthz` : Liveness, needed by all three distribution paths (compose, brew, binary).
- `GET /discovery/targets` : Matched + unrecognized targets found by `pkg/discovery` (process-scan/Docker/K8s), v1 reports only, no auto-wiring into the OTel Collector (see Gate 3 "Least confident decisions" #3).



## Data

DuckDB, one file, opened by `pkg/storage`. Core tables:

- `spans` :  One row per span: `trace_id, span_id, parent_span_id, service_name, name, start_time, end_time, status, attributes (JSON), resource_attributes (JSON)`. Query: `SELECT * FROM spans WHERE trace_id = ? ORDER BY start_time` — returns the raw span tree, not a pre-shaped waterfall payload, so other trace views (flame graph, service map, span list) can render off the same query later. See `future.md`.
- `traces` : One row per trace, written/updated at trace-close: `trace_id, first_seen, last_seen, closed_at, likely_root_cause_span_id, reason, self_time_pct`. Query: trace list = `SELECT * FROM traces WHERE closed_at IS NOT NULL ORDER BY closed_at DESC LIMIT ?`.
- `logs`, `metrics` : Minimal OTel-standard tables for the otelcore module's non-trace signals, not elaborated further here (not on the critical path for root-cause).

Root-cause verdict is a write-back onto `traces`, not a separate table, Gate 3 will confirm this against the storage interface's update semantics.

## Flow

Main path (a request through gateway → pod A → pod B, or an agent's tool-call chain):

1. Each hop's OTel SDK propagates `trace_id` via W3C Trace Context and sends spans to Atlas.
2. `pkg/ingest` receives OTLP, hands spans to the registered module (`otelcore` for infra spans, `llmagent` for agent spans) via `pkg/plugin`.
3. Module writes spans to `pkg/storage` (DuckDB `spans` table).
4. **Trace close, primary trigger: root-span arrival.** A span only reports `end_time` after its own work — including any children it waited on completes, so once the trace's root span (`parent_span_id IS NULL`) is written, every other span in that trace tree must already be in. `pkg/rootcause` closes the trace the moment the root span lands, not on a timer. This is what makes a long-running child (e.g. an LLM call spanning tens of seconds) safe: the trace doesn't close until the root span, which itself can't end before that child — arrives. Validated against reference tools: Phoenix models root-span identity (`parent_id is None`) as a first-class schema concept, not a timing signal (`docs/resources/phoenix/src/phoenix/trace/schemas.py:141-142`); Langfuse never waits on a trace-level idle window at all, it processes per-span with retry/backoff instead (`docs/resources/langfuse/worker/src/features/evaluation/retryObservationNotFound.ts:17-23`).
5. **Trace close, fallback trigger: idle timeout.** `TraceCloseTimeout` (default 30s) only fires when a trace's root span never arrives at all, a hop crashed mid-request, so there's no root span to trigger step 4. This is the HLD's "trace never closes" case: force-close on whatever spans arrived, so a broken request still gets a verdict.
6. `pkg/rootcause` reads the closed trace's spans, runs the heuristic, writes the verdict onto `traces`.
7. `pkg/query` serves `GET /traces/{trace_id}` one query joining `spans` + `traces`, returns the span tree + verdict. Rendered as a waterfall in v1 (client-side); the heuristic itself is a v1 starting point, not final — deeper root-cause design deferred, see `future.md`.

Discovery is a separate, parallel flow: `pkg/discovery` runs its scanners on a timer, matches found services against curated rules, and starts/updates OTel Collector receiver configs for them. It does not sit in the request path above; it only decides what should be sending Atlas data at all.

## External

- OTel Collector — reused upstream binary/image, configured (not vendored) via `deploy/otel-collector/`. Atlas's own ingest endpoint receives already-collected OTLP from it.
- Docker API (local socket) — used by the Docker discoverer.
- Kubernetes API (in-cluster or kubeconfig) — used by the K8s discoverer.
- No other third-party services. No env vars requiring secrets (single-user, no auth).
- **Bind address default:** `127.0.0.1` for both ingest and query listeners. No-auth is only safe if the listener isn't network-reachable by default; `docker-compose`/K8s deployments override to `0.0.0.0` explicitly in `deploy/`, relying on the container/cluster network boundary instead of app-level auth.

