package ocm

import (
	"fmt"
	"io"

	"ocm.software/ocm/api/ocm"
)

// OpenResource returns a streaming reader for a named file within a component version.
// Caller must close the returned ReadCloser.
func OpenResource(cv ocm.ComponentVersionAccess, resourceName string) (io.ReadCloser, int64, error) {
	for _, r := range cv.GetResources() {
		filename := labelString(r.Meta().Labels, LabelFilename)
		if filename == "" {
			filename = r.Meta().GetName()
		}
		if filename != resourceName && r.Meta().GetName() != resourceName {
			continue
		}
		return openReader(r)
	}
	return nil, 0, fmt.Errorf("resource %q not found in component version", resourceName)
}

func openReader(r ocm.ResourceAccess) (io.ReadCloser, int64, error) {
	am, err := r.AccessMethod()
	if err != nil {
		return nil, 0, fmt.Errorf("access method: %w", err)
	}

	size := am.AsBlobAccess().Size()

	reader, err := am.Reader()
	if err != nil {
		am.Close()
		return nil, 0, fmt.Errorf("get reader: %w", err)
	}

	return &closerChain{ReadCloser: reader, extra: am.Close}, size, nil
}

// closerChain closes an extra cleanup func after closing the inner ReadCloser.
type closerChain struct {
	io.ReadCloser
	extra func() error
}

func (c *closerChain) Close() error {
	err := c.ReadCloser.Close()
	if e2 := c.extra(); err == nil {
		err = e2
	}
	return err
}
