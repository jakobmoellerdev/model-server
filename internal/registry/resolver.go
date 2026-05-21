package registry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	ocmclient "github.com/open-component-model/model-server/internal/ocm"
)

// OCMRegistry implements ModelRegistry backed by an OCM client.
type OCMRegistry struct {
	client   *ocmclient.Client
	idx      *index
	mu       sync.RWMutex
	log      *slog.Logger
	indexTTL time.Duration
	ready    bool
}

// NewOCMRegistry creates a registry and builds the initial model index.
func NewOCMRegistry(client *ocmclient.Client, indexTTL time.Duration, log *slog.Logger) (*OCMRegistry, error) {
	r := &OCMRegistry{
		client:   client,
		idx:      newIndex(),
		log:      log,
		indexTTL: indexTTL,
	}
	if err := r.buildIndex(context.Background()); err != nil {
		return nil, fmt.Errorf("initial index build: %w", err)
	}
	return r, nil
}

func (r *OCMRegistry) Ready() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ready
}

func (r *OCMRegistry) Search(_ context.Context, f SearchFilter) ([]ModelDescriptor, error) {
	return r.idx.search(f), nil
}

func (r *OCMRegistry) Describe(ctx context.Context, modelID, version string) (*ModelDescriptor, error) {
	compName, err := r.resolveComponent(modelID)
	if err != nil {
		return nil, err
	}
	cv, err := r.client.LookupVersion(compName, version)
	if err != nil {
		return nil, fmt.Errorf("lookup version: %w", err)
	}
	defer cv.Close()

	info, err := ocmclient.ExtractInfo(cv, r.log)
	if err != nil {
		return nil, err
	}
	d := infoToDescriptor(info)
	return &d, nil
}

func (r *OCMRegistry) ListFiles(ctx context.Context, modelID, revision string) ([]FileEntry, error) {
	desc, err := r.Describe(ctx, modelID, revision)
	if err != nil {
		return nil, err
	}
	return desc.Files, nil
}

func (r *OCMRegistry) OpenFile(_ context.Context, modelID, revision, path string) (io.ReadCloser, int64, error) {
	compName, err := r.resolveComponent(modelID)
	if err != nil {
		return nil, 0, err
	}
	cv, err := r.client.LookupVersion(compName, revision)
	if err != nil {
		return nil, 0, fmt.Errorf("lookup version: %w", err)
	}
	rc, size, err := ocmclient.OpenResource(cv, path)
	if err != nil {
		cv.Close()
		return nil, 0, err
	}
	return &cvReader{ReadCloser: rc, cv: cv}, size, nil
}

func (r *OCMRegistry) Refresh(ctx context.Context) error {
	return r.buildIndex(ctx)
}

func (r *OCMRegistry) buildIndex(ctx context.Context) error {
	names, err := r.client.ListComponents(ctx)
	if err != nil {
		return err
	}

	idx := newIndex()
	for _, name := range names {
		cv, err := r.client.LookupVersion(name, "")
		if err != nil {
			r.log.Warn("skip component: cannot get version", slog.String("component", name), slog.Any("error", err))
			continue
		}
		info, err := ocmclient.ExtractInfo(cv, r.log)
		cv.Close()
		if err != nil {
			r.log.Warn("skip component: cannot extract info", slog.String("component", name), slog.Any("error", err))
			continue
		}
		d := infoToDescriptor(info)
		idx.add(d)
	}

	r.mu.Lock()
	r.idx = idx
	r.ready = true
	r.mu.Unlock()
	return nil
}

func (r *OCMRegistry) resolveComponent(modelID string) (string, error) {
	if comp := r.idx.resolveModelID(modelID); comp != "" {
		return comp, nil
	}
	// fall back: treat model ID as OCM component name
	if strings.Contains(modelID, "/") || strings.Contains(modelID, ".") {
		return modelID, nil
	}
	return "", fmt.Errorf("model %q not found", modelID)
}

func infoToDescriptor(info *ocmclient.ComponentInfo) ModelDescriptor {
	files := make([]FileEntry, len(info.Files))
	for i, f := range info.Files {
		files[i] = FileEntry{
			Path: f.Path, Size: f.Size, Digest: f.Digest,
			MediaType: f.MediaType, IsLFS: f.IsLFS,
		}
	}
	return ModelDescriptor{
		ID: info.ID, Component: info.Component, Version: info.Version,
		Task: info.Task, Library: info.Library, Family: info.Family,
		License: info.License, Gated: info.Gated, Private: info.Private,
		Signed: info.Signed, Labels: info.Labels, Files: files,
		CreatedAt: info.CreatedAt, ModifiedAt: info.ModifiedAt, Digest: info.Digest,
	}
}

// cvReader closes the ComponentVersionAccess when the inner reader closes.
type cvReader struct {
	io.ReadCloser
	cv interface{ Close() error }
}

func (c *cvReader) Close() error {
	err := c.ReadCloser.Close()
	if e2 := c.cv.Close(); err == nil {
		err = e2
	}
	return err
}
