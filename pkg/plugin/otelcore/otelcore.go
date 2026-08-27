// Package otelcore is the first plugin module: it wraps the tracer-bullet
// ingest/query path (logs/metrics/traces on the core spans/traces tables).
// It owns no additional tables and adds no additional routes — both already
// exist from Slice 1 — so RegisterSchema and RegisterRoutes are no-ops.
package otelcore

import (
	"context"
	"fmt"

	"atlas/pkg/api"
	"atlas/pkg/storage"
)

// Name is the module name spans use in their atlas.module resource
// attribute, and the default for spans that omit it.
const Name = "otelcore"

// Module is the otelcore plugin module.
type Module struct {
	store storage.Store
}

// New returns an otelcore Module writing to store.
func New(store storage.Store) *Module {
	return &Module{store: store}
}

// Name implements plugin.Module.
func (m *Module) Name() string { return Name }

// RegisterSchema implements plugin.Module. No-op: the core spans/traces
// tables are created directly by storage.NewDuckDB, not through this hook.
func (m *Module) RegisterSchema(_ storage.SchemaRegistrar) error { return nil }

// RegisterRoutes implements plugin.Module. No-op: the query routes this
// module would own (/traces/{trace_id}, /healthz) are already mounted
// directly by api.NewRouter.
func (m *Module) RegisterRoutes(_ api.RouteRegistrar) error { return nil }

// HandleSpans implements plugin.Module: writes spans straight to storage.
func (m *Module) HandleSpans(ctx context.Context, spans []storage.Span) error {
	if err := m.store.WriteSpans(ctx, spans); err != nil {
		return fmt.Errorf("otelcore writing spans: %w", err)
	}
	return nil
}
