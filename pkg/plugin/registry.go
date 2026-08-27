package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"atlas/pkg/api"
	"atlas/pkg/storage"
)

// Registry holds registered modules and dispatches ingested spans to them.
//
// Deviation from 03-program-design.md: NewRegistry takes explicit
// schema/route registrars (typically storage.DuckDB and api.Router) rather
// than the doc's zero-arg signature, which didn't say where those come
// from. See docs/design-considerations.md.
type Registry struct {
	schema  storage.SchemaRegistrar
	routes  api.RouteRegistrar
	modules map[string]Module
}

// NewRegistry returns an empty Registry. schema and routes are where
// registered modules' RegisterSchema/RegisterRoutes calls land.
func NewRegistry(schema storage.SchemaRegistrar, routes api.RouteRegistrar) *Registry {
	return &Registry{
		schema:  schema,
		routes:  routes,
		modules: make(map[string]Module),
	}
}

// Register adds m to the registry. Errors on duplicate module name or a
// route pattern collision (surfaced from api.RouteRegistrar.Handle).
func (r *Registry) Register(m Module) error {
	name := m.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("registering module %q: %w", name, errDuplicateModuleName)
	}

	if err := m.RegisterSchema(r.schema); err != nil {
		return fmt.Errorf("registering schema for module %q: %w", name, err)
	}
	if err := m.RegisterRoutes(r.routes); err != nil {
		return fmt.Errorf("registering routes for module %q: %w", name, err)
	}

	r.modules[name] = m
	return nil
}

var errDuplicateModuleName = errors.New("duplicate module name")

// Dispatch routes spans to the Module named by each span's atlas.module
// resource attribute (default "otelcore" if absent). A span naming an
// unregistered module is dropped and logged, not silently ignored.
func (r *Registry) Dispatch(ctx context.Context, spans []storage.Span) error {
	grouped := make(map[string][]storage.Span)
	for _, s := range spans {
		grouped[targetModule(s)] = append(grouped[targetModule(s)], s)
	}

	var errs []error
	for name, group := range grouped {
		m, ok := r.modules[name]
		if !ok {
			slog.ErrorContext(ctx, "dropping spans for unregistered module",
				"module", name, "span_count", len(group))
			continue
		}
		if err := m.HandleSpans(ctx, group); err != nil {
			errs = append(errs, fmt.Errorf("module %q handling spans: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func targetModule(s storage.Span) string {
	if v, ok := s.ResourceAttributes[atlasModuleAttr]; ok {
		if name, ok := v.(string); ok && name != "" {
			return name
		}
	}
	return defaultModuleName
}
