package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/open-component-model/model-server/internal/api/health"
	"github.com/open-component-model/model-server/internal/api/hfhub"
	"github.com/open-component-model/model-server/internal/api/mlflow"
	"github.com/open-component-model/model-server/internal/api/ollama"
	"github.com/open-component-model/model-server/internal/api/openai"
	"github.com/open-component-model/model-server/internal/config"
	"github.com/open-component-model/model-server/internal/registry"
	"github.com/open-component-model/model-server/internal/server/middleware"
)

// NewRouter assembles the chi router with all API surfaces mounted.
func NewRouter(cfg *config.Config, reg registry.ModelRegistry) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Logger())
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.Metrics())

	if cfg.Auth.Mode != "none" {
		r.Use(middleware.Auth(cfg.Auth))
	}

	r.Get("/healthz", health.Liveness)
	r.Head("/healthz", health.Liveness)
	r.Get("/readyz", health.Readiness(reg))
	r.Handle("/metrics", promhttp.Handler())

	// HEAD / — used by ollama CLI as a connectivity probe
	r.Head("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	if cfg.APIs.HFHub.Enabled {
		hfhub.MountRoutes(r, reg)
	}

	if cfg.APIs.Ollama.Enabled {
		r.Mount("/api", ollama.NewHandler(reg))
	}

	if cfg.APIs.OpenAI.Enabled {
		openai.MountRoutes(r, reg)
	}

	if cfg.APIs.MLflow.Enabled {
		mlflow.MountRoutes(r, reg)
	}

	return r
}
