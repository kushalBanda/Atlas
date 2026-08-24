# Atlas — Project Folder Structure

Date: 2026-08-24
Status: Approved
Supersedes: none (companion to `2026-08-24-language-choice-design.md`)

## Context

Follow-up to the language/component stack decision (`docs/superpowers/specs/2026-08-24-language-choice-design.md`, which chose Go backend + TypeScript frontend + ClickHouse storage + reused OTel Collector agent). This spec fixes the monorepo folder layout, reviewed via `plan-eng-review`.

Reference: `docs/resources/signoz` was used as the structural template it already proves this layout in production.

## Decision

```
atlas/
  cmd/
    atlas-server/            # main entrypoint (Go)
  pkg/                        # flat, one dir per domain — matches SigNoz reference
    ingest/                    # OTLP ingest handlers
    query/                      # query API (logs/metrics/traces)
    storage/                     # ClickHouse client + schema/migrations
    api/                          # HTTP/gRPC handlers, routing
    config/                        # config loader
    ...                             # *_test.go colocated per package
  frontend/                   # TypeScript/React app
    src/
    public/
  deploy/
    docker-compose.yaml        # backend + ClickHouse + OTel Collector, single-node
    clickhouse/                 # ClickHouse schema/config
    otel-collector/              # collector config (reused upstream image)
    k8s/                          # optional, later
  conf/
    example.yaml                # sample runtime config
  tests/                       # integration/e2e only (unit tests stay in pkg/)
  docs/
    superpowers/specs/
    notes/
    resources/                  # existing cloned reference repos
  scripts/                     # dev/build/lint scripts
  .github/workflows/           # CI — deferred, NOT in scope this pass
  Makefile
  go.mod
```



## Key decisions and rationale

- **Flat** `pkg/<domain>`**, not** `internal/<domain>`**.** Matches the SigNoz reference exactly (`pkg/querier`, `pkg/alertmanager`, `pkg/sqlstore`, etc). Atlas is an application, not a library, so Go's `internal/`
import-restriction has no practical benefit here; flat `pkg/` keeps the layout directly comparable to the reference repo while learning from it.
- **Unit tests colocated,** `tests/` **reserved for integration/e2e.** Matches SigNoz: `pkg/<domain>/*_test.go` sit beside the code they test; the top-level `tests/` directory is for integration/e2e suites only, reserved now even though empty until the first such test is written.
- **ClickHouse and OTel Collector are external dependencies, not vendored code.** Wired through `deploy/` (compose/config), never copied into the repo consistent with the language-choice spec's "reuse, don't rebuild" decision.
- `conf/` **(data) is separate from** `pkg/config/` **(code).** `conf/` holds sample runtime YAML; `pkg/config/` holds the Go loader for it. Not a conflict different concerns.



## NOT in scope

- CI/CD pipeline (`.github/workflows/` folder reserved, empty).
- Build/publish/distribution flow for any artifact.
- Kubernetes manifests (`deploy/k8s/` folder reserved, empty).



## What already exists

- `docs/resources/signoz` direct structural template for this layout.
- `docs/resources/hyperdx`, `docs/resources/openobserve`,
`docs/resources/netdata` available for cross-reference on specific subsystems later, not used as the primary template for this pass.



## Review notes (plan-eng-review)

- Step 0 scope challenge: accepted as-is, no reduction needed (structure only, 7 top-level folders, no new classes/services).
- Architecture: 1 finding resolved `pkg/` flat over `internal/`.
- Code Quality: no issues.
- Test Review: 1 gap resolved added top-level `tests/` for integration/e2e.
- Performance: no issues (no code yet; ClickHouse/OTel Collector are external, already-optimized dependencies).
- Independent second opinion: skipped (low-stakes, structure-only, closely modeled on a proven reference).
- Parallelization: sequential, single structural change, no parallel lanes.
- Unresolved decisions: none.



## Consequences

- New Go code goes under `pkg/<domain>/`, not `internal/`.
- Every new package gets its `_test.go` file colocated in the same directory; only integration/e2e tests go in `tests/`.
- CI/CD and distribution setup are explicitly deferred to a future spec the folders exist as placeholders but are not to be populated in this pass.

