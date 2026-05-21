package mlflow_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/open-component-model/model-server/internal/api/mlflow"
	"github.com/open-component-model/model-server/internal/registry"
)

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
	mlflow.MountRoutes(r, reg)
	return r
}

var testModels = []registry.ModelDescriptor{
	{
		ID:         "meta-llama/Llama-3-8B",
		Version:    "1.0.0",
		Family:     "llama",
		Task:       "text-generation",
		License:    "llama3",
		CreatedAt:  time.Unix(1700000000, 0),
		ModifiedAt: time.Unix(1700000000, 0),
		Labels:     map[string]string{"task": "text-generation"},
	},
	{
		ID:         "mistralai/Mistral-7B",
		Version:    "2",
		Family:     "mistral",
		Task:       "text-generation",
		CreatedAt:  time.Unix(1710000000, 0),
		ModifiedAt: time.Unix(1710000000, 0),
		Labels:     map[string]string{},
	},
}

func TestSearchRegisteredModels_All(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/registered-models/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		RegisteredModels []struct {
			Name string `json:"name"`
		} `json:"registered_models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.RegisteredModels) != 2 {
		t.Fatalf("want 2, got %d", len(body.RegisteredModels))
	}
}

func TestSearchRegisteredModels_FilterByName(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/registered-models/search?filter=name+%3D+%27meta-llama%2FLlama-3-8B%27")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var body struct {
		RegisteredModels []struct {
			Name string `json:"name"`
		} `json:"registered_models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.RegisteredModels) != 1 {
		t.Fatalf("want 1, got %d", len(body.RegisteredModels))
	}
	if body.RegisteredModels[0].Name != "meta-llama/Llama-3-8B" {
		t.Errorf("name %q", body.RegisteredModels[0].Name)
	}
}

func TestGetRegisteredModel_OK(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/registered-models/get?name=meta-llama/Llama-3-8B")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		RegisteredModel struct {
			Name          string `json:"name"`
			LatestVersions []struct {
				Version string `json:"version"`
			} `json:"latest_versions"`
		} `json:"registered_model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.RegisteredModel.Name != "meta-llama/Llama-3-8B" {
		t.Errorf("name %q", body.RegisteredModel.Name)
	}
	if len(body.RegisteredModel.LatestVersions) != 1 {
		t.Errorf("want 1 latest version, got %d", len(body.RegisteredModel.LatestVersions))
	}
}

func TestGetRegisteredModel_MissingName(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/registered-models/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestGetRegisteredModel_NotFound(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/registered-models/get?name=no/such")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSearchModelVersions_All(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/model-versions/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var body struct {
		ModelVersions []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Status  string `json:"status"`
		} `json:"model_versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.ModelVersions) != 2 {
		t.Fatalf("want 2, got %d", len(body.ModelVersions))
	}
	if body.ModelVersions[0].Status != "READY" {
		t.Errorf("status %q", body.ModelVersions[0].Status)
	}
}

func TestGetModelVersion_OK(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/model-versions/get?name=meta-llama/Llama-3-8B&version=1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		ModelVersion struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"model_version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.ModelVersion.Version != "1.0.0" {
		t.Errorf("version %q", body.ModelVersion.Version)
	}
}

func TestGetModelVersionDownloadURI(t *testing.T) {
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/2.0/mlflow/model-versions/get-download-uri?name=meta-llama/Llama-3-8B&version=1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		ArtifactURI string `json:"artifact_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.ArtifactURI, "meta-llama/Llama-3-8B") {
		t.Errorf("uri %q doesn't contain model name", body.ArtifactURI)
	}
	if !strings.Contains(body.ArtifactURI, "/resolve/") {
		t.Errorf("uri %q doesn't contain /resolve/", body.ArtifactURI)
	}
}

func TestFilterToQuery_Variations(t *testing.T) {
	// Test via SearchModelVersions endpoint which uses filterToQuery internally
	srv := httptest.NewServer(makeRouter(&stubRegistry{models: testModels}))
	defer srv.Close()

	cases := []struct {
		filter string
		wantN  int
	}{
		{`name = 'meta-llama/Llama-3-8B'`, 1},
		{`name LIKE 'meta-llama/Llama-3-8B'`, 1},
		{`name == 'meta-llama/Llama-3-8B'`, 1},
		{``, 2},
	}

	for _, tc := range cases {
		reqURL := srv.URL + "/api/2.0/mlflow/model-versions/search"
		if tc.filter != "" {
			req, _ := http.NewRequest(http.MethodGet, reqURL, nil)
			q := req.URL.Query()
			q.Set("filter", tc.filter)
			req.URL.RawQuery = q.Encode()
			reqURL = req.URL.String()
		}
		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("filter=%q: %v", tc.filter, err)
		}
		var body struct {
			ModelVersions []json.RawMessage `json:"model_versions"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close() //nolint:errcheck
			t.Fatalf("filter=%q decode: %v", tc.filter, err)
		}
		resp.Body.Close() //nolint:errcheck
		if len(body.ModelVersions) != tc.wantN {
			t.Errorf("filter=%q: want %d, got %d", tc.filter, tc.wantN, len(body.ModelVersions))
		}
	}
}
