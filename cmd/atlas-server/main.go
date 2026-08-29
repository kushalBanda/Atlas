// Command atlas-server is the Atlas backend entrypoint.
//
// See docs/plans/atlas/04-slices.md for the build order and
// docs/plans/atlas/03-program-design.md for the target startup call stack.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atlas/pkg/api"
	"atlas/pkg/config"
	"atlas/pkg/discovery"
	"atlas/pkg/ingest"
	"atlas/pkg/plugin"
	"atlas/pkg/plugin/llmagent"
	"atlas/pkg/plugin/otelcore"
	"atlas/pkg/query"
	"atlas/pkg/rootcause"
	"atlas/pkg/storage"
)

const (
	defaultConfigPath = "./conf/example.yaml"
	defaultStaticDir  = "./frontend/dist"
	shutdownTimeout   = 10 * time.Second

	// Tick intervals aren't part of pkg/config (only thresholds/paths are,
	// per docs/plans/atlas/03-program-design.md) — internal implementation
	// detail, not something an operator needs to tune in v1.
	rootCauseTick = 5 * time.Second
	discoveryTick = 30 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("atlas-server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := flag.String("config", defaultConfigPath, "path to config file")
	staticDir := flag.String("static-dir", defaultStaticDir, "path to built frontend assets; empty or missing disables static serving")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	store, err := storage.NewDuckDB(cfg.StoragePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Error("closing storage failed", "error", err)
		}
	}()

	queryHandlers := query.NewHandlers(store)
	apiMux := api.NewRouter(queryHandlers, *staticDir)

	registry := plugin.NewRegistry(store, apiMux)
	if err := registry.Register(otelcore.New(store)); err != nil {
		return fmt.Errorf("registering otelcore module: %w", err)
	}
	if err := registry.Register(llmagent.New(store)); err != nil {
		return fmt.Errorf("registering llmagent module: %w", err)
	}

	discoveryRules, err := discovery.LoadRules(cfg.DiscoveryRulesPath)
	if err != nil {
		slog.Warn("loading discovery rules failed, continuing with no curated rules", "error", err, "path", cfg.DiscoveryRulesPath)
	}
	discoveryHandler := discovery.NewHandler([]discovery.Discoverer{
		discovery.NewProcessScanDiscoverer(discoveryRules),
		discovery.NewDockerDiscoverer(discoveryRules),
		discovery.NewK8sDiscoverer(discoveryRules),
	})
	if err := apiMux.Handle("GET /discovery/targets", http.HandlerFunc(discoveryHandler.ServeTargets)); err != nil {
		return fmt.Errorf("registering discovery targets route: %w", err)
	}

	ingestSrv := ingest.NewServer(registry)
	ingestMux := http.NewServeMux()
	ingestMux.HandleFunc("POST /v1/traces", ingestSrv.ServeOTLP)

	ingestHTTP := &http.Server{Addr: cfg.IngestAddr, Handler: ingestMux}
	apiHTTP := &http.Server{Addr: cfg.APIAddr, Handler: apiMux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	watcher := rootcause.NewWatcher(store, cfg.TraceCloseTimeout, cfg.RootCauseSelfTimePct)
	go supervise(ctx, "rootcause-watcher", func(ctx context.Context) error {
		return watcher.Run(ctx, rootCauseTick)
	})
	go supervise(ctx, "discovery", func(ctx context.Context) error {
		return discoveryHandler.Run(ctx, discoveryTick)
	})

	errCh := make(chan error, 2)
	go func() { errCh <- serve(ingestHTTP, "ingest") }()
	go func() { errCh <- serve(apiHTTP, "api") }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var shutdownErrs []error
	if err := ingestHTTP.Shutdown(shutdownCtx); err != nil {
		shutdownErrs = append(shutdownErrs, err)
	}
	if err := apiHTTP.Shutdown(shutdownCtx); err != nil {
		shutdownErrs = append(shutdownErrs, err)
	}
	return errors.Join(shutdownErrs...)
}

// loadConfig loads path if it exists; a missing file falls back to
// config.Default() (dev-friendly first run), but a file that exists and
// fails to parse or validate is a real error, not silently ignored.
func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		slog.Warn("config file not found, using defaults", "path", path)
		cfg := config.Default()
		return &cfg, nil
	}
	return config.Load(path)
}

func serve(srv *http.Server, name string) error {
	slog.Info("listening", "server", name, "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
