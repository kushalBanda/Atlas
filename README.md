# Atlas

**OpenTelemetry-native observability for distributed services, LLM calls, and AI agent runs.**

Atlas is a self-hosted, single-binary observability platform for answering a practical question quickly: **what broke, where did it break, and why?** Send it OpenTelemetry traces, inspect the complete request path, and get an actionable explanation instead of another waterfall to investigate manually.

![Atlas logo](assets/Logo.png)

## Find the cause, not just the trace

Modern requests cross HTTP services, databases, model providers, and agent tools. Atlas is built for the moment after an alert, when a team needs to move from signal to explanation. Its core advantage is **causal debugging for complex software paths**: Atlas uses trace structure, span timing, status, LLM metadata, and agent relationships to surface the span most likely responsible for the incident.

With Atlas, teams can:

- **Locate the likely root cause** — identify the earliest error, or the dominant self-time span when no error is present.
- **Understand the full request path** — follow one request across services, databases, APIs, model providers, and tools.
- **Debug AI systems as systems** — correlate prompts, models, tokens, cost, latency, sessions, and agent steps with the surrounding application trace.
- **See agent behavior clearly** — follow runs across traces and detect repeated steps, retry storms, and tool loops.
- **Start with almost no setup** — discover local processes, Docker containers, and Kubernetes targets, then inspect the telemetry Atlas receives.
- **Keep ownership of your data** — use standard OpenTelemetry ingestion and store telemetry locally in embedded DuckDB.



## Quickstart



### Requirements

- Go 1.25 or newer
- Node.js and npm for building the bundled web interface
- An application or OpenTelemetry Collector that can export OTLP/HTTP traces



### Run from source

```bash
git clone <your-atlas-checkout>
cd Atlas
make run
```

`make run` builds the frontend and starts `atlas-server`. By default:

- Web UI and query API: `http://127.0.0.1:8080`
- OTLP/HTTP ingest: `http://127.0.0.1:4318/v1/traces`
- DuckDB database: `./atlas.duckdb`

Check that the server is ready:

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

Configure your OpenTelemetry exporter to send traces to:

```text
http://127.0.0.1:4318/v1/traces
```

Atlas accepts `application/x-protobuf` and `application/json` OTLP/HTTP trace exports. Once data arrives, open `http://127.0.0.1:8080` to browse traces, runs, sessions, and discovered targets.

### Run the server directly

```bash
go run ./cmd/atlas-server -config ./conf/example.yaml
```

If the config file is missing, Atlas uses its defaults. If a config file exists but is invalid, startup fails with a validation error.

## Product surface



### Traces

The trace view exposes the complete span tree, status, duration, raw OTel attributes, typed LLM fields, and the likely root-cause span. The current heuristic is intentionally small and explainable:

1. The earliest error span wins.
2. Otherwise, the span with the highest self-time is selected if it exceeds the configured threshold.



### LLM and agent observability

Atlas reads self-describing attributes from any span, including spans routed through the `otelcore` or `llmagent` modules. Agent metadata such as `agent.run.id`, `agent.name`, `agent.step.kind`, `session.id`, and `user.id` powers the run and session views.

Run graphs can cross trace boundaries and annotate consecutive repeated steps, making retry storms and tool loops easier to spot.

### Discovery

The discovery endpoint reports recognized and unrecognized targets found by local process scanning, Docker, and Kubernetes in-cluster discovery. Discovery currently reports targets only. It does not hot-patch an OpenTelemetry Collector configuration.

## HTTP API


| Method | Endpoint                 | Purpose                                       |
| ------ | ------------------------ | --------------------------------------------- |
| `POST` | `/v1/traces`             | Receive OTLP/HTTP traces as JSON or protobuf  |
| `GET`  | `/healthz`               | Liveness check                                |
| `GET`  | `/traces`                | List traces with optional filters             |
| `GET`  | `/traces/{trace_id}`     | Fetch a trace and its spans                   |
| `GET`  | `/stats`                 | Aggregate trace, span, and LLM statistics     |
| `GET`  | `/runs`                  | List agent runs                               |
| `GET`  | `/runs/{run_id}`         | Fetch an agent run, spans, and decision graph |
| `GET`  | `/sessions`              | List sessions                                 |
| `GET`  | `/sessions/{session_id}` | Fetch a session                               |
| `GET`  | `/discovery/targets`     | Fetch the latest discovery results            |


The Postman collection in `[docs/atlas.postman_collection.json](docs/atlas.postman_collection.json)` provides a ready-made API exploration surface.

## Configuration

Copy `[conf/example.yaml](conf/example.yaml)` and change only what you need:

```yaml
storage_path: ./atlas.duckdb
ingest_addr: 127.0.0.1:4318
api_addr: 127.0.0.1:8080
trace_close_timeout: 30s
root_cause_self_time_pct: 0.30
discovery_rules_path: ./conf/discovery-rules.yaml
agent_run_repeat_threshold: 3
```

The default listeners bind to loopback because Atlas has no authentication. If you expose Atlas beyond the local machine, put it behind an appropriate network boundary or authenticated reverse proxy.

## Deployment

Atlas can run as a Go binary or inside your own container environment. The repository includes starter assets under `[deploy/](deploy/)`, including an OpenTelemetry Collector configuration and Docker Compose files. The current Compose file is a development baseline and is not yet a turnkey Atlas production deployment.

For a production rollout, plan persistent storage for `atlas.duckdb`, network access from instrumented services to the OTLP listener, access control at the network or reverse-proxy layer, retention and backup, and a capacity test for your expected span volume.

## Development

```bash
make build   # build frontend and Go binaries
make test    # run Go and frontend tests
go vet ./...
go test -race ./cmd/... ./pkg/...
```

The backend is organized around storage, OTLP ingest, plugin modules, field extraction, discovery, root-cause analysis, query handlers, and configuration. The frontend lives in `[frontend/](frontend/)`. Start with `[CLAUDE.md](CLAUDE.md)` for repository conventions and current project status.

## Contributing

Issues, design feedback, tests, and implementation improvements are welcome. Before changing backend behavior, read the architecture and status documents linked above, then run the relevant package tests and race checks.

## License

Atlas is released under the [MIT License](LICENSE).