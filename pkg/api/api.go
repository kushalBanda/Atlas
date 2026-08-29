// Package api wires the HTTP router: query, health, and plugin-module
// routes. The ingest listener binds a separate address and is wired
// directly in cmd/atlas-server (it keeps the OTLP/HTTP fixed /v1/ path).
package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"atlas/pkg/query"
)

// spaRoutePrefixes are the frontend's own route paths that collide with an
// Atlas API path of the same shape (GET /traces/{trace_id} vs the SPA's
// /traces/:traceId page). A browser hard-navigation (or refresh) at one of
// these deep links must render the SPA shell, not the raw JSON the same
// path returns to a fetch() call. Mirrors vite.config.ts's htmlNavBypass,
// which solves the identical ambiguity in dev via the same Accept-header
// check against the same two prefixes.
var spaRoutePrefixes = []string{"/traces", "/discovery"}

// RouteRegistrar lets a plugin module mount its HTTP handlers at
// registration time. Router implements this. Collision on pattern is an
// error (see TestRouter_HandleCollisionErrors and
// TestRegister_RouteCollisionErrors in pkg/plugin).
type RouteRegistrar interface {
	Handle(pattern string, h http.Handler) error
}

// Router is Atlas's HTTP router: mounts query + health endpoints directly
// and accepts further routes from plugin modules through RouteRegistrar.
type Router struct {
	mux       *http.ServeMux
	patterns  map[string]bool
	staticDir string
}

// NewRouter mounts Atlas's own query + health endpoints. Atlas's own API
// (/traces, /healthz, ...) has no /v1/ prefix.
//
// staticDir, if non-empty and present on disk, mounts a static-file route
// at "/" serving the built frontend (see docs/plans/atlas-frontend). An
// empty staticDir, or one that doesn't exist, registers no static route
// at all — a warning is logged in the latter case, but this is never an
// error: a missing frontend build must not prevent the API from serving.
// This mirrors loadConfig's own "warn and fall back" pattern in
// cmd/atlas-server/main.go for a missing -config file.
func NewRouter(queryHandlers *query.Handlers, staticDir string) *Router {
	r := &Router{
		mux:       http.NewServeMux(),
		patterns:  make(map[string]bool),
		staticDir: staticDir,
	}

	// Core routes are registered through the same Handle path as plugin
	// modules so a module can never silently shadow one of these.
	mustHandle(r, "GET /traces", http.HandlerFunc(queryHandlers.ListTraces))
	mustHandle(r, "GET /traces/{trace_id}", http.HandlerFunc(queryHandlers.GetTrace))
	mustHandle(r, "GET /stats", http.HandlerFunc(queryHandlers.GetStats))
	mustHandle(r, "GET /healthz", http.HandlerFunc(healthz))

	if staticDir != "" {
		if _, err := os.Stat(staticDir); err != nil {
			slog.Warn("static dir not found, frontend will not be served", "static_dir", staticDir, "error", err)
			r.staticDir = ""
		} else {
			// Registered through Handle too, so it can never silently
			// shadow a plugin module's route registered after this call.
			mustHandle(r, "/", http.FileServer(http.Dir(staticDir)))
		}
	}

	return r
}

func mustHandle(r *Router, pattern string, h http.Handler) {
	if err := r.Handle(pattern, h); err != nil {
		// Core route registration is not attacker- or config-controlled;
		// a collision here is a programming error, caught at startup.
		panic(err)
	}
}

// Handle implements RouteRegistrar. Registering the same pattern twice is
// an error, not a silent overwrite.
func (r *Router) Handle(pattern string, h http.Handler) error {
	if r.patterns[pattern] {
		return fmt.Errorf("route already registered: %q", pattern)
	}
	r.mux.Handle(pattern, h)
	r.patterns[pattern] = true
	return nil
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.isSPANavigation(req) {
		http.ServeFile(w, req, filepath.Join(r.staticDir, "index.html"))
		return
	}
	r.mux.ServeHTTP(w, req)
}

// isSPANavigation reports whether req is a browser hard-navigation (or
// refresh) at a path the SPA owns but that also matches one of Atlas's own
// API route patterns (GET /traces/{trace_id}, /discovery). A fetch()/XHR
// call from the already-loaded React app sends Accept: application/json
// and must still reach the real handler; only a browser page load, whose
// Accept header lists text/html, gets redirected to the SPA shell instead.
func (r *Router) isSPANavigation(req *http.Request) bool {
	if r.staticDir == "" || req.Method != http.MethodGet {
		return false
	}
	if !strings.Contains(req.Header.Get("Accept"), "text/html") {
		return false
	}
	for _, prefix := range spaRoutePrefixes {
		if req.URL.Path == prefix || strings.HasPrefix(req.URL.Path, prefix+"/") {
			return true
		}
	}
	return false
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
