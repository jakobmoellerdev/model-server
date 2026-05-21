package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-component-model/model-server/internal/config"
	ocmclient "github.com/open-component-model/model-server/internal/ocm"
	"github.com/open-component-model/model-server/internal/observability"
	"github.com/open-component-model/model-server/internal/registry"
	"github.com/open-component-model/model-server/internal/server"
)

// version is injected at build time via -ldflags "-X main.version=v0.1.0".
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configPath = flag.String("config", "model-server.yaml", "path to config file")
		logLevel   = flag.String("log-level", "info", "log level: debug|info|warn|error")
		logFormat  = flag.String("log-format", "json", "log format: json|text")
	)
	flag.Parse()

	log := observability.NewLogger(*logLevel, *logFormat)
	slog.SetDefault(log)
	log.Info("starting model-server", "version", version)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	reg := prometheus.NewRegistry()
	observability.RegisterMetrics(reg)

	ocmClient, err := ocmclient.NewClient(cfg.OCM, cfg.Credentials, log)
	if err != nil {
		return fmt.Errorf("create OCM client: %w", err)
	}

	modelReg, err := registry.NewOCMRegistry(ocmClient, cfg.OCM.IndexTTL.Duration, log)
	if err != nil {
		return fmt.Errorf("create registry: %w", err)
	}

	// Background index refresh
	go func() {
		ticker := time.NewTicker(cfg.OCM.RefreshInterval.Duration)
		defer ticker.Stop()
		for range ticker.C {
			if err := modelReg.Refresh(context.Background()); err != nil {
				log.Warn("registry refresh failed", slog.Any("error", err))
			}
		}
	}()

	handler := server.NewRouter(cfg, modelReg)
	srv := server.New(cfg.Server, handler, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	select {
	case <-ctx.Done():
		return srv.Shutdown(cfg.Server.ShutdownTimeout.Duration)
	case err := <-errCh:
		return err
	}
}
