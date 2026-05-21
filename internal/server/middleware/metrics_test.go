package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectAPI_OllamaHFHub(t *testing.T) {
	assert.Equal(t, "ollama_or_hfhub", detectAPI("/api/models"))
	assert.Equal(t, "ollama_or_hfhub", detectAPI("/api/tags"))
	assert.Equal(t, "ollama_or_hfhub", detectAPI("/api/show"))
}

func TestDetectAPI_OpenAI(t *testing.T) {
	assert.Equal(t, "openai", detectAPI("/v1/completions"))
	assert.Equal(t, "openai", detectAPI("/v1/chat/completions"))
}

func TestDetectAPI_HFHub(t *testing.T) {
	assert.Equal(t, "hfhub", detectAPI("/org/model/resolve/main/config.json"))
	assert.Equal(t, "hfhub", detectAPI("/healthz"))
	assert.Equal(t, "hfhub", detectAPI("/"))
}

func TestMetrics_RecordsRequest(t *testing.T) {
	mw := Metrics()
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}
