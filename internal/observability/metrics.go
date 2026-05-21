package observability

import (
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics registered by RegisterMetrics.
var (
	RequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "modelserver_requests_total",
			Help: "Total HTTP requests by API, method, path and status.",
		},
		[]string{"api", "method", "path", "status"},
	)
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "modelserver_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api", "method", "path"},
	)
	IndexSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "modelserver_index_model_count",
		Help: "Number of models in the registry index.",
	})
)

// RegisterMetrics registers all metrics with reg.
func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(RequestTotal, RequestDuration, IndexSize)
}

// NewLogger returns a structured logger for the given level and format.
func NewLogger(level, format string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	if format == "text" {
		return slog.New(slog.NewTextHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}
