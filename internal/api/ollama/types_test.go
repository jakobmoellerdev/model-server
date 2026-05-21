package ollama

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	// last colon is the split point
	id, ver := splitTag("ghcr.io/org/model:latest")
	assert.Equal(t, "ghcr.io/org/model", id)
	assert.Equal(t, "latest", ver)
}

func TestSplitTag_Empty(t *testing.T) {
	id, ver := splitTag("")
	assert.Equal(t, "", id)
	assert.Equal(t, "", ver)
}
