# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository's `frontend/` directory.

## Project state

Atlas's frontend is done. All 4 gates and all 8 slices are complete (see `docs/plans/atlas-frontend/00-status.md`). It has 3 screens, all wired to the real backend, no mocks:

- **Trace list** (`/traces`) — filters and client-side pagination.
- **Trace detail** (`/traces/:traceId`) — waterfall view, root-cause verdict line, LLM span badges, span detail panel.
- **Discovery** (`/discovery`) — discovery targets list.

Stack: React 19 + Vite + TypeScript + Tailwind v4 + TanStack Query + React Router. No React Compiler, no shadcn/ui components pulled in beyond what is already in `src/components`.

## Commands

Run these from `frontend/`:

```
npm run dev      # vite dev server, proxies API calls to atlas-server on :8080
npm run build    # tsc -b && vite build, outputs to frontend/dist
npm run lint     # oxlint
npm run test     # NODE_ENV=test vitest run
```

Run a single test file:
```
NODE_ENV=test npx vitest run src/lib/verdict.test.ts
```

**`NODE_ENV` must be pinned to `test` for Vitest.** An ambient `NODE_ENV=production` makes `react` resolve its production build, which strips `React.act`, and every `@testing-library/react` render throws `React.act is not a function`. Always run tests through the `test` npm script (or the repo root `make test`), never `vitest run` bare with an unknown ambient `NODE_ENV`.

From the repo root, `make build` runs `npm --prefix frontend run build` before the Go build, so a fresh checkout has a frontend to serve. `make run` starts `atlas-server`, which serves the built frontend from `./frontend/dist` (or via the `-static-dir` flag) at `/`, alongside the API, one origin one process.

## Architecture

Full design rationale lives in `docs/plans/atlas-frontend/02-architecture.md` and `docs/plans/atlas-frontend/00-status.md`; read those before changing data flow or adding a screen.

**No new backend endpoints.** The frontend only reads three existing read-only endpoints:
- `GET /traces?has_root_cause=&since=&limit=`
- `GET /traces/{trace_id}`
- `GET /discovery/targets`

**Data fetching**: every screen uses a TanStack Query hook that calls `apiGet<T>` (`src/api/client.ts`). `ApiError` carries the HTTP status so callers can branch: 404 means "does not exist, don't offer retry"; anything else means "transient, offer retry." No websockets, no polling — pages fetch on mount and on filter/param change; refresh is manual.

**Pagination is client-side.** The backend has no cursor param (`since` is a lower bound only, ordered newest-first — there is no way to page toward older rows through it). The frontend fetches one batch capped at `limit: 200` and paginates in the browser (25 rows/page). Do not reintroduce a `since`-as-cursor assumption.

**Backend JSON field-naming gotcha**: `Trace`/`Span`/`Target`/`Rule` structs have no JSON tags, so they serialize as Go's PascalCase field names (`TraceID`, `FirstSeen`, `ClosedAt`, ...), not snake_case. Only query-layer wrapper structs and computed fields (`duration_nano`, `matched`/`unrecognized`) are explicitly tagged lowercase. Check `src/api/types.ts` against a live `curl /traces` before assuming a field name.

**Discovery nil-slice gotcha**: Go's `json.Marshal` encodes a nil slice as `null`, not `[]`. `discovery.Handler.ServeTargets` passes `matched`/`unrecognized` through unwrapped, so an empty discovery result is `null` on the wire. `pkg/query`'s wrappers already guard against this for traces/spans, but discovery does not — any discovery-consuming code must normalize with `?? []` before touching `.length`/`.map`.

**Dev vs prod request flow**: in dev, `vite.config.ts`'s `server.proxy` forwards `/traces*` and `/discovery/*` to `atlas-server` on `:8080`, with an HTML-navigation bypass (`htmlNavBypass`) so a browser hard-navigation to `/traces` or `/discovery` (which are both API paths and SPA route paths) falls through to the SPA's `index.html` instead of hitting the backend. In prod, the built frontend and the API are one origin, one process, one port — no proxy, no CORS handling anywhere.

## Code organization

- `src/api/` — `client.ts` (fetch wrapper + `ApiError`), `traces.ts`, `discovery.ts` (query hooks), `types.ts` (response shapes).
- `src/lib/` — pure logic, unit-tested: `verdict.ts`, `spanTree.ts`, `duration.ts`, `llm.ts`, `stats.ts`.
- `src/components/` — presentational components; `waterfall/` holds the trace-detail waterfall's sub-components (`Waterfall.tsx`, `SpanRow.tsx`, `TimelineBar.tsx`).
- `src/pages/` — one file per route (`TraceList`, `TraceDetail`, `Discovery`).
- `src/test/` — shared test setup (`setup.ts`), fixtures, and `renderWithProviders.tsx` (wraps a component with the router + query client needed for most tests).

Keep pure/computed logic in `src/lib/`, not inline in components or hooks — this is why `hasLLMFields` was moved out of `LLMBadges.tsx` into `lib/llm.ts` (a component file must only export the component, or Vite's fast-refresh breaks) and why `TraceList`'s `since` timestamp computation was moved out of a render-phase `useMemo` into `useTraces`'s `queryFn` (computing `Date.now()`-derived values during render is an impurity that breaks React's rendering guarantees).

## Visual direction

Dark theme following Claude Code's own palette: warm dark charcoal background (`#1f1e1d`/`#262524`), off-white text (`#ece8e1`), Claude orange accent (`#d97757`). Density and layout follow SigNoz; LLM span badges follow Langfuse (see `docs/resources/signoz`, `docs/resources/langfuse` for the reference source). Fonts are self-hosted `@fontsource/ibm-plex-sans` and `@fontsource/ibm-plex-mono`, not loaded from Google Fonts at runtime. Do not use em-dash in UI copy; use comma, colon, or period instead.
