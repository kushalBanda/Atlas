package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	router := NewRouter(query.NewHandlers(&fakeStore{}), "")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_Handle_CollisionErrors(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}), "")

	err := router.Handle("GET /healthz", http.NotFoundHandler())
	require.Error(t, err)
}

func TestRouter_Handle_NewPatternSucceeds(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}), "")

	err := router.Handle("GET /plugins/example", http.NotFoundHandler())
	require.NoError(t, err)
}

func TestNewRouter_StaticDirEmpty_NoStaticRouteRegistered(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}), "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_StaticDirMissing_WarnsAndSkipsRoute(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}), filepath.Join(t.TempDir(), "does-not-exist"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_StaticDirSet_ServesIndexHTML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>atlas</html>"), 0o644))

	router := NewRouter(query.NewHandlers(&fakeStore{}), dir)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "<html>atlas</html>", w.Body.String())
}

func TestNewRouter_StaticRoute_DoesNotShadowAPIRoutes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>atlas</html>"), 0o644))

	router := NewRouter(query.NewHandlers(&fakeStore{}), dir)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEqual(t, "<html>atlas</html>", w.Body.String())
}
