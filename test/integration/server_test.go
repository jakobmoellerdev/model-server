// Package integration tests the full server stack against an in-memory OCM repository.
// It creates real OCM components with the ai.modelserver.io/* label schema,
// starts an httptest.Server with the production router, and exercises both the
// HuggingFace Hub and Ollama-compatible API endpoints.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ocmapi "ocm.software/ocm/api/ocm"
	metav1 "ocm.software/ocm/api/ocm/compdesc/meta/v1"
	"ocm.software/ocm/api/ocm/extensions/repositories/composition"
	"ocm.software/ocm/api/utils/blobaccess"

	"github.com/open-component-model/model-server/internal/api/hfhub"
	"github.com/open-component-model/model-server/internal/api/ollama"
	"github.com/open-component-model/model-server/internal/config"
	ocmclient "github.com/open-component-model/model-server/internal/ocm"
	"github.com/open-component-model/model-server/internal/registry"
	"github.com/open-component-model/model-server/internal/server"
)

const (
	testModelID   = "test-org/my-model"
	testComponent = "github.com/test-org/my-model"
	testVersion   = "1.0.0"
	testConfig    = `{"model_type":"llama","hidden_size":4096}`
	testWeights   = `fake-safetensors-bytes`
	testCard      = `---
license: apache-2.0
---
# My Model

A test model.`
)

// testRegistry wraps an OCM composition repo as a ModelRegistry via the real OCM client + resolver.
type testSetup struct {
	srv    *httptest.Server
	client *http.Client
}

func newTestSetup(t *testing.T) *testSetup {
	t.Helper()

	ctx := ocmapi.DefaultContext()

	// In-memory OCM repository (no disk, no network)
	repo := composition.NewRepository(ctx)
	t.Cleanup(func() { repo.Close() })

	addTestComponent(t, ctx, repo)

	reg := newInMemRegistry(t, repo)

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Auth:   config.AuthConfig{Mode: "none"},
		APIs:   config.APIsConfig{HFHub: config.APIConfig{Enabled: true}, Ollama: config.APIConfig{Enabled: true}},
	}

	handler := server.NewRouter(cfg, reg)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &testSetup{srv: srv, client: srv.Client()}
}

func (ts *testSetup) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := ts.client.Get(ts.srv.URL + path)
	require.NoError(t, err)
	return resp
}

func (ts *testSetup) post(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := ts.client.Post(ts.srv.URL+path, "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	return resp
}

// ---------------------------------------------------------------------------
// HF Hub tests
// ---------------------------------------------------------------------------

func TestHFHub_ListModels(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/api/models")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var models []hfhub.ModelInfo
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&models))
	require.Len(t, models, 1)
	assert.Equal(t, testModelID, models[0].ID)
	assert.Equal(t, "text-generation", models[0].PipelineTag)
	assert.Equal(t, "transformers", models[0].LibraryName)
	assert.Equal(t, "apache-2.0", models[0].License)
}

func TestHFHub_ListModels_FilterByTask(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/api/models?task=embeddings")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var models []hfhub.ModelInfo
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&models))
	assert.Empty(t, models, "no embedding models exist")
}

func TestHFHub_ModelInfo(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/api/models/test-org/my-model")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var info hfhub.ModelInfo
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&info))
	assert.Equal(t, testModelID, info.ID)
	assert.Equal(t, "test-org", info.Author)
	assert.False(t, info.Private)
	assert.False(t, info.Gated)

	// siblings should include all 3 resources
	assert.Len(t, info.Siblings, 3)
	paths := make([]string, len(info.Siblings))
	for i, s := range info.Siblings {
		paths[i] = s.Rfilename
	}
	assert.Contains(t, paths, "config.json")
	assert.Contains(t, paths, "model.safetensors")
	assert.Contains(t, paths, "README.md")
}

func TestHFHub_ModelInfo_NotFound(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/api/models/nonexistent/model")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHFHub_FileTree(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/api/models/test-org/my-model/tree/main")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var entries []hfhub.TreeEntry
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&entries))
	assert.Len(t, entries, 3)
	for _, e := range entries {
		assert.Equal(t, "blob", e.Type)
	}
}

func TestHFHub_DownloadFile(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/test-org/my-model/resolve/main/config.json")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, testConfig, string(body))
}

