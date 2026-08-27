package discovery

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Handler serves the latest discovery results and refreshes them on a
// timer, independent of the request path (see package doc).
type Handler struct {
	discoverers []Discoverer

	mu           sync.RWMutex
	matched      []Target
	unrecognized []Target
}

// NewHandler returns a Handler running discoverers.
func NewHandler(discoverers []Discoverer) *Handler {
	return &Handler{discoverers: discoverers}
}

// Run refreshes discovery results every tick until ctx is canceled.
func (h *Handler) Run(ctx context.Context, tick time.Duration) error {
	h.refresh(ctx) // populate immediately, don't wait for the first tick

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			h.refresh(ctx)
		}
	}
}

func (h *Handler) refresh(ctx context.Context) {
	matched, unrecognized, err := RunAll(ctx, h.discoverers)
	if err != nil {
		slog.ErrorContext(ctx, "discovery run had errors", "error", err)
	}

	h.mu.Lock()
	h.matched = matched
	h.unrecognized = unrecognized
	h.mu.Unlock()
}

type targetsResponse struct {
	Matched      []Target `json:"matched"`
	Unrecognized []Target `json:"unrecognized"`
}

// ServeTargets handles GET /discovery/targets: the latest matched and
// unrecognized targets found. v1 reports only — it doesn't hot-patch the
// OTel Collector.
func (h *Handler) ServeTargets(w http.ResponseWriter, _ *http.Request) {
	h.mu.RLock()
	resp := targetsResponse{Matched: h.matched, Unrecognized: h.unrecognized}
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encoding discovery targets response failed", "error", err)
	}
}
