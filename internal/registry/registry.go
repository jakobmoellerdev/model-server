package registry

import (
	"context"
	"io"
	"time"
)

// ModelDescriptor is an immutable snapshot of model metadata derived from an OCM component.
type ModelDescriptor struct {
	ID         string            // public model identifier, e.g. "meta-llama/Llama-3-8B"
	Component  string            // OCM component name
	Version    string            // OCM component version
	Task       string            // text-generation, embeddings, etc.
	Library    string            // transformers, diffusers, etc.
	Family     string            // llama, mistral, etc.
	License    string
	Gated      bool
	Private    bool
	Signed     bool
	Labels     map[string]string
	Files      []FileEntry
	CreatedAt  time.Time
	ModifiedAt time.Time
	Digest     string
}

// FileEntry is a single file within a model component version.
type FileEntry struct {
	Path      string // relative path, e.g. "model.safetensors"
	Size      int64
	Digest    string // sha256:...
	MediaType string
	IsLFS     bool // if true, large file — serve via redirect
}

// SearchFilter controls which models Search returns.
type SearchFilter struct {
	Query  string
	Task   string
	Tags   []string
	Sort   string // "modified" (default) | "id"
	Limit  int
	Offset int
}

// ModelRegistry is the primary interface for model discovery and file access.
type ModelRegistry interface {
	Search(ctx context.Context, f SearchFilter) ([]ModelDescriptor, error)
	Describe(ctx context.Context, modelID, version string) (*ModelDescriptor, error)
	ListFiles(ctx context.Context, modelID, revision string) ([]FileEntry, error)
	// OpenFile opens a named file for streaming. Caller must close the reader.
	OpenFile(ctx context.Context, modelID, revision, path string) (io.ReadCloser, int64, error)
	Refresh(ctx context.Context) error
	Ready() bool
}
