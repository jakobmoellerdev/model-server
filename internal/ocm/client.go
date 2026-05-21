// Package ocm wraps the new OCM Go bindings for model-server use.
package ocm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"ocm.software/open-component-model/bindings/go/blob/filesystem"
	"ocm.software/open-component-model/bindings/go/ctf"
	"ocm.software/open-component-model/bindings/go/oci"
	ocictf "ocm.software/open-component-model/bindings/go/oci/ctf"
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
	entries []repoEntry
	log     *slog.Logger
}

// NewClient opens all configured OCM repositories.
func NewClient(cfg config.OCMConfig, _ map[string]config.CredentialSpec, log *slog.Logger) (*Client, error) {
	entries := make([]repoEntry, 0, len(cfg.Repositories))
	for _, repoCfg := range cfg.Repositories {
		entry, err := openRepository(repoCfg, log)
		if err != nil {
			return nil, fmt.Errorf("open repository %s: %w", repoCfg.Name, err)
		}
		entries = append(entries, entry)
		log.Info("opened OCM repository", slog.String("name", repoCfg.Name), slog.String("url", repoCfg.URL))
	}
	return &Client{entries: entries, log: log}, nil
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
				// reset for next repo
				version = ""
			}
			continue
		}
		return ComponentVersion{Descriptor: desc, repo: e.repo}, nil
	}
	return ComponentVersion{}, fmt.Errorf("component %q version %q not found", componentName, version)
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

func openRepository(cfg config.RepositorySpec, log *slog.Logger) (repoEntry, error) {
	switch cfg.Type {
	case "OCIRegistry":
		resolver, err := url.New(
			url.WithBaseURL(cfg.URL),
		)
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
		lister := ocictf.NewComponentLister(archive)
		return repoEntry{repo: repo, lister: lister}, nil
	default:
		return repoEntry{}, fmt.Errorf("unsupported repository type %q", cfg.Type)
	}
}
