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
	// Model discovery — wildcard captures multi-segment model IDs
	r.Get("/api/models", listModels(reg))
	r.Get("/api/models/*", modelInfoOrTree(reg))

	// File downloads — catch-all; handler splits on /resolve/ or /raw/ to extract
	// the model ID, which may itself contain slashes.
	dl := downloadFileWild(reg)
	r.Get("/*", dl)
	r.Head("/*", dl)
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

func modelInfoOrTree(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wild := chi.URLParam(r, "*") // everything after /api/models/
		// detect .../tree/{revision}
		if idx := strings.Index(wild, "/tree/"); idx != -1 {
			modelID := wild[:idx]
			revision := wild[idx+len("/tree/"):]
			files, err := reg.ListFiles(r.Context(), modelID, revision)
			if err != nil {
				jsonError(w, statusFor(err), err)
				return
			}
			entries := make([]TreeEntry, len(files))
			for i, f := range files {
				entries[i] = TreeEntry{Type: "file", Path: f.Path, Size: f.Size, BlobID: f.Digest}
			}
			jsonOK(w, entries)
			return
		}
		serveModelInfo(w, r, reg, wild)
	}
}

// downloadFileWild extracts the model ID from the raw request path by splitting on
// "/resolve/" or "/raw/", since the model ID may contain slashes.
func downloadFileWild(reg registry.ModelRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path // e.g. /github.com/org/repo/resolve/main/config.json

		var modelID, revision, filePath string
		for _, sep := range []string{"/resolve/", "/raw/"} {
			if idx := strings.Index(path, sep); idx != -1 {
				modelID = strings.TrimPrefix(path[:idx], "/")
				rest := path[idx+len(sep):]
				if slash := strings.Index(rest, "/"); slash != -1 {
					revision = rest[:slash]
					filePath = rest[slash+1:]
				} else {
					revision = rest
				}
				break
			}
		}
		if modelID == "" {
			http.NotFound(w, r)
			return
		}

		desc, err := reg.Describe(r.Context(), modelID, revision)
		if err != nil {
			jsonError(w, statusFor(err), err)
			return
		}

		rc, size, err := reg.OpenFile(r.Context(), modelID, revision, filePath)
		if err != nil {
			jsonError(w, statusFor(err), err)
			return
		}
		defer rc.Close() //nolint:errcheck

		var fileDigest string
		for _, f := range desc.Files {
			if f.Path == filePath {
				fileDigest = f.Digest
				break
			}
		}

		w.Header().Set("X-Repo-Commit", desc.Version)
		if fileDigest != "" {
			w.Header().Set("ETag", `"`+fileDigest+`"`)
		}
		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filePath))
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			io.Copy(w, rc) //nolint:errcheck
		}
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
