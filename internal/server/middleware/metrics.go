package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-component-model/model-server/internal/observability"
)

// Metrics returns a middleware that records request count and latency.
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t := time.Now()
			next.ServeHTTP(ww, r)

			api := detectAPI(r.URL.Path)
			observability.RequestTotal.With(prometheus.Labels{
				"api": api, "method": r.Method,
				"path": r.URL.Path, "status": strconv.Itoa(ww.Status()),
			}).Inc()
			observability.RequestDuration.With(prometheus.Labels{
				"api": api, "method": r.Method, "path": r.URL.Path,
			}).Observe(time.Since(t).Seconds())
		})
	}
}

func detectAPI(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/"):
		return "ollama_or_hfhub"
	case strings.HasPrefix(path, "/v1/"):
		return "openai"
	default:
		return "hfhub"
	}
}
