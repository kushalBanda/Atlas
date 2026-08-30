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
	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), "")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_Handle_CollisionErrors(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), "")

	err := router.Handle("GET /healthz", http.NotFoundHandler())
	require.Error(t, err)
}

func TestRouter_Handle_NewPatternSucceeds(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), "")

	err := router.Handle("GET /plugins/example", http.NotFoundHandler())
	require.NoError(t, err)
}

func TestNewRouter_StaticDirEmpty_NoStaticRouteRegistered(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_StaticDirMissing_WarnsAndSkipsRoute(t *testing.T) {
	t.Parallel()
	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), filepath.Join(t.TempDir(), "does-not-exist"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_StaticDirSet_ServesIndexHTML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>atlas</html>"), 0o644))

	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), dir)

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

	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), dir)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEqual(t, "<html>atlas</html>", w.Body.String())
}

// TestNewRouter_SPAShell_NotCacheableAcrossAccept proves the SPA-shell
// response for a path like /traces/{id} carries Cache-Control: no-store and
// Vary: Accept. A browser hard-navigation and a later fetch() both hit
// this exact URL with different Accept headers; without these, the
// html-shell response (cacheable via http.ServeFile's Last-Modified) can
// get replayed from the browser's cache for the fetch() call too, so the
// JSON API is never actually reached — see the ServeHTTP comment.
func TestNewRouter_SPAShell_NotCacheableAcrossAccept(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>atlas</html>"), 0o644))

	router := NewRouter(query.NewHandlers(&fakeStore{}, 3), dir)

	req := httptest.NewRequest(http.MethodGet, "/traces/abc123", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "<html>atlas</html>", w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "Accept", w.Header().Get("Vary"))
}
