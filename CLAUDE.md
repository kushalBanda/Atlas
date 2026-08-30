# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first. It routes to the other Go skills the task needs.

## Required Go skills

Load these skills at the start of every Go-related task, even if the user does not mention them:

- `samber/cc-skills-golang@golang-error-handling`
- `samber/cc-skills-golang@golang-testing`
- `samber/cc-skills-golang@golang-security`
- `samber/cc-skills-golang@golang-database`
- `samber/cc-skills-golang@golang-observability`

## Project state

Atlas is an observability platform. The backend has a working v1: all 8 planned slices are done (see `docs/plans/atlas/00-status.md`). 110+ tests pass across 12 packages, clean under `go vet` and `go test -race`.

Implemented packages:
- `pkg/storage` — DuckDB-backed `Store` interface (`duckdb.go`, `schema.go`). `schema.go` also carries `migrationDDL`, applied at open time — `CREATE TABLE IF NOT EXISTS` cannot add a column to an existing database file, so column additions (e.g. the agent-run fields) need an explicit `ALTER TABLE ADD COLUMN IF NOT EXISTS` run unconditionally on every startup.
- `pkg/ingest` — OTLP/HTTP receive (JSON and protobuf).
- `pkg/plugin` — module registry (`Registry`, `Module` interface); `otelcore` and `llmagent` modules registered.
- `pkg/fields` — modular per-span field extractors (LLM fields, agent-run fields; `kind.go`/`llm.go`/`agent.go`), run unconditionally in `Registry.Dispatch` on every span regardless of `atlas.module`. `Span` also carries `SessionID`/`UserID`/`AgentRunID`/`AgentName`/`AgentStepKind`, populated from `session.id`/`user.id`/`agent.run.id`/`agent.name`/`agent.step.kind` (or their `gen_ai.*` equivalents) on any span.
- `pkg/discovery` — process-scan, Docker, and K8s discoverers behind one `Discoverer` interface, plus `Handler` for `GET /discovery/targets`.
- `pkg/rootcause` — trace-close `Watcher` and `Score` heuristic.
- `pkg/agentrun` — derives the agent-run decision graph (`Build`) at read time from spans sharing an `agent_run_id`; a run may cross traces. Nothing here is stored — see `docs/superpowers/specs/2026-08-29-agent-run-debugging-design.md`.
- `pkg/query`, `pkg/api`, `pkg/config` — query API (`GET /traces`, `GET /traces/{trace_id}`, `GET /stats`, `GET /runs`, `GET /runs/{run_id}`, `GET /sessions`, `GET /sessions/{session_id}`), HTTP router, YAML config loader.
- `cmd/atlas-server` — entrypoint, plus `supervise.go` (recover+backoff wrapper for the watcher goroutine).

`scripts/ai-gateway/` — a standalone Python FastAPI reference client (not part of the Go module) that instruments an OpenRouter-backed LLM call and agent/tool-call flow with OTel, exporting to Atlas's ingest endpoint. Backs the `.claude/skills/atlas-instrument/SKILL.md` skill, which documents Atlas's ingest/attribute contract for instrumenting any external service.

`frontend/` has a complete v1 (all 3 screens, wired to the real backend). See `frontend/CLAUDE.md` for frontend-specific commands and architecture.

Known gaps (intentional, not bugs): logs/metrics query endpoints, kubeconfig-based K8s discovery, discovery-to-collector auto-wiring, a real load test. See `docs/design-considerations.md`.

## Commands

```
make build   # go build ./cmd/... ./pkg/...
make run     # go run ./cmd/atlas-server
make test    # go test ./cmd/... ./pkg/...
```

Run a single test:
```
go test ./pkg/storage/... -run TestName
```

Race-check and vet before calling work done:
```
go test -race ./cmd/... ./pkg/...
go vet ./...
```

Server takes a `-config` flag (default `./conf/example.yaml`); it falls back to `config.Default()` with a warning if the file is absent, but errors hard if the file exists and fails to parse or validate.

## Design source of truth

The real architecture lives in planning docs, not just code. Read these before you add or change backend code:

