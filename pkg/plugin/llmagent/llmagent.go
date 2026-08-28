// Package llmagent is the second plugin module: prompt/tool-call/agent
// spans routed here via atlas.module. LLM field extraction moved to
// pkg/fields and runs regardless of module, so llmagent is currently
// structurally identical to otelcore; kept for future llmagent-only
// routes/tables.
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

// HandleSpans implements plugin.Module: writes spans straight to storage.
// LLM field extraction already ran in Registry.Dispatch before spans reach
// here (see pkg/fields).
func (m *Module) HandleSpans(ctx context.Context, spans []storage.Span) error {
	if err := m.store.WriteSpans(ctx, spans); err != nil {
		return fmt.Errorf("llmagent writing spans: %w", err)
	}
	return nil
}
