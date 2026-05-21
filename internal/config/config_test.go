package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(f, []byte(content), 0o600))
	return f
}

func TestLoad_MinimalValid(t *testing.T) {
	path := writeConfig(t, `
server:
  listen: ":9090"
auth:
  mode: none
ocm:
  repositories:
    - name: test
      type: CTF
      url: /tmp/test.ctf
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, ":9090", cfg.Server.Listen)
	assert.Equal(t, "none", cfg.Auth.Mode)
	require.Len(t, cfg.OCM.Repositories, 1)
	assert.Equal(t, "CTF", cfg.OCM.Repositories[0].Type)
}

func TestLoad_Defaults(t *testing.T) {
	path := writeConfig(t, `
server:
  listen: ":8080"
auth:
  mode: none
ocm:
  repositories:
    - name: r
      type: CTF
      url: /tmp/x.ctf
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.True(t, cfg.APIs.HFHub.Enabled)
	assert.True(t, cfg.APIs.Ollama.Enabled)
	assert.Greater(t, cfg.OCM.RefreshInterval.Seconds(), 0.0)
	assert.Greater(t, cfg.OCM.IndexTTL.Seconds(), 0.0)
	assert.Greater(t, cfg.OCM.BlobCache.MaxSizeBytes, int64(0))
}

func TestLoad_EnvExpansion(t *testing.T) {
	t.Setenv("TEST_REPO_URL", "ghcr.io/example/models")
	path := writeConfig(t, `
server:
  listen: ":8080"
auth:
  mode: none
ocm:
  repositories:
    - name: r
      type: OCIRegistry
      url: ${TEST_REPO_URL}
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "ghcr.io/example/models", cfg.OCM.Repositories[0].URL)
}

func TestLoad_EnvExpansion_Missing(t *testing.T) {
	t.Setenv("MISSING_VAR", "") // ensure unset for this test
	path := writeConfig(t, `
server:
  listen: ":8080"
auth:
  mode: none
ocm:
  repositories:
    - name: r
      type: OCIRegistry
      url: ${MISSING_VAR}
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	// unexpanded placeholder left intact
	assert.Equal(t, "${MISSING_VAR}", cfg.OCM.Repositories[0].URL)
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

func TestLoad_InvalidAuthMode(t *testing.T) {
	path := writeConfig(t, `
server:
  listen: ":8080"
auth:
  mode: invalid
ocm:
  repositories:
    - name: r
      type: CTF
      url: /tmp/x.ctf
`)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestLoad_MissingRepositories(t *testing.T) {
	path := writeConfig(t, `
server:
  listen: ":8080"
auth:
  mode: none
ocm:
  repositories: []
`)
	_, err := Load(path)
	require.Error(t, err)
}
