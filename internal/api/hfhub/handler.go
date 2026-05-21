package hfhub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/registry"
)

// MountRoutes registers all HF Hub-compatible routes on the given router.
// Routes are registered directly to avoid conflicts with /api prefix used by Ollama.
func MountRoutes(r chi.Router, reg registry.ModelRegistry) {
	// Model discovery
	r.Get("/api/models", listModels(reg))
	r.Get("/api/models/{owner}/{model}/tree/{revision}", fileTree(reg))
	r.Get("/api/models/{owner}/{model}", modelInfoOwner(reg))
	r.Get("/api/models/{model}", modelInfoSingle(reg))

	// File downloads
	r.Get("/{owner}/{model}/resolve/{revision}/*", downloadFile(reg))
	r.Get("/{owner}/{model}/raw/{revision}/*", downloadFile(reg))
}

// NewHandler returns an http.Handler for all HF Hub-compatible routes.
// Deprecated: use MountRoutes to avoid /api prefix conflicts with Ollama.
func NewHandler(reg registry.ModelRegistry) http.Handler {
	r := chi.NewRouter()
	MountRoutes(r, reg)
	return r
}

func listModels(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		offset, _ := strconv.Atoi(q.Get("skip"))
		if limit == 0 {
			limit = 100
		}
		models, err := reg.Search(r.Context(), registry.SearchFilter{
			Query: q.Get("search"), Task: q.Get("task"),
			Limit: limit, Offset: offset,
		})
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		infos := make([]ModelInfo, len(models))
		for i, m := range models {
			infos[i] = toModelInfo(m)
		}
		jsonOK(w, infos)
	}
}

func modelInfoOwner(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveModelInfo(w, r, reg, chi.URLParam(r, "owner")+"/"+chi.URLParam(r, "model"))
	}
}

func modelInfoSingle(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveModelInfo(w, r, reg, chi.URLParam(r, "model"))
	}
}

func serveModelInfo(w http.ResponseWriter, r *http.Request, reg registry.ModelRegistry, modelID string) {
	desc, err := reg.Describe(r.Context(), modelID, "")
	if err != nil {
		jsonError(w, statusFor(err), err)
		return
	}
	jsonOK(w, toModelInfo(*desc))
}

func fileTree(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modelID := chi.URLParam(r, "owner") + "/" + chi.URLParam(r, "model")
		revision := chi.URLParam(r, "revision")

		files, err := reg.ListFiles(r.Context(), modelID, revision)
		if err != nil {
			jsonError(w, statusFor(err), err)
			return
		}
		entries := make([]TreeEntry, len(files))
		for i, f := range files {
			entries[i] = TreeEntry{Type: "blob", Path: f.Path, Size: f.Size, BlobID: f.Digest}
		}
		jsonOK(w, entries)
	}
}

func downloadFile(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modelID := chi.URLParam(r, "owner") + "/" + chi.URLParam(r, "model")
		revision := chi.URLParam(r, "revision")
		filePath := chi.URLParam(r, "*")

		rc, size, err := reg.OpenFile(r.Context(), modelID, revision, filePath)
		if err != nil {
			jsonError(w, statusFor(err), err)
			return
		}
		defer rc.Close()

		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filePath))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, rc) //nolint:errcheck
	}
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
