// Package llmagent is the second plugin module: prompt/tool-call/agent
// spans (gen_ai.* semantic-convention attributes, carried in Span.Attributes
// — no schema change needed on the spans table itself). Structurally
// identical to otelcore, just a different Name().
package llmagent

import (
	"context"
	"fmt"

	"atlas/pkg/api"
	"atlas/pkg/storage"
)

// Name is the module name spans use in their atlas.module resource
// attribute to route to this module instead of the otelcore default.
const Name = "llmagent"

// Module is the llmagent plugin module.
type Module struct {
	store storage.Store
}

// New returns an llmagent Module writing to store.
func New(store storage.Store) *Module {
	return &Module{store: store}
}

// Name implements plugin.Module.
func (m *Module) Name() string { return Name }

// RegisterSchema implements plugin.Module. No-op: prompt/tool-call data
// lives in the existing spans.attributes JSON column (gen_ai.* keys), not
// a new table.
func (m *Module) RegisterSchema(_ storage.SchemaRegistrar) error { return nil }

// RegisterRoutes implements plugin.Module. No-op in v1: llmagent has no
// routes of its own yet (query still goes through the shared /traces path).
func (m *Module) RegisterRoutes(_ api.RouteRegistrar) error { return nil }

// HandleSpans implements plugin.Module: writes spans straight to storage,
// same path as otelcore.
func (m *Module) HandleSpans(ctx context.Context, spans []storage.Span) error {
	if err := m.store.WriteSpans(ctx, spans); err != nil {
		return fmt.Errorf("llmagent writing spans: %w", err)
	}
	return nil
}
