# Design Considerations

## OTLP ingest is HTTP-only; gRPC not built

**What's missing:** `02-architecture.md` and `03-program-design.md` both describe
`pkg/ingest` as "OTLP receive (HTTP/gRPC)." Only the HTTP receiver
(`POST /v1/traces`, protobuf or JSON body) exists. gRPC was not attempted,
not rejected for a reason — this is a gap, not a considered trade-off.

**Why it hasn't blocked anything:** every OTel SDK/exporter used so far in
verification defaults to or supports OTLP/HTTP, so nothing built on top of
ingest depends on gRPC existing. `04-slices.md`'s 8-slice plan never calls
for it explicitly either.

**If this becomes needed:** add a gRPC server alongside the HTTP one in
`cmd/atlas-server`, both feeding the same `ingest.Dispatcher`/`plugin.Registry`
path — `pkg/ingest`'s span-conversion logic (`tracesToSpans`) is already
protocol-agnostic (it operates on `ptrace.Traces`, not raw bytes), so this
is additive, not a rewrite.

## `GET /traces?filter=`, `/logs`, `/metrics` endpoints not built

**What's missing:** `02-architecture.md` lists these as required Atlas API
endpoints. Only `GET /traces/{trace_id}` exists.

**Why this isn't a slice gap:** none of `04-slices.md`'s 8 slices call for
these endpoints — the inconsistency is between `02-architecture.md` and
`04-slices.md` themselves (both gate-approved), not a missed implementation
step. The underlying storage-level plumbing already exists and is tested:
`storage.Store.ListTraces` and `storage.TraceFilter` are implemented and
covered by `TestListTraces_FiltersByHasRootCause` — only the HTTP handler
and route are missing, not the query logic behind them.

## DuckDB file lock blocks external DB clients while atlas-server runs

**Observed:** Opening `atlas.duckdb` in a separate DB client (GUI tool, `duckdb` CLI)
while `atlas-server` is running fails:

```
Failed to open DuckDB database at '.../atlas.duckdb': IO Error: Could not set lock
on file "..." : Conflicting lock is held in .../atlas-server (PID ...) by user ...
```

**Why:** DuckDB is embedded, not client-server. There is no daemon accepting
connections (unlike Postgres/ClickHouse). `storage.NewDuckDB` opens one
`*sql.DB` handle at startup (`pkg/storage/duckdb.go`) and holds it for the
server's whole lifetime. DuckDB takes an exclusive process-level lock on the
file for read-write access — a second process opening the same file for
read-write always conflicts. This is the same behavior SQLite has.

Atlas's own HTTP endpoints (`GET /traces/{trace_id}`, `GET /healthz`, ingest)
work fine while the server runs — they go through the same in-process handle,
not a second connection.

**Is this a design flaw? Arguable both ways.**

Not a flaw:
- Embedded DB was chosen specifically to avoid running/operating a separate
  DB server for a single-user, no-auth v1 (see HLD, `02-architecture.md`).
- The lock is DuckDB protecting write consistency, not a bug.
- The intended way to inspect data is Atlas's own query API, not a raw DB
  client — ad-hoc SQL access is a dev convenience, not a first-class path.

Real cost:
- Blocks live raw-SQL inspection while the server runs — an ergonomics tax
  during development/debugging.
- Blocks running two Atlas processes against the same file — but v1 is
  explicitly single-instance, so this is in scope, not a surprise.

**Escape hatches, in order of preference:**
1. Query through Atlas's own API instead of a separate DB client.
2. Stop `atlas-server`, then open the file in a DB client.
3. If the DB client supports DuckDB's read-only attach mode, it may be able
   to attach alongside the running (read-write) server.

**Not planned as a fix:** the storage interface already has an escape hatch
if this becomes painful — ClickHouse is designed to sit behind the same
`storage.Store` interface later, and it doesn't have this single-process
limitation.

## K8s discoverer: in-cluster only, no kubeconfig support in v1

**What's built:** `pkg/discovery/k8s.go`'s `K8sDiscoverer` authenticates using
in-cluster service account credentials (`KUBERNETES_SERVICE_HOST` env var +
the service account token/CA mounted into every pod). This is the standard
way a workload running *inside* a cluster talks to its own API server.

**What's not built:** discovery from a developer machine pointed at a
*remote* cluster via `~/.kube/config` (context selection, client
certificates, exec-based auth plugins, etc). Outside a cluster, `Discover`
returns zero targets and no error — a documented no-op, not a failure.

**Why this is an acceptable v1 gap:** Atlas's discovery subsystem only
*reports* candidate targets (`GET /discovery/targets`); it doesn't auto-wire
the OTel Collector to actually start scraping them (see the "Least
confident decisions" #3 note in `03-program-design.md`). Since discovery's
output isn't yet load-bearing for ingest, a dev-machine convenience path
isn't worth the added complexity (kubeconfig parsing, exec auth plugins,
context switching) until auto-wiring exists and someone actually needs to
develop against a remote cluster.

**If this becomes needed:** the natural fix is `client-go`'s
`clientcmd.BuildConfigFromFlags` for kubeconfig loading, kept behind the
same `podLister` interface `K8sDiscoverer` already uses — no interface
change required, only a second constructor path.

## Write-contention smoke test, not a real load test

**What's built:** `TestConcurrentWritesAndScans_NoErrors` in
`pkg/storage/duckdb_test.go` runs `WriteSpans` (ingest path) and
`ListRootArrivedTraces`/`ListStaleOpenTraces` (root-cause poll path)
concurrently from 16 goroutines, ~25 operations each, against DuckDB's
single (`MaxOpenConns(1)`) connection. It passes clean under `-race`.

**What this proves:** concurrent access from both hot paths doesn't error,
deadlock, or corrupt data at this modest volume.

**What this does NOT prove:** a throughput ceiling, latency under
sustained load, or behavior at the "tens of GB, single-user" scale the HLD
targets. `03-program-design.md`'s "Write contention note" calls for an
actual load test before calling this a settled question — that requires a
real benchmark harness (sustained ingest rate, concurrent poll-loop
pressure, measured p50/p99 write latency) that wasn't built in this pass.
Treat the current smoke test as "doesn't obviously break," not "verified
at scale."