func TestHFHub_DownloadFile_NotFound(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/test-org/my-model/resolve/main/nonexistent.bin")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Ollama tests
// ---------------------------------------------------------------------------

func TestOllama_Tags(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/api/tags")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var tags ollama.TagsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&tags))
	require.Len(t, tags.Models, 1)
	assert.Contains(t, tags.Models[0].Name, testModelID)
	assert.Equal(t, "llama", tags.Models[0].Details.Family)
}

func TestOllama_Show(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.post(t, "/api/show", ollama.ShowRequest{Name: testModelID})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var show ollama.ShowResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&show))
	assert.Contains(t, show.Name, testModelID)
	assert.Equal(t, "llama", show.Details.Family)
	assert.Equal(t, "apache-2.0", show.License)
}

func TestOllama_Show_NotFound(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.post(t, "/api/show", ollama.ShowRequest{Name: "ghost/model"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOllama_Pull_StreamsProgress(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.post(t, "/api/pull", ollama.PullRequest{Name: testModelID})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/x-ndjson", resp.Header.Get("Content-Type"))

	// Read all NDJSON lines
	var events []ollama.PullEvent
	decoder := json.NewDecoder(resp.Body)
	for {
		var e ollama.PullEvent
		if err := decoder.Decode(&e); err != nil {
			break
		}
		events = append(events, e)
	}

	require.NotEmpty(t, events)
	assert.Equal(t, "pulling manifest", events[0].Status)
	last := events[len(events)-1]
	assert.Equal(t, "success", last.Status)
}

func TestOllama_Delete_NotAllowed(t *testing.T) {
	ts := newTestSetup(t)

	req, _ := http.NewRequest(http.MethodDelete, ts.srv.URL+"/api/delete", nil)
	resp, err := ts.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Health endpoints
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/healthz")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestReadyz(t *testing.T) {
	ts := newTestSetup(t)

	resp := ts.get(t, "/readyz")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// addTestComponent creates a model component in the composition repo.
func addTestComponent(t *testing.T, ctx ocmapi.Context, repo ocmapi.Repository) {
	t.Helper()

	cv := composition.NewComponentVersion(ctx, testComponent, testVersion)
	t.Cleanup(func() { cv.Close() })

	// Set required labels on the component descriptor
	cd := cv.GetDescriptor()
	setLabel(t, &cd.Labels, ocmclient.LabelModelID, testModelID)
	setLabel(t, &cd.Labels, ocmclient.LabelTask, "text-generation")
	setLabel(t, &cd.Labels, ocmclient.LabelLibrary, "transformers")
	setLabel(t, &cd.Labels, ocmclient.LabelFamily, "llama")
	setLabel(t, &cd.Labels, ocmclient.LabelLicense, "apache-2.0")

	// config.json
	configMeta := ocmapi.NewResourceMeta("config", ocmclient.ResourceTypeConfig, metav1.LocalRelation)
	configMeta.Labels = labelsFor(t, "config.json", "json")
	require.NoError(t, cv.SetResourceBlob(configMeta,
		blobaccess.ForString("application/json", testConfig), "", nil))

	// model.safetensors (treated as weights → IsLFS=true)
	weightsMeta := ocmapi.NewResourceMeta("weights", ocmclient.ResourceTypeWeights, metav1.LocalRelation)
	weightsMeta.Labels = labelsFor(t, "model.safetensors", "safetensors")
	require.NoError(t, cv.SetResourceBlob(weightsMeta,
		blobaccess.ForString("application/x-safetensors", testWeights), "", nil))

	// README.md
	cardMeta := ocmapi.NewResourceMeta("model-card", ocmclient.ResourceTypeModelCard, metav1.LocalRelation)
	cardMeta.Labels = labelsFor(t, "README.md", "markdown")
	require.NoError(t, cv.SetResourceBlob(cardMeta,
		blobaccess.ForString("text/markdown", testCard), "", nil))

	require.NoError(t, repo.AddComponentVersion(cv))
}

func setLabel(t *testing.T, labels *metav1.Labels, name, value string) {
	t.Helper()
	require.NoError(t, labels.SetValue(name, value))
}

func labelsFor(t *testing.T, filename, format string) metav1.Labels {
	t.Helper()
	var lbls metav1.Labels
	setLabel(t, &lbls, ocmclient.LabelFilename, filename)
	setLabel(t, &lbls, ocmclient.LabelFormat, format)
	return lbls
}

// newInMemRegistry wires the composition repo into an OCMRegistry via a thin shim.
func newInMemRegistry(t *testing.T, repo ocmapi.Repository) registry.ModelRegistry {
	t.Helper()

	client := ocmclient.NewClientFromRepository(repo)
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	reg, err := registry.NewOCMRegistry(client, 0, log)
	require.NoError(t, err)
	return reg
}

// Ensure the server runs without errors
var _ = fmt.Sprintf
var _ = slog.Default
