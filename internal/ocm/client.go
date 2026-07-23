// Package ocm wraps the new OCM Go bindings for model-server use.
package ocm

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"oras.land/oras-go/v2/registry/remote/auth"

	"ocm.software/open-component-model/bindings/go/blob/filesystem"
	"ocm.software/open-component-model/bindings/go/ctf"
	"ocm.software/open-component-model/bindings/go/oci"
	ocictf "ocm.software/open-component-model/bindings/go/oci/ctf"
	descriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/resolver/url"
	"ocm.software/open-component-model/bindings/go/repository"

	"github.com/open-component-model/model-server/internal/config"
)

// repoEntry groups a ComponentVersionRepository with its optional ComponentLister.
type repoEntry struct {
	repo   repository.ComponentVersionRepository
	lister repository.ComponentLister // nil for OCI registries that don't support listing
}

// Client holds one or more OCM repositories opened from config.
type Client struct {
	entries   []repoEntry
	log       *slog.Logger
	cachePath string

	// In-memory descriptor cache: key = "component:version"
	descMu    sync.RWMutex
	descCache map[string]*descriptor.Descriptor
}

// NewClient opens all configured OCM repositories.
func NewClient(cfg config.OCMConfig, creds map[string]config.CredentialSpec, log *slog.Logger) (*Client, error) {
	entries := make([]repoEntry, 0, len(cfg.Repositories))
	for _, repoCfg := range cfg.Repositories {
		entry, err := openRepository(repoCfg, creds, log)
		if err != nil {
			return nil, fmt.Errorf("open repository %s: %w", repoCfg.Name, err)
		}
		entries = append(entries, entry)
		log.Info("opened OCM repository", slog.String("name", repoCfg.Name), slog.String("url", repoCfg.URL))
	}

	cachePath := cfg.BlobCache.Path
	if cachePath == "" {
		cachePath = "/tmp/model-server-cache"
	}
	if err := os.MkdirAll(cachePath, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	return &Client{entries: entries, log: log, cachePath: cachePath, descCache: make(map[string]*descriptor.Descriptor)}, nil
}

// NewClientFromRepository creates a Client from a single already-open repository.
// The lister is used for listing components; pass nil if the repo doesn't support it.
func NewClientFromRepository(repo repository.ComponentVersionRepository, lister repository.ComponentLister) *Client {
	return &Client{
		entries: []repoEntry{{repo: repo, lister: lister}},
		log:     slog.Default(),
	}
}

// LookupVersion finds a component version across configured repos.
// version "", "main", or "latest" returns the latest available version.
func (c *Client) LookupVersion(ctx context.Context, componentName, version string) (ComponentVersion, error) {
	if version == "" || version == "main" || version == "latest" {
		version = ""
	}

	// Check descriptor cache
	if version != "" {
		cacheKey := componentName + ":" + version
		c.descMu.RLock()
		if desc, ok := c.descCache[cacheKey]; ok {
			c.descMu.RUnlock()
			c.log.Debug("descriptor cache hit", slog.String("key", cacheKey))
			// Find the repo that owns this (use first entry for blob access)
			return ComponentVersion{Descriptor: desc, repo: c.entries[0].repo}, nil
		}
		c.descMu.RUnlock()
	}

	for _, e := range c.entries {
		if version == "" {
			versions, err := e.repo.ListComponentVersions(ctx, componentName)
			if err != nil || len(versions) == 0 {
				continue
			}
			version = versions[0] // sorted descending by semver
		}

		desc, err := e.repo.GetComponentVersion(ctx, componentName, version)
		if err != nil {
			if version != "" {
				version = ""
			}
			continue
		}

		// Cache the descriptor
		cacheKey := componentName + ":" + version
		c.descMu.Lock()
		c.descCache[cacheKey] = desc
		c.descMu.Unlock()
		c.log.Debug("descriptor cached", slog.String("key", cacheKey))

		return ComponentVersion{Descriptor: desc, repo: e.repo}, nil
	}
	return ComponentVersion{}, fmt.Errorf("component %q version %q not found", componentName, version)
}

// cacheKeyForResource returns a filesystem-safe cache key for a resource blob.
func cacheKeyForResource(component, version, resourceName string) string {
	h := sha256.Sum256([]byte(component + ":" + version + ":" + resourceName))
	return hex.EncodeToString(h[:16])
}

// GetCachedResource returns a cached resource blob if available, or fetches and caches it.
func (c *Client) GetCachedResource(ctx context.Context, cv ComponentVersion, resourceName string) (io.ReadCloser, int64, error) {
	comp := cv.Descriptor.Component
	cacheFile := filepath.Join(c.cachePath, cacheKeyForResource(comp.Name, comp.Version, resourceName))

	// Check disk cache
	if info, err := os.Stat(cacheFile); err == nil && info.Size() > 0 {
		c.log.Debug("blob cache hit", slog.String("resource", resourceName), slog.String("file", cacheFile))
		f, err := os.Open(cacheFile)
		if err != nil {
			return nil, 0, err
		}
		return f, info.Size(), nil
	}

	// Fetch from remote
	c.log.Info("blob cache miss, fetching from registry", slog.String("resource", resourceName))
	rc, size, err := OpenResource(cv, resourceName)
	if err != nil {
		return nil, 0, err
	}

	// Write to cache file while streaming
	tmpFile := cacheFile + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		rc.Close()
		return nil, 0, fmt.Errorf("create cache file: %w", err)
	}

	written, err := io.Copy(f, rc)
	rc.Close()
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return nil, 0, fmt.Errorf("write cache file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		os.Remove(tmpFile)
		return nil, 0, fmt.Errorf("rename cache file: %w", err)
	}

	if size <= 0 {
		size = written
	}

	// Return the cached file
	cachedF, err := os.Open(cacheFile)
	if err != nil {
		return nil, 0, err
	}
	return cachedF, size, nil
}

