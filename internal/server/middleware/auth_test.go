package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-component-model/model-server/internal/config"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuth_ModeNone_PassesThrough(t *testing.T) {
	mw := Auth(config.AuthConfig{Mode: "none"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_ModeEmpty_PassesThrough(t *testing.T) {
	mw := Auth(config.AuthConfig{Mode: ""})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_Bearer_MissingHeader_Unauthorized(t *testing.T) {
	mw := Auth(config.AuthConfig{Mode: "bearer"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_Bearer_WrongScheme_Unauthorized(t *testing.T) {
	mw := Auth(config.AuthConfig{Mode: "bearer"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_Bearer_InvalidToken_Forbidden(t *testing.T) {
	f := writeTokenFile(t, "valid-token-abc\n")
	mw := Auth(config.AuthConfig{Mode: "bearer", TokensFile: f})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuth_Bearer_ValidToken_OK(t *testing.T) {
	f := writeTokenFile(t, "valid-token-abc\n# comment line\n\nsecond-token\n")
	mw := Auth(config.AuthConfig{Mode: "bearer", TokensFile: f})

	for _, tok := range []string{"valid-token-abc", "second-token"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		mw(okHandler()).ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code, "token=%q", tok)
	}
}

func TestAuth_Bearer_EmptyTokensFile_AllForbidden(t *testing.T) {
	f := writeTokenFile(t, "# only comments\n\n")
	mw := Auth(config.AuthConfig{Mode: "bearer", TokensFile: f})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer anything")
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuth_Bearer_MissingTokensFile_AllForbidden(t *testing.T) {
	mw := Auth(config.AuthConfig{Mode: "bearer", TokensFile: "/nonexistent/tokens.txt"})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer anything")
	w := httptest.NewRecorder()
	mw(okHandler()).ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestLoadTokens_ParsesLines(t *testing.T) {
	f := writeTokenFile(t, "abc\ndef\n# comment\n\n  ghi  \n")
	tokens := loadTokens(f)
	assert.Contains(t, tokens, "abc")
	assert.Contains(t, tokens, "def")
	assert.Contains(t, tokens, "ghi")
	assert.NotContains(t, tokens, "# comment")
	assert.Len(t, tokens, 3)
}

func TestLoadTokens_EmptyPath(t *testing.T) {
	tokens := loadTokens("")
	assert.Empty(t, tokens)
}

func TestValidToken_ConstantTime(t *testing.T) {
	tokens := map[string]struct{}{"secret": {}}
	assert.True(t, validToken("secret", tokens))
	assert.False(t, validToken("wrong", tokens))
	assert.False(t, validToken("", tokens))
}

func writeTokenFile(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "tokens.txt")
	require.NoError(t, os.WriteFile(f, []byte(content), 0o600))
	return f
}
