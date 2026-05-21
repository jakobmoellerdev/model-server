package openai

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/registry"
)

// MountRoutes registers all OpenAI-compatible routes on r.
func MountRoutes(r chi.Router, reg registry.ModelRegistry) {
	r.Get("/v1/models", listModels(reg))
	r.Get("/v1/models/{owner}/{model}", getModel(reg))
	r.Get("/v1/models/{model}", getModel(reg))
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

// resolveModelID reconstructs the model ID from URL params, supporting both
// /v1/models/{model} (single segment) and /v1/models/{owner}/{model} (two segments).
func resolveModelID(r *http.Request) string {
	if owner := chi.URLParam(r, "owner"); owner != "" {
		return owner + "/" + chi.URLParam(r, "model")
	}
	return chi.URLParam(r, "model")
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
