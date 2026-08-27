# Program Design: Atlas Backend

## Files

```
pkg/storage/
  storage.go          # Storage interface (backend-agnostic)
  duckdb.go            # DuckDB implementation
  schema.go             # table DDL, migrations
  duckdb_test.go

pkg/ingest/
  ingest.go            # OTLP HTTP/gRPC receive, dispatches to plugin registry
  ingest_test.go

pkg/plugin/
  plugin.go            # Module interface + Registry
  registry.go           # register/lookup, route/schema collision checks
  plugin_test.go
  otelcore/
    otelcore.go          # first module: logs/metrics/traces
    otelcore_test.go
  llmagent/
    llmagent.go          # second module: prompts/tool-calls/agent spans
    llmagent_test.go

pkg/discovery/
  discovery.go         # Discoverer interface, orchestrator (runs all discoverers on a timer)
  processscan.go        # discoverer 1
  docker.go              # discoverer 2
  k8s.go                   # discoverer 3
  rules.go                  # loads curated (port, process/image) -> receiver config rules from conf/discovery-rules.yaml
  processscan_test.go
  docker_test.go
  k8s_test.go
  rules_test.go

pkg/rootcause/
  rootcause.go         # trace-close watcher + scoring entrypoint
  heuristic.go           # self-time calc, error-candidate, ranking
  rootcause_test.go
  heuristic_test.go

pkg/query/
  query.go             # trace-by-id, trace-list, logs, metrics query handlers
  query_test.go

pkg/api/
  api.go               # HTTP router, mounts ingest + query + healthz, wires module routes
  api_test.go

pkg/config/
  config.go            # config struct + loader (YAML)
  config_test.go

cmd/atlas-server/
  main.go              # load config -> open storage -> register modules -> start discovery -> start servers
  supervise.go          # recover+restart+log wrapper for background loops (rootcause watcher, discovery)
  supervise_test.go

conf/
  example.yaml          # sample: storage path, discovery rules ref, root-cause thresholds, trace-close timeout
  discovery-rules.yaml   # curated (port, process/image) -> receiver-config rules, external, netdata-sd-style
```

## Types & signatures

