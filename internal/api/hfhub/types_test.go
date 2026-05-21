package hfhub

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-component-model/model-server/internal/registry"
)

func TestHFTime_MarshalJSON_UTCFormat(t *testing.T) {
	ts := time.Date(2024, 3, 15, 12, 30, 45, 0, time.UTC)
	ht := hfTime{ts}
	b, err := json.Marshal(ht)
	require.NoError(t, err)
	// Must match exactly: "2006-01-02T15:04:05.000Z"
	assert.Equal(t, `"2024-03-15T12:30:45.000Z"`, string(b))
}

func TestHFTime_MarshalJSON_NonUTCConverted(t *testing.T) {
	// 13:00 UTC expressed as 08:00 in a UTC-5 fixed zone
	loc := time.FixedZone("UTC-5", -5*60*60)
	ts := time.Date(2024, 3, 15, 8, 0, 0, 0, loc) // 08:00 -05:00 = 13:00 UTC
	ht := hfTime{ts}
	b, err := json.Marshal(ht)
	require.NoError(t, err)
	assert.Equal(t, `"2024-03-15T13:00:00.000Z"`, string(b))
}

func TestToModelInfo_Fields(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	desc := registry.ModelDescriptor{
		ID:         "test-org/my-model",
		Task:       "text-generation",
		Library:    "transformers",
		Family:     "llama",
		License:    "apache-2.0",
		Gated:      false,
		Private:    false,
		ModifiedAt: now,
		CreatedAt:  now,
		Files: []registry.FileEntry{
			{Path: "config.json", Size: 100, Digest: "sha256:abc"},
			{Path: "model.safetensors", Size: 1000, Digest: "sha256:def", IsLFS: true},
		},
	}

	info := toModelInfo(desc)

	assert.Equal(t, "test-org/my-model", info.ID)
	assert.Equal(t, "test-org/my-model", info.ModelID)
	assert.Equal(t, "test-org", info.Author)
	assert.Equal(t, "text-generation", info.PipelineTag)
	assert.Equal(t, "transformers", info.LibraryName)
	assert.False(t, info.Private)
	assert.False(t, info.Gated)

	require.NotNil(t, info.CardData)
	assert.Equal(t, "apache-2.0", info.CardData.License)

	assert.Contains(t, info.Tags, "text-generation")
	assert.Contains(t, info.Tags, "llama")
}

func TestToModelInfo_Siblings(t *testing.T) {
	desc := registry.ModelDescriptor{
		ID: "org/model",
		Files: []registry.FileEntry{
			{Path: "config.json", Size: 42, Digest: "sha256:aaa"},
			{Path: "model.safetensors", Size: 999, Digest: "sha256:bbb", IsLFS: true},
		},
	}
	info := toModelInfo(desc)

	require.Len(t, info.Siblings, 2)

	cfg := info.Siblings[0]
	assert.Equal(t, "config.json", cfg.Rfilename)
	assert.Equal(t, int64(42), cfg.Size)
	assert.Equal(t, "sha256:aaa", cfg.BlobID)
	assert.Nil(t, cfg.LFS)

	wts := info.Siblings[1]
	assert.Equal(t, "model.safetensors", wts.Rfilename)
	require.NotNil(t, wts.LFS)
	assert.Equal(t, int64(999), wts.LFS.Size)
	assert.Equal(t, "sha256:bbb", wts.LFS.SHA256)
}

func TestToModelInfo_AuthorNoSlash(t *testing.T) {
	desc := registry.ModelDescriptor{ID: "singlename"}
	info := toModelInfo(desc)
	assert.Equal(t, "", info.Author)
}

func TestModelInfo_JSON_BlobIDCamelCase(t *testing.T) {
	desc := registry.ModelDescriptor{
		ID:    "org/model",
		Files: []registry.FileEntry{{Path: "f.bin", Digest: "sha256:xyz"}},
	}
	info := toModelInfo(desc)
	b, err := json.Marshal(info)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(b), `"blobId"`), "blobId must be camelCase in JSON")
	assert.False(t, strings.Contains(string(b), `"blob_id"`), "snake_case blob_id must not appear")
}

func TestModelInfo_JSON_CardDataNested(t *testing.T) {
	desc := registry.ModelDescriptor{ID: "org/model", License: "mit"}
	info := toModelInfo(desc)
	b, err := json.Marshal(info)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"cardData"`)
	assert.Contains(t, string(b), `"license":"mit"`)
}
