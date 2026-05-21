package ocm

import (
	"context"
	"fmt"
	"log/slog"

	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"

	"github.com/open-component-model/model-server/internal/config"
)

// Client wraps an OCM context and its configured repositories.
type Client struct {
	ctx   ocm.Context
	repos []ocm.Repository
	log   *slog.Logger
}

// NewClient opens all configured OCM repositories.
func NewClient(cfg config.OCMConfig, creds map[string]config.CredentialSpec, log *slog.Logger) (*Client, error) {
	ctx := ocm.DefaultContext()

	repos := make([]ocm.Repository, 0, len(cfg.Repositories))
	for _, repoCfg := range cfg.Repositories {
		repo, err := openRepository(ctx, repoCfg, creds, log)
		if err != nil {
			return nil, fmt.Errorf("open repository %s: %w", repoCfg.Name, err)
		}
		repos = append(repos, repo)
		log.Info("opened OCM repository", slog.String("name", repoCfg.Name), slog.String("url", repoCfg.URL))
	}

	return &Client{ctx: ctx, repos: repos, log: log}, nil
}

// Close releases all repository connections.
func (c *Client) Close() {
	for _, r := range c.repos {
		if err := r.Close(); err != nil {
			c.log.Warn("error closing OCM repository", slog.Any("error", err))
		}
	}
}

// LookupVersion finds a component version across configured repos.
// version="" returns the latest available version.
func (c *Client) LookupVersion(componentName, version string) (ocm.ComponentVersionAccess, error) {
	for _, repo := range c.repos {
		comp, err := repo.LookupComponent(componentName)
		if err != nil {
			continue
		}

		if version == "" {
			versions, err := comp.ListVersions()
			comp.Close()
			if err != nil || len(versions) == 0 {
				continue
			}
			version = versions[len(versions)-1]

			// re-open component for the version lookup
			comp2, err := repo.LookupComponent(componentName)
			if err != nil {
				continue
			}
			cv, err := comp2.LookupVersion(version)
			comp2.Close()
			if err != nil {
				continue
			}
			return cv, nil
		}

		cv, err := comp.LookupVersion(version)
		comp.Close()
		if err != nil {
			continue
		}
		return cv, nil
	}
	return nil, fmt.Errorf("component %q version %q not found", componentName, version)
}

// ListComponents returns all component names visible across configured repos.
func (c *Client) ListComponents(_ context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	var names []string

	for _, repo := range c.repos {
		lister, ok := repo.(interface {
			ComponentLister() ocm.ComponentLister
		})
		if !ok {
			continue
		}
		cl := lister.ComponentLister()
		if cl == nil {
			continue
		}
		comps, err := cl.GetComponents("", true)
		if err != nil {
			c.log.Warn("cannot list components", slog.Any("error", err))
			continue
		}
		for _, name := range comps {
			if _, dup := seen[name]; !dup {
				seen[name] = struct{}{}
				names = append(names, name)
			}
		}
	}
	return names, nil
}

func openRepository(
	ctx ocm.Context,
	cfg config.RepositorySpec,
	_ map[string]config.CredentialSpec,
	_ *slog.Logger,
) (ocm.Repository, error) {
	switch cfg.Type {
	case "OCIRegistry":
		spec := ocireg.NewRepositorySpec(cfg.URL)
		return ctx.RepositoryForSpec(spec)
	default:
		return nil, fmt.Errorf("unsupported repository type %q", cfg.Type)
	}
}
