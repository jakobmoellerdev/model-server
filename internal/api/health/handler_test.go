package health

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-component-model/model-server/internal/registry"
)

type stubRegistry struct{ ready bool }

func (s *stubRegistry) Ready() bool { return s.ready }
func (s *stubRegistry) Search(_ context.Context, _ registry.SearchFilter) ([]registry.ModelDescriptor, error) {
	return nil, nil
}
func (s *stubRegistry) Describe(_ context.Context, _, _ string) (*registry.ModelDescriptor, error) {
	return nil, nil
}
func (s *stubRegistry) ListFiles(_ context.Context, _, _ string) ([]registry.FileEntry, error) {
	return nil, nil
}
func (s *stubRegistry) OpenFile(_ context.Context, _, _, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, nil
}
func (s *stubRegistry) Refresh(_ context.Context) error { return nil }

func TestLiveness_AlwaysOK(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	Liveness(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestReadiness_NotReady_503(t *testing.T) {
	reg := &stubRegistry{ready: false}
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	Readiness(reg)(w, r)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "not ready", body["status"])
}

func TestReadiness_Ready_200(t *testing.T) {
	reg := &stubRegistry{ready: true}
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	Readiness(reg)(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "ready", body["status"])
}
