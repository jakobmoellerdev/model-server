package openai

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/registry"
)

// MountRoutes registers all OpenAI-compatible routes on r.
func MountRoutes(r chi.Router, reg registry.ModelRegistry) {
	r.Get("/v1/models", listModels(reg))
	r.Get("/v1/models/*", getModel(reg))
}

func listModels(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models, err := reg.Search(r.Context(), registry.SearchFilter{Limit: 1000})
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		data := make([]ModelObject, len(models))
		for i, m := range models {
			data[i] = toModelObject(m)
		}
		jsonOK(w, ModelList{Object: "list", Data: data})
	}
}

func getModel(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modelID := resolveModelID(r)
		desc, err := reg.Describe(r.Context(), modelID, "")
		if err != nil {
			jsonError(w, statusFor(err), err)
			return
		}
		jsonOK(w, toModelObject(*desc))
	}
}

// resolveModelID extracts the model ID from the wildcard path param.
// Also handles URL-encoded slashes (e.g. "owner%2Fmodel").
func resolveModelID(r *http.Request) string {
	raw := chi.URLParam(r, "*")
	if decoded, err := url.QueryUnescape(raw); err == nil {
		return decoded
	}
	return raw
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
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"error": map[string]string{
			"message": err.Error(),
			"type":    "invalid_request_error",
		},
	})
}
