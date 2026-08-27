# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Go development

Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first it routes to whichever other Go skills the task needs.

## Required Go skills

The following Go skills from `samber/cc-skills-golang` MUST always be applied when working on this project. Load them at the start of every Go-related task, regardless of whether the user explicitly mentions them.

- `samber/cc-skills-golang@golang-error-handling`
- `samber/cc-skills-golang@golang-testing`
- `samber/cc-skills-golang@golang-security`
- `samber/cc-skills-golang@golang-database`
- `samber/cc-skills-golang@golang-observability`

## Project state

Atlas is a greenfield observability platform. The backend has a folder layout but almost no code yet. `cmd/atlas-server/main.go` is a println stub. `pkg/api`, `pkg/config`, `pkg/ingest`, `pkg/query`, `pkg/storage` hold only `.gitkeep` files. `pkg/plugin`, `pkg/discovery`, and `pkg/rootcause` do not exist yet but are part of the planned layout. Check `docs/plans/atlas-v1/00-status.md` for the current gate status before you plan new work.

## Commands

```
make build   # go build ./cmd/... ./pkg/...
make run     # go run ./cmd/atlas-server
make test    # go test ./cmd/... ./pkg/...
```

Run a single test with the normal Go flag, for example:
```
go test ./pkg/storage/... -run TestName
```

## Design source of truth

The real architecture lives in planning docs, not code. Read these before you add or change backend code:

- `docs/plans/atlas-v1/00-status.md` — current gate status and locked decisions. Read this first.
- `docs/plans/atlas-v1/02-architecture.md` — package layout, endpoints, data model, request flow.
- `docs/plans/atlas-v1/01-product.md`, `docs/plans/atlas-v1/03-program-design.md` — product and program design gates.
- `docs/superpowers/specs/2026-08-25-hld-system-design.md` — high-level design. This wins on conflict with other docs.
- `docs/superpowers/specs/2026-08-24-folder-structure-design.md` — folder layout rationale.
- `docs/superpowers/specs/2026-08-24-language-choice-design.md` — stack choice (Go + TS + OTel).

## Architecture (planned, from the HLD)

Atlas ingests OpenTelemetry traces, stores them in embedded DuckDB, and finds the likely root-cause span in a trace at trace-close time.

Packages, by role:
- `pkg/storage` — DuckDB implementation behind a storage interface. ClickHouse stays behind the same interface for later, not built now.
- `pkg/ingest` — OTLP receive (HTTP/gRPC), shared by all plugin modules.
- `pkg/plugin` — module registry interface. `otelcore` and `llmagent` modules register through it.
- `pkg/discovery` — auto-discovery of services to monitor. Build order: process-scan, then Docker, then K8s. Runs on a timer, separate from the request path; it only configures what the OTel Collector should send.
- `pkg/rootcause` — ingest-time trace-tree scoring. A trace-close watcher detects no new spans for a configured window, marks the trace closed, then runs a rule-based heuristic (self-time threshold, default 30%, unvalidated) and writes the verdict onto the trace row.
- `pkg/query` — query API: logs/metrics/traces plus root-cause verdict.
- `pkg/api` — HTTP routing; wires each module's routes at startup.
- `pkg/config` — config loader (discovery rules, storage path, thresholds).
- `cmd/atlas-server` — entrypoint: load config, init storage, register modules, start ingest + API servers.

Data model (DuckDB):
- `spans` — one row per span: `trace_id, span_id, parent_span_id, service_name, name, start_time, end_time, status, attributes (JSON), resource_attributes (JSON)`.
- `traces` — one row per trace, written/updated at trace-close: `trace_id, first_seen, last_seen, closed_at, likely_root_cause_span_id, reason, self_time_pct`. The root-cause verdict is a write-back onto this table, not a separate one.
- `logs`, `metrics` — minimal OTel-standard tables for non-trace signals, not central to root-cause work.

External dependencies: the OTel Collector runs as a reused upstream binary/image, configured (not vendored) via `deploy/otel-collector/`. Atlas's ingest endpoint only receives already-collected OTLP from it. The Docker discoverer talks to the local Docker socket; the K8s discoverer talks to the Kubernetes API (in-cluster or kubeconfig). Single-user, no auth, no other third-party services.

Scope note: v1 is backend only. No UI screens are planned yet; the `frontend/` folder is a placeholder for future work.
