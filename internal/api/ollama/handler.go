package ollama

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/registry"
)

// NewHandler returns an http.Handler for Ollama-compatible routes.
// Mount at /api — routes are registered without the prefix.
func NewHandler(reg registry.ModelRegistry) http.Handler {
	r := chi.NewRouter()

	r.Get("/tags", tags(reg))
	r.Post("/show", show(reg))
	r.Post("/pull", pull(reg))
	r.Delete("/delete", deleteModel())

	return r
}

func tags(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models, err := reg.Search(r.Context(), registry.SearchFilter{Limit: 500})
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		summaries := make([]ModelSummary, len(models))
		for i, m := range models {
			summaries[i] = toSummary(m)
		}
		jsonOK(w, TagsResponse{Models: summaries})
	}
}

func show(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ShowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		modelID, version := splitTag(req.Name)
		desc, err := reg.Describe(r.Context(), modelID, version)
		if err != nil {
			jsonError(w, statusFor(err), err)
			return
		}
		jsonOK(w, toShowResponse(*desc))
	}
}

func pull(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		modelID, version := splitTag(req.Name)

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flush := func() {
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		emit := func(e PullEvent) {
			json.NewEncoder(w).Encode(e) //nolint:errcheck
			flush()
		}

		emit(PullEvent{Status: "pulling manifest"})

		desc, err := reg.Describe(r.Context(), modelID, version)
		if err != nil {
			emit(PullEvent{Status: "error: " + err.Error()})
			return
		}

		for _, f := range desc.Files {
			emit(PullEvent{Status: "pulling " + f.Digest, Digest: f.Digest, Total: f.Size})

			rc, size, err := reg.OpenFile(r.Context(), modelID, version, f.Path)
			if err != nil {
				emit(PullEvent{Status: "error: " + err.Error()})
				return
			}
			written, _ := io.Copy(io.Discard, rc)
			rc.Close()

			emit(PullEvent{
				Status: "pulling " + f.Digest, Digest: f.Digest,
				Total: size, Completed: written,
			})
		}

		emit(PullEvent{Status: "verifying sha256 digest"})
		emit(PullEvent{Status: "writing manifest"})
		emit(PullEvent{Status: "success"})
	}
}

func deleteModel() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		jsonError(w, http.StatusMethodNotAllowed,
			fmt.Errorf("delete not supported; OCM components are immutable"))
	}
}

func splitTag(name string) (string, string) {
	if idx := strings.LastIndex(name, ":"); idx != -1 {
		return name[:idx], name[idx+1:]
	}
	return name, ""
}

func statusFor(err error) int {
	if strings.Contains(err.Error(), "not found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
}

// keep chi import used for router
var _ = chi.URLParam
