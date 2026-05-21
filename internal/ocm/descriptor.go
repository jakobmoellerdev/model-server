package ocm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	descriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/repository"
)

// ComponentVersion groups a descriptor with the repo it came from, for blob access.
type ComponentVersion struct {
	Descriptor *descriptor.Descriptor
	repo       repository.ComponentVersionRepository
}

// ComponentInfo holds model metadata extracted from an OCM component version.
type ComponentInfo struct {
	ID         string
	Component  string
	Version    string
	Task       string
	Library    string
	Family     string
	License    string
	Gated      bool
	Private    bool
	Signed     bool
	Labels     map[string]string
	Files      []ModelFile
	CreatedAt  time.Time
	ModifiedAt time.Time
	Digest     string
}

// ModelFile is a single file entry within an OCM component version.
type ModelFile struct {
	Path      string
	Size      int64
	Digest    string
	MediaType string
	IsLFS     bool
}

// ExtractInfo maps an OCM component descriptor to ComponentInfo using the
// ext.ocm.software/model-server.* label schema.
func ExtractInfo(cv ComponentVersion, log *slog.Logger) (*ComponentInfo, error) {
	comp := cv.Descriptor.Component
	labels := comp.Labels

	modelID := labelString(labels, LabelModelID)
	if modelID == "" {
		return nil, fmt.Errorf("component %s/%s missing required label %s",
			comp.Name, comp.Version, LabelModelID)
	}

	files := mapFiles(cv, log)
	signed := len(cv.Descriptor.Signatures) > 0

	return &ComponentInfo{
		ID:         modelID,
		Component:  comp.Name,
		Version:    comp.Version,
		Task:       labelString(labels, LabelTask),
		Library:    labelString(labels, LabelLibrary),
		Family:     labelString(labels, LabelFamily),
		License:    labelString(labels, LabelLicense),
		Gated:      labelBool(labels, LabelGated),
		Private:    labelBool(labels, LabelPrivate),
		Signed:     signed,
		Labels:     allLabels(labels),
		Files:      files,
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Digest:     comp.Name + "@" + comp.Version,
	}, nil
}

func mapFiles(cv ComponentVersion, log *slog.Logger) []ModelFile {
	entries := make([]ModelFile, 0, len(cv.Descriptor.Component.Resources))

	for i := range cv.Descriptor.Component.Resources {
		r := &cv.Descriptor.Component.Resources[i]
		rlabels := r.Labels

		filename := labelString(rlabels, LabelFilename)
		if filename == "" {
			filename = r.Name
		}

		mt := MediaTypeForFormat(labelString(rlabels, LabelFormat))
		isLFS := labelBool(rlabels, LabelIsLFS) || r.Type == ResourceTypeWeights

		var size int64
		var digest string
		if r.Digest != nil {
			digest = r.Digest.Value
		}

		// Try to get size from the blob
		b, _, err := cv.repo.GetLocalResource(
			context.Background(),
			cv.Descriptor.Component.Name,
			cv.Descriptor.Component.Version,
			r.ToIdentity(),
		)
		if err != nil {
			log.Debug("cannot get local resource for size probe",
				slog.String("resource", r.Name), slog.Any("error", err))
		} else if sa, ok := b.(interface{ Size() int64 }); ok {
			size = sa.Size()
		}

		entries = append(entries, ModelFile{
			Path:      filename,
			Size:      size,
			Digest:    "sha256:" + digest,
			MediaType: mt,
			IsLFS:     isLFS,
		})
	}
	return entries
}

func labelString(labels []descriptor.Label, name string) string {
	for i := range labels {
		if labels[i].Name == name {
			var s string
			if err := json.Unmarshal(labels[i].Value, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

func labelBool(labels []descriptor.Label, name string) bool {
	for i := range labels {
		if labels[i].Name == name {
			var b bool
			if err := json.Unmarshal(labels[i].Value, &b); err == nil {
				return b
			}
			// also accept "true"/"false" string
			var s string
			if err := json.Unmarshal(labels[i].Value, &s); err == nil {
				return s == "true"
			}
		}
	}
	return false
}

func allLabels(labels []descriptor.Label) map[string]string {
	m := make(map[string]string, len(labels))
	for _, l := range labels {
		var s string
		if err := json.Unmarshal(l.Value, &s); err == nil {
			m[l.Name] = s
		}
	}
	return m
}