// ListComponents returns all component names visible across configured repos.
func (c *Client) ListComponents(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	var names []string

	for _, e := range c.entries {
		if e.lister == nil {
			continue
		}
		if err := e.lister.ListComponents(ctx, "", func(page []string) error {
			for _, name := range page {
				if _, dup := seen[name]; !dup {
					seen[name] = struct{}{}
					names = append(names, name)
				}
			}
			return nil
		}); err != nil {
			c.log.Warn("cannot list components", slog.Any("error", err))
		}
	}
	return names, nil
}

func openRepository(cfg config.RepositorySpec, creds map[string]config.CredentialSpec, log *slog.Logger) (repoEntry, error) {
	switch cfg.Type {
	case "OCIRegistry":
		// Force tcp4 to avoid IPv6 DNS fallback timeouts on macOS.
		dialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", addr)
			},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		}
		httpClient := &http.Client{Transport: transport}

		var resolverOpts []url.Option
		resolverOpts = append(resolverOpts, url.WithBaseURL(cfg.URL))

		// Use ORAS auth.Client for proper OCI token exchange
		if cfg.CredentialsRef != "" {
			if cred, ok := creds[cfg.CredentialsRef]; ok && cred.Password != "" {
				log.Info("using ORAS auth client for OCI registry", slog.String("ref", cfg.CredentialsRef))
				orasClient := &auth.Client{
					Client: httpClient,
					Cache:  auth.NewCache(),
					Credential: auth.StaticCredential("ghcr.io", auth.Credential{
						Username: cred.Username,
						Password: cred.Password,
					}),
				}
				resolverOpts = append(resolverOpts, url.WithBaseClient(orasClient))
			} else {
				resolverOpts = append(resolverOpts, url.WithBaseClient(httpClient))
			}
		} else {
			resolverOpts = append(resolverOpts, url.WithBaseClient(httpClient))
		}

		resolver, err := url.New(resolverOpts...)
		if err != nil {
			return repoEntry{}, fmt.Errorf("create resolver for %s: %w", cfg.URL, err)
		}
		repo, err := oci.NewRepository(
			oci.WithResolver(resolver),
			oci.WithCreator("model-server"),
			oci.WithLogger(log),
		)
		if err != nil {
			return repoEntry{}, fmt.Errorf("create OCI repository: %w", err)
		}
		return repoEntry{repo: repo, lister: nil}, nil
	case "CTF":
		fs, err := filesystem.NewFS(cfg.URL, os.O_RDONLY)
		if err != nil {
			return repoEntry{}, fmt.Errorf("open CTF filesystem %s: %w", cfg.URL, err)
		}
		archive := ctf.NewFileSystemCTF(fs)
		store := ocictf.NewFromCTF(archive)
		repo, err := oci.NewRepository(
			ocictf.WithCTF(store),
			oci.WithCreator("model-server"),
			oci.WithLogger(log),
		)
		if err != nil {
			return repoEntry{}, fmt.Errorf("create CTF repository: %w", err)
		}
		return repoEntry{repo: repo, lister: nil}, nil
	default:
		return repoEntry{}, fmt.Errorf("unsupported repository type %q", cfg.Type)
	}
}
