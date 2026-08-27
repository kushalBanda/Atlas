# Product: Atlas v1 (backend)

## Problem

A developer running a stack with a mix of normal services and LLM/agent code has no single place to see a request's full path. Today they check one tool for HTTP/DB traces and a different tool for LLM/agent traces, and neither tool tells them which step in a broken request actually caused the failure. They have to eyeball a waterfall by hand.

## Success metric

For a trace with an error or a dominant slow span, the tool names the correct root-cause span at least 80% of the time on a hand-labeled test set of synthetic traces (linear chain, fan-out, error-in-child, slow-leaf). Measured by the `pkg/rootcause` test suite plus a small manually-labeled fixture set.

## Announcement — the blog post before the feature

Atlas is a single binary that watches your stack and tells you what broke. Point it at a machine, a Docker host, or a Kubernetes cluster, and it finds your services on its own; no collector config to write by hand. Every request, whether it hits a database, a REST API, or an LLM agent's tool call, shows up as one trace. When a request fails or runs slow, Atlas points straight at the span responsible and says why, instead of leaving you to read a waterfall chart yourself.

## Screens

No UI this pass. Output is a query API (HTTP/gRPC) a human or a future UI can call. Verification for this plan is via API calls (curl) and test suites, not screens. Waterfall is the assumed v1 rendering shape once a UI exists — see `future.md` for why the query API stays view-agnostic instead of baking that in.

## Deferred

Two items scoped out of this doc, tracked in `future.md`: which trace-view(s) a future UI renders beyond waterfall, and deeper root-cause-scoring design (threshold validation, scope expansion) once the base backend is running with real data.
