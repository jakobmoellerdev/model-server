package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/open-component-model/model-server/internal/config"
)

// Auth returns a middleware enforcing bearer token auth.
func Auth(cfg config.AuthConfig) func(http.Handler) http.Handler {
	if cfg.Mode == "none" || cfg.Mode == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	tokens := loadTokens(cfg.TokensFile)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get("Authorization")
			if !strings.HasPrefix(hdr, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(hdr, "Bearer ")
			if !validToken(token, tokens) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validToken(token string, tokens map[string]struct{}) bool {
	for allowed := range tokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(allowed)) == 1 {
			return true
		}
	}
	return false
}

func loadTokens(path string) map[string]struct{} {
	tokens := make(map[string]struct{})
	if path == "" {
		return tokens
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return tokens
	}
	for _, line := range strings.Split(string(data), "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
			tokens[t] = struct{}{}
		}
	}
	return tokens
}
