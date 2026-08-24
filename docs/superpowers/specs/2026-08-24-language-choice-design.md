# Atlas: Language and Component Stack Choice

Date: 2026-08-24
Status: Approved

## Context

Atlas is a new open-source observability platform. Prior research (`docs/notes/observability-platform-alternatives.md`) surveyed 20+ OSS observability projects. Four were cloned into `docs/resources/` for closer study: SigNoz (Go + TypeScript, ClickHouse), HyperDX (TypeScript + Rust, ClickHouse), OpenObserve (Rust, single binary), Netdata (C, agent).

This spec decides the language/component split for Atlas's stack.

## Decision drivers

- Priority: blend of performance and dev speed (not pure max-performance).
- Team is open to a polyglot split by component purpose, not one language for everything.
- Target scale: small / self-hosted first (single node or small cluster), not hyperscale from day one.



## Decision

Split by component, matching the pattern used by SigNoz and HyperDX:


| Component                                                   | Choice                                                         | Reason                                                                                             |
| ----------------------------------------------------------- | -------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Agent / collector                                           | OpenTelemetry Collector (Go, reused as-is)                     | Already the OSS standard; no reason to write a new one                                             |
| Ingest + query backend                                      | Go                                                             | Good enough performance, fast dev speed, large OTel/Go ecosystem, matches SigNoz/HyperDX precedent |
| Storage                                                     | ClickHouse (existing project, not custom-built)                | Proven for logs/metrics/traces workload; used by SigNoz, HyperDX, Uptrace                          |
| Frontend                                                    | TypeScript / React                                             | Standard choice; used by all four cloned reference repos                                           |
| Future CPU-hot paths (e.g. custom query engine, eBPF agent) | Rust, added later only if profiling shows Go is the bottleneck | Keep initial scope simple; add Rust only where proven necessary                                    |




## Rejected alternatives

- **Rust-centric backend** (OpenObserve/Parseable model): best raw performance, but slower dev velocity and smaller hiring pool. Not justified at small/self-host scale where dev speed matters more.
- **Custom storage engine**: unnecessary risk; ClickHouse already solves this problem and is proven in three of the four reference repos.



## Consequences

- Go becomes the primary backend language; contributors need Go + TypeScript, not Rust, to work on the core platform initially.
- Rust is not ruled out — it stays available as a targeted tool for a specific future bottleneck, not a whole-stack commitment.
- Reusing OTel Collector as the agent avoids building and maintaining a second collection agent from scratch.

