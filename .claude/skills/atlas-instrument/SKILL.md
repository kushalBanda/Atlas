---
name: atlas-instrument
description: Add OpenTelemetry tracing to a service so its LLM/agent calls show up in Atlas with root-cause scoring. Use when the user wants to send traces or LLM spans to Atlas, or asks to instrument a service for Atlas.
---

Instrument a service so its spans land in Atlas (self-hosted OTel trace store, DuckDB-backed, with automatic root-cause scoring at trace close). This is a one-shot OTLP export, not a live integration — no server or SDK from Atlas itself is needed on the client side, only a standard OTel exporter.

## Atlas ingest contract

- Endpoint: `http://<atlas-host>:4318/v1/traces` (OTLP/HTTP, standard fixed path — do not add a version prefix, do not change it)
- Default host if Atlas runs on the same machine: `127.0.0.1`
- Every span's resource attributes must set `atlas.module`, naming which Atlas plugin module owns it:
  - `llmagent` — LLM calls, agent steps, tool calls (anything gen_ai-shaped)
  - `otelcore` — everything else (default if the attribute is absent, so it only needs to be set explicitly for `llmagent` spans)

## gen_ai.* attributes (llmagent module)

Set these span attributes on any span representing an LLM call, so Atlas parses them into typed columns instead of leaving them as opaque JSON:

| Attribute | Type | Meaning |
|---|---|---|
| `gen_ai.request.model` | string | model name/id requested |
| `gen_ai.response.model` | string | model name/id actually used (fallback if request.model absent) |
| `gen_ai.usage.prompt_tokens` / `gen_ai.usage.input_tokens` | int | prompt token count (either key works) |
| `gen_ai.usage.completion_tokens` / `gen_ai.usage.output_tokens` | int | completion token count (either key works) |
| `gen_ai.usage.cost` | float | cost of the call, if the provider returns it |
| `gen_ai.prompt` | string | full prompt text |
| `gen_ai.completion` | string | full completion text |

`gen_ai.request.model`/`response.model`, prompt/completion token counts, and `gen_ai.usage.cost` are extracted into typed, queryable columns (`LLMModel`, `LLMPromptTokens`, `LLMCompletionTokens`, `LLMCost`). `gen_ai.prompt`/`gen_ai.completion` are full-text and not duplicated into typed columns — still readable, just from the span's `attributes` JSON, not a dedicated field.

This extraction is attribute-driven, not module-gated: it runs on every span regardless of `atlas.module`, so tagging a span `llmagent` is not required for `gen_ai.*` extraction to happen — only `gen_ai.*` keys being present on the span matters.

Non-LLM spans (tool calls, agent steps) don't need these — set whatever attributes make sense (`tool.name`, `tool.arguments`, `tool.output`, ...), they land as-is in the span's attributes JSON.

## Minimal setup (Python example — adapt per language, same OTel SDK concepts apply everywhere)

1. Install: `opentelemetry-api`, `opentelemetry-sdk`, `opentelemetry-exporter-otlp-proto-http`
2. Once at startup:

```python
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

resource = Resource.create({"service.name": "<your-service>", "atlas.module": "llmagent"})
provider = TracerProvider(resource=resource)
provider.add_span_processor(BatchSpanProcessor(
    OTLPSpanExporter(endpoint="http://127.0.0.1:4318/v1/traces")
))
trace.set_tracer_provider(provider)
tracer = trace.get_tracer("<your-service>")
```

3. Wrap each LLM call (or tool call, or agent step) in its own span. Spans opened while another is current nest automatically (parent-child), so wrapping an outer "agent run" span around inner "tool call" and "LLM call" spans produces a correct trace tree with no extra wiring:

```python
with tracer.start_as_current_span("llm.chat_completion") as span:
    span.set_attribute("gen_ai.request.model", model_name)
    span.set_attribute("gen_ai.prompt", prompt_text)
    # ... make the call ...
    span.set_attribute("gen_ai.completion", result_text)
    span.set_attribute("gen_ai.usage.prompt_tokens", usage["prompt_tokens"])
    span.set_attribute("gen_ai.usage.completion_tokens", usage["completion_tokens"])
```

A working reference implementation (FastAPI + httpx, single LLM call and a nested agent/tool-call example) lives at `scripts/ai-gateway/main.py` in this repo — read it for a complete working example if adapting an existing service.

## Cutting boilerplate

For HTTP-framework-level spans (a span per inbound request, per outbound HTTP call), prefer auto-instrumentation over hand-wrapping when a library exists for the stack in use (e.g. `opentelemetry-instrumentation-fastapi`, `opentelemetry-instrumentation-httpx` for Python; equivalents exist for most frameworks). These auto-create correctly parented spans with zero manual `start_as_current_span` calls. The `gen_ai.*` attributes still need to be set manually — no generic instrumentor knows Atlas's attribute contract — but they're a few `set_attribute` calls inside the already-existing call site, not new span-management code.

If the LLM call goes through an SDK that has a dedicated OTel instrumentor (e.g. `opentelemetry-instrumentation-openai` for OpenAI-SDK-shaped clients), prefer that over manual attribute-setting — it sets `gen_ai.*` automatically. This only applies when the SDK shape matches; a raw REST call to a provider (as in the reference implementation) has no such instrumentor and needs the manual attributes above.

## Verifying it worked

1. Start Atlas: `go run ./cmd/atlas-server -config ./conf/example.yaml` (or wherever it's deployed)
2. Trigger the instrumented call in the target service
3. Grab the trace ID — either log `format(span.get_span_context().trace_id, "032x")` at the point the outermost span is created, or return it in the service's own response for lookup
4. Query it back: `GET http://<atlas-host>:8080/traces/{trace_id}`
5. Confirm: the trace has the expected span tree, `LLMModel`/`LLMPromptTokens`/`LLMCost`/etc. are populated (not null) on LLM spans, and — once the trace closes (root span arrived, or after the idle timeout) — `LikelyRootCauseSpanID` and `Reason` are set on the trace itself
