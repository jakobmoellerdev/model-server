package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"sigs.k8s.io/yaml"
)

// Duration wraps time.Duration to support both string ("5m") and nanosecond-int
// forms in JSON/YAML unmarshaling.
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		d.Duration = v
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	d.Duration = time.Duration(n)
	return nil
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Config is the top-level server configuration.
type Config struct {
	Server      ServerConfig              `json:"server"      validate:"required"`
	Auth        AuthConfig                `json:"auth"`
	OCM         OCMConfig                 `json:"ocm"         validate:"required"`
	APIs        APIsConfig                `json:"apis"`
	Credentials map[string]CredentialSpec `json:"credentials"`
}

type ServerConfig struct {
	Listen          string   `json:"listen"          validate:"required"`
	ReadTimeout     Duration `json:"readTimeout"`
	WriteTimeout    Duration `json:"writeTimeout"`
	IdleTimeout     Duration `json:"idleTimeout"`
	ShutdownTimeout Duration `json:"shutdownTimeout"`
}

type AuthConfig struct {
	Mode       string      `json:"mode"       validate:"oneof=none bearer oidc"`
	TokensFile string      `json:"tokensFile"`
	OIDC       *OIDCConfig `json:"oidc"`
}

type OIDCConfig struct {
	Issuer   string `json:"issuer"   validate:"required,url"`
	Audience string `json:"audience" validate:"required"`
}

type OCMConfig struct {
	Repositories    []RepositorySpec `json:"repositories"    validate:"required,min=1"`
	BlobCache       BlobCacheConfig  `json:"blobCache"`
	Signatures      SignatureConfig  `json:"signatures"`
	RefreshInterval Duration         `json:"refreshInterval"`
	IndexTTL        Duration         `json:"indexTTL"`
}

type RepositorySpec struct {
	Name           string `json:"name"           validate:"required"`
	Type           string `json:"type"           validate:"required,oneof=OCIRegistry CTF"`
	URL            string `json:"url"            validate:"required"`
	CredentialsRef string `json:"credentialsRef"`
}

type BlobCacheConfig struct {
	Path         string   `json:"path"`
	MaxSizeBytes int64    `json:"maxSizeBytes"`
	TTL          Duration `json:"ttl"`
}

type SignatureConfig struct {
	Required    bool     `json:"required"`
	TrustedKeys []string `json:"trustedKeys"`
}

type APIsConfig struct {
	HFHub  APIConfig `json:"hfhub"`
	Ollama APIConfig `json:"ollama"`
	OpenAI APIConfig `json:"openai"`
	MLflow APIConfig `json:"mlflow"`
}

type APIConfig struct {
	Enabled bool `json:"enabled"`
}

type CredentialSpec struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Load reads, env-expands, and validates a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	expanded := expandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)

	if err := validator.New().Struct(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func expandEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val := os.Getenv(name); val != "" {
			return val
		}
		return match
	})
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.Server.ReadTimeout.Duration == 0 {
		cfg.Server.ReadTimeout = Duration{30 * time.Second}
	}
	if cfg.Server.IdleTimeout.Duration == 0 {
		cfg.Server.IdleTimeout = Duration{120 * time.Second}
	}
	if cfg.Server.ShutdownTimeout.Duration == 0 {
		cfg.Server.ShutdownTimeout = Duration{30 * time.Second}
	}
	if cfg.Auth.Mode == "" {
		cfg.Auth.Mode = "none"
	}
	if cfg.OCM.RefreshInterval.Duration == 0 {
		cfg.OCM.RefreshInterval = Duration{5 * time.Minute}
	}
	if cfg.OCM.IndexTTL.Duration == 0 {
		cfg.OCM.IndexTTL = Duration{60 * time.Second}
	}
	if cfg.OCM.BlobCache.MaxSizeBytes == 0 {
		cfg.OCM.BlobCache.MaxSizeBytes = 100 * 1024 * 1024 * 1024
	}
	if cfg.OCM.BlobCache.TTL.Duration == 0 {
		cfg.OCM.BlobCache.TTL = Duration{168 * time.Hour}
	}
	if cfg.OCM.BlobCache.Path == "" {
		cfg.OCM.BlobCache.Path = "/var/cache/model-server"
	}
	cfg.APIs.HFHub.Enabled = true
	cfg.APIs.Ollama.Enabled = true
	cfg.APIs.OpenAI.Enabled = true
	cfg.APIs.MLflow.Enabled = true
}
