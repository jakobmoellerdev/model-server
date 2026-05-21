package health

import (
	"encoding/json"
	"net/http"

	"github.com/open-component-model/model-server/internal/registry"
)

func Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}

func Readiness(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if !reg.Ready() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"}) //nolint:errcheck
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"}) //nolint:errcheck
	}
}
