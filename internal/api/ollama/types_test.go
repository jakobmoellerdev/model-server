package ollama

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-component-model/model-server/internal/registry"
)

func TestOllamaName_WithVersion(t *testing.T) {
	d := registry.ModelDescriptor{ID: "org/model", Version: "1.0.0"}
	assert.Equal(t, "org/model:1.0.0", ollamaName(d))
}

func TestOllamaName_NoVersion(t *testing.T) {
	d := registry.ModelDescriptor{ID: "org/model", Version: ""}
	assert.Equal(t, "org/model:latest", ollamaName(d))
}

func TestToSummary_SizeSumsFiles(t *testing.T) {
	now := time.Now()
	d := registry.ModelDescriptor{
		ID: "org/model", Version: "1.0.0", Family: "llama",
		Digest: "sha256:abc", ModifiedAt: now,
		Files: []registry.FileEntry{
			{Path: "config.json", Size: 100},
			{Path: "model.safetensors", Size: 4000},
		},
	}
	s := toSummary(d)
	assert.Equal(t, "org/model:1.0.0", s.Name)
	assert.Equal(t, int64(4100), s.Size)
	assert.Equal(t, "llama", s.Details.Family)
	assert.Equal(t, "sha256:abc", s.Digest)
}

func TestToSummary_NoFiles(t *testing.T) {
	d := registry.ModelDescriptor{ID: "org/model"}
	s := toSummary(d)
	assert.Equal(t, int64(0), s.Size)
}

func TestToShowResponse_Fields(t *testing.T) {
	d := registry.ModelDescriptor{
		ID: "org/model", Version: "2.0", Family: "mistral",
		License: "apache-2.0", Digest: "sha256:xyz",
		Files: []registry.FileEntry{{Size: 500}, {Size: 300}},
	}
	s := toShowResponse(d)
	assert.Equal(t, "org/model:2.0", s.Name)
	assert.Equal(t, int64(800), s.Size)
	assert.Equal(t, "mistral", s.Details.Family)
	assert.Equal(t, "apache-2.0", s.License)
	assert.Equal(t, "sha256:xyz", s.Digest)
}

func TestSplitTag_WithColon(t *testing.T) {
	id, ver := splitTag("org/model:v1.2")
	assert.Equal(t, "org/model", id)
	assert.Equal(t, "v1.2", ver)
}

func TestSplitTag_NoColon(t *testing.T) {
	id, ver := splitTag("org/model")
	assert.Equal(t, "org/model", id)
	assert.Equal(t, "", ver)
}

func TestSplitTag_MultipleColons(t *testing.T) {
	id, ver := splitTag("ghcr.io/org/model:latest")
	assert.Equal(t, "ghcr.io/org/model", id)
	assert.Equal(t, "latest", ver)
}

func TestSplitTag_Empty(t *testing.T) {
	id, ver := splitTag("")
	assert.Equal(t, "", id)
	assert.Equal(t, "", ver)
}

func TestStatusFor_NotFound(t *testing.T) {
	assert.Equal(t, 404, statusFor(fmt.Errorf("model %q not found", "x")))
}

func TestStatusFor_Other(t *testing.T) {
	assert.Equal(t, 500, statusFor(fmt.Errorf("connection refused")))
}
