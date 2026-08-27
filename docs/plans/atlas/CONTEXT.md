# Atlas

Single-user, developer-first observability platform unifying infra and LLM/agent traces in one span model, with automatic root-cause pinpointing.

## Language

**Root span**:
The span in a trace whose `parent_span_id` is null, the entry point of a request. Its `end_time` cannot be recorded until every span it depends on, direct or transitive, has completed. Root-span arrival is the structural signal that a trace's span tree is complete.

**Trace close**: The point at which a trace is considered final and eligible for root-cause scoring. Two triggers: root-span arrival (primary, the trace is structurally complete) or the trace-close timeout (fallback, no root span arrived at all, e.g. a crashed hop). A closed trace's `traces` row gets a written verdict and never re-opens.
*Avoid*: "trace timeout" alone (ambiguous, implies timeout is always the trigger, when it's the fallback).

**Verdict**:
The output of root-cause scoring for a closed trace: `likely_root_cause_span_id`, a human-readable `reason`, and `self_time_pct`. Written once, at close, never recomputed.

**Self-time**: A span's own duration minus the total duration of its direct children the time a span spent doing work itself, not delegated.