```go
// pkg/storage/storage.go
package storage

type Span struct {
    TraceID             string
    SpanID               string
    ParentSpanID           string
    ServiceName              string
    Name                       string
    StartTime                    time.Time
    EndTime                        time.Time
    StatusCode                       string // "ok" | "error" | "unset"
    Attributes                         map[string]any
    ResourceAttributes                   map[string]any
}

type Trace struct {
    TraceID              string
    FirstSeen              time.Time
    LastSeen                 time.Time
    ClosedAt                   *time.Time
    LikelyRootCauseSpanID        *string
    Reason                          *string
    SelfTimePct                       *float64
}

type Store interface {
    WriteSpans(ctx context.Context, spans []Span) error
    GetTraceSpans(ctx context.Context, traceID string) ([]Span, error)
    // ListRootArrivedTraces returns trace_ids whose root span (parent_span_id
    // IS NULL) has been written but the trace row is not yet closed_at.
    // Primary close trigger — checked by pkg/rootcause after every WriteSpans batch.
    ListRootArrivedTraces(ctx context.Context) ([]string, error)
    // ListStaleOpenTraces returns trace_ids with no new span since idleSince
    // AND no root span yet. Fallback close trigger only (crashed-hop case).
    ListStaleOpenTraces(ctx context.Context, idleSince time.Time) ([]string, error)
    MarkTraceClosed(ctx context.Context, traceID string, verdict CloseVerdict) error // targeted UPDATE: closed_at, likely_root_cause_span_id, reason, self_time_pct only — first_seen/last_seen untouched
    ListTraces(ctx context.Context, f TraceFilter) ([]Trace, error)
    GetTrace(ctx context.Context, traceID string) (*Trace, error)
    Close() error
}

// CloseVerdict avoids storage importing pkg/rootcause (would cycle back
// through pkg/rootcause's own storage.Store dependency). pkg/rootcause.Verdict
// converts to this shape at the call site in Watcher.Run.
type CloseVerdict struct {
    SpanID           string
    Reason              string
    SelfTimePct           float64
}

type TraceFilter struct {
    HasRootCause *bool
    Since          *time.Time
    Limit             int
}

// pkg/plugin/plugin.go
package plugin

type Module interface {
    Name() string
    RegisterSchema(s storage.SchemaRegistrar) error
    RegisterRoutes(r api.RouteRegistrar) error
    HandleSpans(ctx context.Context, spans []storage.Span) error
}

// SchemaRegistrar lets a module declare the tables it owns at registration
// time; storage.DuckDB implements it. Collision on table name is an error.
type SchemaRegistrar interface {
    CreateTable(ddl string) error
}

// RouteRegistrar lets a module mount its HTTP handlers at registration time;
// pkg/api.Router implements it. Collision on pattern is an error (this is the
// "route collision" TestRegister_RouteCollisionErrors checks).
type RouteRegistrar interface {
    Handle(pattern string, h http.Handler) error
}

type Registry struct { /* unexported: modules []Module, routes map[string]bool */ }

func NewRegistry() *Registry
func (r *Registry) Register(m Module) error   // errors on name or route collision
func (r *Registry) Dispatch(ctx context.Context, spans []storage.Span) error
// Dispatch routing rule: each span carries a resource attribute
// `atlas.module` naming its owning module (set by the SDK/exporter or
// defaulted by pkg/ingest); Dispatch routes to the Module whose Name()
// matches. A span with no `atlas.module` attribute defaults to "otelcore".
// An `atlas.module` value naming an unregistered module is an ingest error
// (dropped span, logged), not a silent no-op.

// pkg/discovery/discovery.go
package discovery

type Target struct {
    Host          string
    Port             int
    ProcessOrImage      string
    MatchedRule            *Rule
}

type Rule struct {
    Port           int
    ProcessMatch      string // substring or exact match against process name / image
    ReceiverConfig       string // OTel Collector receiver config template name
}

type Discoverer interface {
    Name() string
    Discover(ctx context.Context) ([]Target, error)
}

func RunAll(ctx context.Context, discoverers []Discoverer) ([]Target, []Target /* unrecognized */, error)

// pkg/rootcause/heuristic.go
package rootcause

type Verdict struct {
    SpanID           string
    Reason              string
    SelfTimePct           float64
}

func Score(spans []storage.Span, selfTimeThreshold float64) (*Verdict, error) // nil verdict = no candidate found (should not happen for non-empty trace)
func selfTime(span storage.Span, children []storage.Span) time.Duration
func earliestError(spans []storage.Span) *storage.Span

// This heuristic (self-time threshold + earliest-error) is the v1 starting
// point, built once the base structure below is running and there's real
// trace data to validate against — not a finished design. See future.md.

// pkg/rootcause/rootcause.go
package rootcause

type Watcher struct { /* unexported: store storage.Store, closeTimeout time.Duration, threshold float64 */ }

func NewWatcher(store storage.Store, closeTimeout time.Duration, threshold float64) *Watcher
// Run drives both close triggers each tick:
//   1. ListRootArrivedTraces (primary) — root span present means the tree is complete.
//   2. ListStaleOpenTraces (fallback) — idle past closeTimeout with no root span, crashed-hop case.
// Both paths converge on the same score-and-write step.
func (w *Watcher) Run(ctx context.Context, tick time.Duration) error
func (w *Watcher) closeAndScore(ctx context.Context, traceID string) error // GetTraceSpans -> Score -> MarkTraceClosed, shared by both triggers

// pkg/query/query.go
package query

type Handlers struct { /* unexported: store storage.Store */ }

func NewHandlers(store storage.Store) *Handlers
func (h *Handlers) GetTrace(w http.ResponseWriter, r *http.Request) // 404 + JSON {"error": "trace not found"} if storage.Store.GetTrace returns nil
// Returns the raw span tree + verdict, not a pre-shaped waterfall payload —
// keeps the endpoint view-agnostic so other trace views can render off the
// same response later without an API change. See future.md.
func (h *Handlers) ListTraces(w http.ResponseWriter, r *http.Request)

// pkg/config/config.go
package config

type Config struct {
    StoragePath         string        `yaml:"storage_path"`
    IngestAddr             string        `yaml:"ingest_addr"`
    APIAddr                   string        `yaml:"api_addr"`
    TraceCloseTimeout           time.Duration `yaml:"trace_close_timeout"` // fallback trigger only (root span never arrives); default 30s, matches OTel Collector tail_sampling decision_wait. Primary close trigger is root-span arrival, not this timeout.
    RootCauseSelfTimePct           float64       `yaml:"root_cause_self_time_pct"` // default 0.30
    DiscoveryRulesPath                string        `yaml:"discovery_rules_path"`
}

func Load(path string) (*Config, error)
```

