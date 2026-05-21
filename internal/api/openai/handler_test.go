package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/api/openai"
	"github.com/open-component-model/model-server/internal/registry"
)

// stubRegistry satisfies registry.ModelRegistry for tests.
type stubRegistry struct {
	models []registry.ModelDescriptor
}

func (s *stubRegistry) Search(_ context.Context, f registry.SearchFilter) ([]registry.ModelDescriptor, error) {
	var out []registry.ModelDescriptor
	for _, m := range s.models {
		if f.Query != "" && m.ID != f.Query {
			continue
		}
		out = append(out, m)
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (s *stubRegistry) Describe(_ context.Context, modelID, _ string) (*registry.ModelDescriptor, error) {
	for i, m := range s.models {
		if m.ID == modelID {
			return &s.models[i], nil
		}
	}
	return nil, &notFoundError{modelID}
}

func (s *stubRegistry) ListFiles(_ context.Context, _, _ string) ([]registry.FileEntry, error) {
	return nil, nil
}

func (s *stubRegistry) OpenFile(_ context.Context, _, _, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, &notFoundError{"file"}
}

func (s *stubRegistry) Refresh(_ context.Context) error { return nil }
func (s *stubRegistry) Ready() bool                     { return true }

type notFoundError struct{ name string }

func (e *notFoundError) Error() string { return e.name + " not found" }

func makeRouter(reg registry.ModelRegistry) http.Handler {
	r := chi.NewRouter()
	openai.MountRoutes(r, reg)
	return r
}

var testModels = []registry.ModelDescriptor{
	{
		ID:        "meta-llama/Llama-3-8B",
		CreatedAt: time.Unix(1700000000, 0),
		Labels:    map[string]string{"ext.ocm.software/model-server.library": "transformers"},
	},
	{
		ID:        "mistralai/Mistral-7B",
		CreatedAt: time.Unix(1710000000, 0),
		Labels:    map[string]string{},
	},
}

func TestListModels_ReturnsAll(t *testing.T) {
	reg := &stubRegistry{models: testModels}
	srv := httptest.NewServer(makeRouter(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}

	var list struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Object != "list" {
		t.Fatalf("object %q", list.Object)
	}
	if len(list.Data) != 2 {
		t.Fatalf("want 2 models, got %d", len(list.Data))
	}
	if list.Data[0].Object != "model" {
		t.Fatalf("data[0].object %q", list.Data[0].Object)
	}
	if list.Data[0].OwnedBy != "transformers" {
		t.Errorf("owned_by: want transformers, got %q", list.Data[0].OwnedBy)
	}
}

func TestListModels_Empty(t *testing.T) {
	reg := &stubRegistry{}
	srv := httptest.NewServer(makeRouter(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var list struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 0 {
		t.Fatalf("want 0, got %d", len(list.Data))
	}
}

func TestGetModel_TwoSegment(t *testing.T) {
	reg := &stubRegistry{models: testModels}
	srv := httptest.NewServer(makeRouter(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models/meta-llama/Llama-3-8B")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var m struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m.ID != "meta-llama/Llama-3-8B" {
		t.Errorf("id %q", m.ID)
	}
	if m.Object != "model" {
		t.Errorf("object %q", m.Object)
	}
	if m.Created == 0 {
		t.Error("created should be non-zero")
	}
}

func TestGetModel_NotFound(t *testing.T) {
	reg := &stubRegistry{models: testModels}
	srv := httptest.NewServer(makeRouter(reg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models/no-such/model")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestGetModel_OwnedByFallback(t *testing.T) {
	reg := &stubRegistry{models: testModels}
	srv := httptest.NewServer(makeRouter(reg))
	defer srv.Close()

	// mistralai/Mistral-7B has no library label — owned_by derives from ID prefix
	resp, err := http.Get(srv.URL + "/v1/models/mistralai/Mistral-7B")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var m struct {
		OwnedBy string `json:"owned_by"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m.OwnedBy != "mistralai" {
		t.Errorf("owned_by: want mistralai, got %q", m.OwnedBy)
	}
}