- `docs/plans/atlas/00-status.md` — current gate/slice status, locked decisions, and backtrack history. Read this first.
- `docs/plans/atlas/02-architecture.md` — package layout, endpoints, data model, request flow.
- `docs/plans/atlas/01-product.md`, `docs/plans/atlas/03-program-design.md` — product and program design gates.
- `docs/superpowers/specs/2026-08-25-hld-system-design.md` — high-level design. This wins on conflict with other docs.
- `docs/superpowers/specs/2026-08-24-folder-structure-design.md` — folder layout rationale.
- `docs/superpowers/specs/2026-08-24-language-choice-design.md` — stack choice (Go + TS + OTel).
- `docs/design-considerations.md` — documented deviations from the plan docs and known gaps, with reasoning.
- `CONTEXT.md` — domain glossary (root span, trace close, verdict, self-time).

## Architecture

Atlas ingests OpenTelemetry traces, stores them in embedded DuckDB, and finds the likely root-cause span in a trace at trace-close time.

Request flow: OTLP spans arrive at ingest (`POST /v1/traces`, the fixed OTLP/HTTP path — not Atlas's own API, so it keeps its version prefix even though the rest of Atlas's API dropped `/v1/`). Ingest hands each batch to `plugin.Registry.Dispatch`, which first runs `pkg/fields.Apply` over every span (typed-field extraction, unconditional), then groups spans by their `atlas.module` resource attribute (default `otelcore`) and calls the matching `Module.HandleSpans`. Each module writes through the shared `storage.Store` interface. A trace-close `Watcher` polls after every write batch and on an idle-timeout fallback, closes eligible traces, and writes the root-cause verdict back onto the `traces` row via `MarkTraceClosed`.

Trace-close triggers, in order:
1. **Primary**: the root span (`parent_span_id IS NULL`) has arrived — `ListRootArrivedTraces`.
2. **Fallback**: no new span for the configured idle window (default 30s) and no root span ever arrived (crashed-hop case) — `ListStaleOpenTraces`.

Root-cause heuristic (`pkg/rootcause`, unvalidated, default self-time threshold 30%): earliest error span wins; otherwise the span with the highest self-time as a percent of trace duration, if it clears the threshold.

Plugin module contract (`pkg/plugin/plugin.go`): a `Module` declares its own storage tables (`RegisterSchema`), its own HTTP routes (`RegisterRoutes`), and a span handler (`HandleSpans`). `otelcore` and `llmagent` are structurally identical — both write through the same `storage.Store` — and are distinguished only by `Name()` and by which spans route to them via `atlas.module`. Registering a new module needs no changes to the `Module` interface itself. Field extraction (below) is intentionally *not* part of this per-module split — it runs once in `Registry.Dispatch`, attribute-driven, before spans reach any module's `HandleSpans`.

Storage (`pkg/storage/storage.go`): `Store` is backend-agnostic. DuckDB is the only implementation now; ClickHouse can implement the same interface later without touching call sites. `Span.Attributes`/`ResourceAttributes` are stored as `VARCHAR` JSON text, not DuckDB's native `JSON` type — the driver scans native `JSON` columns back as `map[string]interface{}` instead of a string, which breaks `Scan` into `*string`. `Span` also carries typed fields (`SpanKind`, `Level`, and the `LLM*` fields — model, tokens, cost, temperature/top_p/max_tokens, usage/cost detail maps, time-to-first-token, prompt id/name/version) populated by `pkg/fields` extractors from self-describing attributes (`gen_ai.*`, `openinference.span.kind`, `level`) on *any* span, not gated by `atlas.module`. Prompt/completion text is deliberately not duplicated into a typed column — read it from `Attributes` (`gen_ai.prompt`/`gen_ai.completion`) instead; only the fields worth aggregating in SQL got a typed column.

Discovery (`pkg/discovery`): one `Discoverer` interface, three implementations layered in build order — process-scan (`lsof`-backed), Docker (talks to the local Docker socket), K8s (in-cluster auth only; a no-op outside a cluster, not an error). `RunAll` merges results across discoverers and isolates a single discoverer's failure from the rest. Discovery runs on its own timer, separate from the request path, and only reports targets (`GET /discovery/targets`) — it does not auto-wire the OTel Collector.

External dependencies: the OTel Collector runs as a reused upstream binary/image, configured (not vendored) via `deploy/otel-collector/`. Atlas's ingest endpoint only receives already-collected OTLP from it. Single-user, no auth, no other third-party services.