## Call stack

**Ingest path:**
`cmd/atlas-server.main` → `pkg/ingest.Server.ServeOTLP` → `pkg/plugin.Registry.Dispatch` → `{otelcore,llmagent}.Module.HandleSpans` → `pkg/storage.Store.WriteSpans`

**Root-cause path (background loop, dual trigger):**
`pkg/rootcause.Watcher.Run` (ticks every `tick`) →
  (a) primary: `pkg/storage.Store.ListRootArrivedTraces` — root span present, tree is structurally complete
  (b) fallback: `pkg/storage.Store.ListStaleOpenTraces(idleSince = now - closeTimeout)` — no root span, idle past timeout, crashed hop
→ for each traceID from either list: `Watcher.closeAndScore` → `pkg/storage.Store.GetTraceSpans` → `pkg/rootcause.Score` → `pkg/storage.Store.MarkTraceClosed(ctx, traceID, storage.CloseVerdict{...})` (targeted update, first_seen/last_seen preserved)

A long-running child span (e.g. a 60s LLM call) no longer causes a premature close: (a) only fires once the root span itself has arrived, and the root span's `end_time` cannot be written until every span it waited on — including that LLM call — has completed and been received. (b) is a safety net for a hop that crashes and never emits a root span at all.

**Query path:**
`pkg/api.Router` → `pkg/query.Handlers.GetTrace` → `pkg/storage.Store.GetTrace` + `GetTraceSpans` → JSON response

**Discovery path (independent loop):**
`cmd/atlas-server.main` → `pkg/discovery.RunAll` (ticks) → each `Discoverer.Discover` → match against `pkg/discovery.Rule` set → (v1: log/expose matched + unrecognized targets via `pkg/query` or logs; writing live OTel Collector config is stretch, see Least confident decisions)

**Startup:**
`main` → `config.Load` → `storage.NewDuckDB(cfg.StoragePath)` → `plugin.NewRegistry()` → `registry.Register(otelcore.New(...))` → `registry.Register(llmagent.New(...))` → `supervise(rootcause.NewWatcher(...).Run)` (goroutine) → `supervise(discovery.RunAll)` (goroutine) → `api.NewRouter(...).ListenAndServe`

`supervise(fn)` (new, `cmd/atlas-server/supervise.go`): runs `fn` in a loop, `recover()`s a panic, logs it, restarts after a backoff (e.g. 1s, capped). Applies to both background loops so a panic in root-cause scoring or discovery doesn't silently freeze that subsystem while ingest/query keep serving stale state — no recover means the failure is invisible until someone notices traces stopped getting verdicts.

**Write contention note:** DuckDB is single-writer. `pkg/ingest.WriteSpans` batches per OTLP export call (not per-span) — this is the mitigation against contention with `pkg/rootcause.Watcher`'s scan-heavy poll loop. No write-queue/buffer is built in v1; slice 8 adds a load-test note to confirm this holds at target scale (single-user, tens of GB) rather than building speculative machinery now.

## Test plan

- `pkg/storage`: `TestWriteAndGetTraceSpans`, `TestListOpenTraces_ExcludesRecentlyActive`, `TestMarkTraceClosed_PersistsVerdict`, `TestMarkTraceClosed_PreservesFirstSeenLastSeen` (regression guard for the targeted-update fix), `TestListTraces_FiltersByHasRootCause`.
- `pkg/plugin`: `TestRegister_DuplicateNameErrors`, `TestRegister_RouteCollisionErrors`, `TestDispatch_RoutesToCorrectModule`, `TestDispatch_UnclaimedSpanDefaultsToOtelcore`, `TestDispatch_UnregisteredModuleAttributeLogsAndDrops`.
- `pkg/discovery`: `TestProcessScan_MatchesCuratedRule`, `TestProcessScan_UnmatchedPortSurfacedAsUnrecognized`, `TestDocker_MatchesImageName`, `TestK8s_AttachesPodResourceAttributes`, `TestRunAll_MergesAcrossDiscoverers`.
- `pkg/rootcause`: `TestScore_LinearChain_NoError_FlagsHighestSelfTime`, `TestScore_ErrorInChild_FlagsEarliestError`, `TestScore_FanOut_ComputesSelfTimeCorrectly`, `TestScore_NoErrorBelowThreshold_ReturnsNilOrLowConfidenceVerdict`, `TestWatcher_ClosesOnRootSpanArrival` (primary trigger), `TestWatcher_LongRunningChildSpan_DoesNotCloseTraceEarly` (regression guard for the LLM-call bug), `TestWatcher_ClosesStaleTraceAfterTimeout_NoRootSpan` (fallback trigger only).
- `pkg/query`: `TestGetTrace_ReturnsWaterfallAndVerdict`, `TestGetTrace_UnknownTraceID_Returns404`, `TestListTraces_RespectsFilterAndLimit`.
- `pkg/config`: `TestLoad_AppliesDefaults`, `TestLoad_RejectsInvalidYAML`.
- `cmd/atlas-server`: `TestSupervise_RecoversPanicAndRestarts`, `TestSupervise_LogsPanicBeforeRestart`.
- Integration (`tests/`): `TestEndToEnd_MultiHopTrace_ProducesExpectedRootCauseVerdict` send synthetic OTLP spans through real ingest, poll query API until verdict appears, assert against known-answer fixture.

