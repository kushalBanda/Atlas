# Vertical Slices: Atlas Backend

Build order. Each slice ends runnable and testable.

1. **Tracer bullet** — `pkg/storage` (DuckDB, `spans`+`traces` tables) + `pkg/ingest` (OTLP HTTP receive, no plugin dispatch yet, writes straight to storage) + `pkg/api`/`pkg/query` minimal (`GET /traces/{trace_id}`, `GET /healthz`). No root-cause, no discovery, no plugin registry. Send one OTLP span via curl, GET it back.
2. **Plugin registry** — `pkg/plugin` (Module interface, Registry), `otelcore` module wraps the tracer-bullet ingest/query path. Ingest now dispatches through the registry instead of calling storage directly. Prove `otelcore` and a second dummy test module don't collide.
3. **Root-cause scoring** — `pkg/rootcause` (heuristic + watcher). Trace-close loop uses two triggers: root-span arrival (primary) and `TraceCloseTimeout` = 30s idle (fallback, crashed-hop only). Scores and writes verdict onto `traces`. `GET /traces/{trace_id}` now returns the verdict too. Prove with a 2-span error trace via curl, and prove a synthetic long-running child span does NOT trigger an early close.
4. `llmagent` **module** — second module through `pkg/plugin`, proves the interface holds for a different schema (prompt/tool-call attributes on spans). No interface changes allowed; a change here means Gate 3 was wrong and needs backtracking.
5. **Discovery: process-scan** — `pkg/discovery` interface + `processscan.go` + `rules.go` loading `conf/discovery-rules.yaml`. Reports matched/unrecognized targets via a `GET /discovery/targets` endpoint (log-only is the mock; this is the first real discoverer).
6. **Discovery: Docker** — `docker.go`, same `Discoverer` interface, merges into `RunAll` alongside process-scan.
7. **Discovery: K8s** — `k8s.go`, attaches pod/node/namespace resource attributes to `Target`; confirms these attributes flow onto spans ingested from K8s-discovered sources.
8. **Config + polish** — `pkg/config` full loader (all fields, defaults, validation), `conf/example.yaml` filled in, error-handling edge cases from the HLD (trace-never-closes timeout path, unrecognized-service surfacing) covered by tests. Integration test (`tests/TestEndToEnd_MultiHopTrace...`) closes the loop. Load-test note: confirm ingest-write-batching vs. root-cause-poll DuckDB contention is a non-issue at target scale (see Gate 3 "Write contention note").

Slice 1 proves the skeleton runs end to end with fake-simple data. Slices 2-4 build out the plugin/data-model core (north star #1). Slices 5-7 build out auto-discovery. Slice 8 is the hardening pass before calling v1 backend done.