// Package plugin defines the module registry: modules (otelcore, llmagent,
// ...) register their storage schema, HTTP routes, and span handler at
// startup. Ingest dispatches every batch through Registry.Dispatch instead
// of writing to storage directly.
package plugin

import (
	"context"

	"atlas/pkg/api"
	"atlas/pkg/storage"
)

// Module is one Atlas plugin (otelcore, llmagent, ...).
type Module interface {
	Name() string
	RegisterSchema(s storage.SchemaRegistrar) error
	RegisterRoutes(r api.RouteRegistrar) error
	HandleSpans(ctx context.Context, spans []storage.Span) error
}

// defaultModuleName is used when a span carries no atlas.module resource
// attribute (see Registry.Dispatch).
const defaultModuleName = "otelcore"

// atlasModuleAttr is the resource attribute naming a span's owning module.
const atlasModuleAttr = "atlas.module"
