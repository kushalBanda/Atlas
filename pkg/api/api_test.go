package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"atlas/pkg/query"
	"atlas/pkg/storage"
)

type fakeStore struct {
	storage.Store
}

func TestNewRouter_HealthzRespondsOK(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_Handle_CollisionErrors(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}))

	err := router.Handle("GET /healthz", http.NotFoundHandler())
	require.Error(t, err)
}

func TestRouter_Handle_NewPatternSucceeds(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}))

	err := router.Handle("GET /plugins/example", http.NotFoundHandler())
	require.NoError(t, err)
}