## Least confident decisions

1. **Trace-close mechanism: poll vs. push.** Design uses a ticking `Watcher.Run` that polls both `ListRootArrivedTraces` and `ListStaleOpenTraces`. Confirmed against `docs/resources/netdata/src/go/plugin/agent/discovery/pipeline.go:176-177` netdata's own service-discovery pipeline runs on a `time.NewTicker`, not an event/push model, at comparable single-agent scale. Keeping poll; no repo pushed back on this.
2. **Trace-close trigger: idle-timeout was wrong as the primary signal — superseded.** Original design closed every trace purely on `TraceCloseTimeout` (30s idle). This broke on any trace containing a long-running span (an LLM call spanning tens of seconds): once sibling spans went quiet, the idle clock closed the trace *before* the LLM span arrived, silently losing the most interesting span and scoring an incomplete tree. Fixed: primary trigger is now **root-span arrival** (`parent_span_id IS NULL`, see `ListRootArrivedTraces` above) — structurally guaranteed complete, since a root span's `end_time` can't be written until everything it waited on has finished. `TraceCloseTimeout` (still **30s** default, matching OTel Collector's `tail_sampling` `decision_wait` convention) survives only as the fallback for a root span that never arrives at all (crashed hop). Validated against reference tools: Phoenix treats root-span identity as first-class schema (`docs/resources/phoenix/src/phoenix/trace/schemas.py:141-142`, `"If the parent_id is None, this is the root span"`); Langfuse never blocks on a trace-level idle window, it processes per-observation with retry/backoff instead (`docs/resources/langfuse/worker/src/features/evaluation/retryObservationNotFound.ts:17-23`). Neither reference tool uses idle-time as the primary completion signal — this was an Atlas-specific gap, now closed.
3. **Discovery → OTel Collector wiring in v1.** Checked netdata's HTTP/file service-discovery config (`docs/resources/netdata/src/go/plugin/go.d/config/go.d/sd/http.conf`): even netdata's SD only *produces* matched job configs for its own collector plugin to consume on next reload, it does not hot-patch a running third-party collector process either. That matches the proposal: v1 discovery **reports** matched/unrecognized targets (via endpoint/log), it does not hot-reload the OTel Collector. Auto-wiring (writing OTel Collector config, triggering reload) is a fast-follow, same shape as netdata's own SD-to-collector handoff.
4. `rootcause.Score` **return contract when no candidate clears the bar.** No direct reference: signoz/hyperdx/openobserve/phoenix/langfuse don't compute a root-cause verdict at all, so there's no upstream pattern to borrow here; this stays a judgment call. Keeping: always return a verdict (never nil), `Reason` explains low confidence when nothing crosses the threshold.
5. **Discovery rules format** (`pkg/discovery/rules.go`). Checked netdata's SD config shape directly (`docs/resources/netdata/src/go/plugin/go.d/config/go.d/sd/http.conf`): netdata ships discovery rules as an **external YAML file** (`disabled`, `interval`, `services` match/template rules), not compiled into the Go binary, this is the config editors actually touch to add a curated match without a rebuild. Reversing my earlier proposal: v1 should ship `conf/discovery-rules.yaml` (loaded by `pkg/config`, matching the (port, process/image) → receiver-config shape) instead of a hardcoded Go table, so it matches the pattern the reference repo actually ships and doesn't require a rebuild to add a rule.

