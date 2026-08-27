package plugin

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/api"
	"atlas/pkg/storage"
)

// fakeModule is a minimal Module used only to test Registry behavior in
// isolation from any real module (otelcore, llmagent, ...).
type fakeModule struct {
	name         string
	schemaErr    error
	routesErr    error
	handleErr    error
	routePattern string
	handled      []storage.Span
}

func (m *fakeModule) Name() string { return m.name }

func (m *fakeModule) RegisterSchema(_ storage.SchemaRegistrar) error { return m.schemaErr }

func (m *fakeModule) RegisterRoutes(r api.RouteRegistrar) error {
	if m.routesErr != nil {
		return m.routesErr
	}
	if m.routePattern == "" {
		return nil
	}
	return r.Handle(m.routePattern, http.NotFoundHandler())
}

func (m *fakeModule) HandleSpans(_ context.Context, spans []storage.Span) error {
	if m.handleErr != nil {
		return m.handleErr
	}
	m.handled = append(m.handled, spans...)
	return nil
}

// fakeRouteRegistrar is a minimal api.RouteRegistrar used to test route
// collisions between two modules without a real api.Router.
type fakeRouteRegistrar struct {
	patterns map[string]bool
}

func newFakeRouteRegistrar() *fakeRouteRegistrar {
	return &fakeRouteRegistrar{patterns: make(map[string]bool)}
}

var errFakeRouteCollision = errors.New("route already registered")

func (f *fakeRouteRegistrar) Handle(pattern string, _ http.Handler) error {
	if f.patterns[pattern] {
		return errFakeRouteCollision
	}
	f.patterns[pattern] = true
	return nil
}

type noopSchemaRegistrar struct{}

func (noopSchemaRegistrar) CreateTable(_ string) error { return nil }

var errAssertAnError = errors.New("assert.AnError general error for testing")

func TestRegister_DuplicateNameErrors(t *testing.T) {
	t.Parallel()
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())

	require.NoError(t, reg.Register(&fakeModule{name: "otelcore"}))
	err := reg.Register(&fakeModule{name: "otelcore"})
	require.ErrorIs(t, err, errDuplicateModuleName)
}

func TestRegister_SchemaErrorPropagates(t *testing.T) {
	t.Parallel()
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())

	err := reg.Register(&fakeModule{name: "broken", schemaErr: errAssertAnError})
	require.Error(t, err)
}

func TestRegister_RouteCollisionErrors(t *testing.T) {
	t.Parallel()
	router := newFakeRouteRegistrar()
	reg := NewRegistry(noopSchemaRegistrar{}, router)

	require.NoError(t, reg.Register(&fakeModule{name: "a", routePattern: "GET /shared"}))
	err := reg.Register(&fakeModule{name: "b", routePattern: "GET /shared"})
	require.Error(t, err)
}

func TestDispatch_RoutesToCorrectModule(t *testing.T) {
	t.Parallel()
	otelcoreMod := &fakeModule{name: "otelcore"}
	llmMod := &fakeModule{name: "llmagent"}
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())
	require.NoError(t, reg.Register(otelcoreMod))
	require.NoError(t, reg.Register(llmMod))

	spans := []storage.Span{
		{SpanID: "s1", ResourceAttributes: map[string]any{"atlas.module": "llmagent"}},
		{SpanID: "s2", ResourceAttributes: map[string]any{"atlas.module": "otelcore"}},
	}

	require.NoError(t, reg.Dispatch(context.Background(), spans))
	require.Len(t, llmMod.handled, 1)
	require.Equal(t, "s1", llmMod.handled[0].SpanID)
	require.Len(t, otelcoreMod.handled, 1)
	require.Equal(t, "s2", otelcoreMod.handled[0].SpanID)
}

func TestDispatch_UnclaimedSpanDefaultsToOtelcore(t *testing.T) {
	t.Parallel()
	otelcoreMod := &fakeModule{name: "otelcore"}
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())
	require.NoError(t, reg.Register(otelcoreMod))

	spans := []storage.Span{{SpanID: "no-module-attr"}}
	require.NoError(t, reg.Dispatch(context.Background(), spans))
	require.Len(t, otelcoreMod.handled, 1)
}

func TestDispatch_UnregisteredModuleAttributeLogsAndDrops(t *testing.T) {
	t.Parallel()
	otelcoreMod := &fakeModule{name: "otelcore"}
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())
	require.NoError(t, reg.Register(otelcoreMod))

	spans := []storage.Span{
		{SpanID: "dropped", ResourceAttributes: map[string]any{"atlas.module": "unregistered-module"}},
	}

	err := reg.Dispatch(context.Background(), spans)
	require.NoError(t, err) // dropped + logged, not an error
	require.Empty(t, otelcoreMod.handled)
}

func TestDispatch_ModuleHandleErrorPropagates(t *testing.T) {
	t.Parallel()
	otelcoreMod := &fakeModule{name: "otelcore", handleErr: errAssertAnError}
	reg := NewRegistry(noopSchemaRegistrar{}, newFakeRouteRegistrar())
	require.NoError(t, reg.Register(otelcoreMod))

	err := reg.Dispatch(context.Background(), []storage.Span{{SpanID: "s1"}})
	require.Error(t, err)
}
