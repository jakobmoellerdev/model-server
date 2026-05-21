package ocm

import (
	"context"
	"fmt"
	"io"

	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// OpenResource returns a streaming reader for a named file within a component version.
// It matches by the ext.ocm.software/model-server.filename label or by resource name.
// Caller must close the returned ReadCloser.
func OpenResource(cv ComponentVersion, resourceName string) (io.ReadCloser, int64, error) {
	comp := cv.Descriptor.Component
	for i := range comp.Resources {
		r := &comp.Resources[i]
		filename := labelString(r.Labels, LabelFilename)
		if filename == "" {
			filename = r.Name
		}
		if filename != resourceName && r.Name != resourceName {
			continue
		}
		return openReader(cv.repo, comp.Name, comp.Version, r.ToIdentity())
	}
	return nil, 0, fmt.Errorf("resource %q not found in component version", resourceName)
}

func openReader(
	repo repository.ComponentVersionRepository,
	component, version string,
	identity runtime.Identity,
) (io.ReadCloser, int64, error) {
	b, _, err := repo.GetLocalResource(context.Background(), component, version, identity)
	if err != nil {
		return nil, 0, fmt.Errorf("get local resource: %w", err)
	}

	var size int64 = -1
	if sa, ok := b.(interface{ Size() int64 }); ok {
		size = sa.Size()
	}

	rc, err := b.ReadCloser()
	if err != nil {
		return nil, 0, fmt.Errorf("open blob reader: %w", err)
	}

	return rc, size, nil
}
