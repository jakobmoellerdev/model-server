package mlflow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/registry"
)

// MountRoutes registers all MLflow Model Registry read-only routes on r.
// Prefix: /api/2.0/mlflow
func MountRoutes(r chi.Router, reg registry.ModelRegistry) {
	r.Route("/api/2.0/mlflow", func(r chi.Router) {
		// Registered models
		r.Get("/registered-models/search", searchRegisteredModels(reg))
		r.Get("/registered-models/get", getRegisteredModel(reg))

		// Model versions
		r.Get("/model-versions/search", searchModelVersions(reg))
		r.Get("/model-versions/get", getModelVersion(reg))
		r.Get("/model-versions/get-download-uri", getModelVersionDownloadURI(reg))
	})
}

// searchRegisteredModels handles GET /api/2.0/mlflow/registered-models/search
// Query params: filter (ignored), max_results, page_token (ignored)
func searchRegisteredModels(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("max_results"))
		if limit == 0 {
			limit = 100
		}

		models, err := reg.Search(r.Context(), registry.SearchFilter{
			Query: filterToQuery(q.Get("filter")),
			Limit: limit,
		})
		if err != nil {
			mlflowError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err)
			return
		}

		result := make([]RegisteredModel, len(models))
		for i, m := range models {
			result[i] = toRegisteredModel(m)
		}
		jsonOK(w, SearchRegisteredModelsResponse{RegisteredModels: result})
	}
}

// getRegisteredModel handles GET /api/2.0/mlflow/registered-models/get?name=<name>
func getRegisteredModel(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			mlflowError(w, http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
				fmt.Errorf("name parameter is required"))
			return
		}
		desc, err := reg.Describe(r.Context(), name, "")
		if err != nil {
			mlflowError(w, statusFor(err), mlflowCode(err), err)
			return
		}
		jsonOK(w, GetRegisteredModelResponse{RegisteredModel: toRegisteredModel(*desc)})
	}
}

// searchModelVersions handles GET /api/2.0/mlflow/model-versions/search
// Query params: filter (name=<name> parsed), max_results
func searchModelVersions(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("max_results"))
		if limit == 0 {
			limit = 100
		}

		// filter may be e.g. "name='org/model'" — extract model name if present
		modelName := filterToQuery(q.Get("filter"))

		models, err := reg.Search(r.Context(), registry.SearchFilter{
			Query: modelName,
			Limit: limit,
		})
		if err != nil {
			mlflowError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err)
			return
		}

		var versions []ModelVersion
		for _, m := range models {
			versions = append(versions, toModelVersion(m))
		}
		if versions == nil {
			versions = []ModelVersion{}
		}
		jsonOK(w, SearchModelVersionsResponse{ModelVersions: versions})
	}
}

// getModelVersion handles GET /api/2.0/mlflow/model-versions/get?name=<name>&version=<version>
func getModelVersion(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		name := q.Get("name")
		version := q.Get("version")
		if name == "" {
			mlflowError(w, http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
				fmt.Errorf("name parameter is required"))
			return
		}
		desc, err := reg.Describe(r.Context(), name, version)
		if err != nil {
			mlflowError(w, statusFor(err), mlflowCode(err), err)
			return
		}
		jsonOK(w, GetModelVersionResponse{ModelVersion: toModelVersion(*desc)})
	}
}

// getModelVersionDownloadURI handles
// GET /api/2.0/mlflow/model-versions/get-download-uri?name=<name>&version=<version>
// Returns a URI pointing at the HF-Hub-compatible resolve endpoint so existing
// download logic is reused without duplicating blob streaming.
func getModelVersionDownloadURI(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		name := q.Get("name")
		version := q.Get("version")
		if name == "" {
			mlflowError(w, http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
				fmt.Errorf("name parameter is required"))
			return
		}
		// Verify the model exists
		desc, err := reg.Describe(r.Context(), name, version)
		if err != nil {
			mlflowError(w, statusFor(err), mlflowCode(err), err)
			return
		}
		ref := desc.Version
		if ref == "" {
			ref = "main"
		}
		// Build a base URI; callers append the specific filename.
		// e.g. <base>/<owner>/<model>/resolve/<version>/
		parts := strings.SplitN(name, "/", 2)
		var uri string
		if len(parts) == 2 {
			uri = fmt.Sprintf("%s/%s/%s/resolve/%s/",
				baseURL(r), parts[0], parts[1], ref)
		} else {
			uri = fmt.Sprintf("%s/%s/resolve/%s/",
				baseURL(r), name, ref)
		}
		jsonOK(w, GetModelVersionDownloadURIResponse{ArtifactURI: uri})
	}
}

// baseURL reconstructs the scheme+host from the incoming request.
func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	return scheme + "://" + host
}

// filterToQuery does minimal parsing of MLflow filter expressions.
// "name = 'foo'" or "name LIKE '%llama%'" → extracts the value for use as a search query.
func filterToQuery(filter string) string {
	if filter == "" {
		return ""
	}
	// strip quotes and extract RHS after first operator
	for _, op := range []string{" LIKE ", " like ", " = ", " == "} {
		if idx := strings.Index(filter, op); idx != -1 {
			val := strings.TrimSpace(filter[idx+len(op):])
			val = strings.Trim(val, "'\"")
			val = strings.Trim(val, "%")
			return val
		}
	}
	return ""
}

func statusFor(err error) int {
	if strings.Contains(err.Error(), "not found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func mlflowCode(err error) string {
	if strings.Contains(err.Error(), "not found") {
		return "RESOURCE_DOES_NOT_EXIST"
	}
	return "INTERNAL_ERROR"
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func mlflowError(w http.ResponseWriter, status int, code string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{ //nolint:errcheck
		ErrorCode: code,
		Message:   err.Error(),
	})
}

// keep chi import used
var _ = chi.URLParam
