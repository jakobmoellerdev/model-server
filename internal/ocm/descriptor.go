// Package ocm wraps the OCM SDK for model-server use.
// It exposes ComponentInfo (label-derived metadata) and file access
// without importing internal/registry, avoiding import cycles.
package ocm

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"ocm.software/ocm/api/ocm"
	metav1 "ocm.software/ocm/api/ocm/compdesc/meta/v1"
)

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

// ExtractInfo maps an OCM ComponentVersionAccess to a ComponentInfo using the
// ai.modelserver.io/* label schema.
func ExtractInfo(cv ocm.ComponentVersionAccess, log *slog.Logger) (*ComponentInfo, error) {
	cd := cv.GetDescriptor()
	labels := cd.Labels

	modelID := labelString(labels, LabelModelID)
	if modelID == "" {
		return nil, fmt.Errorf("component %s/%s missing required label %s",
			cd.Name, cd.Version, LabelModelID)
	}

	files := mapFiles(cv, log)
	signed := len(cd.Signatures) > 0

	return &ComponentInfo{
		ID:         modelID,
		Component:  cd.Name,
		Version:    cd.Version,
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
		Digest:     cd.Name + "@" + cd.Version,
	}, nil
}

func mapFiles(cv ocm.ComponentVersionAccess, log *slog.Logger) []ModelFile {
	resources := cv.GetResources()
	entries := make([]ModelFile, 0, len(resources))

	for _, r := range resources {
		meta := r.Meta()
		rlabels := meta.Labels

		filename := labelString(rlabels, LabelFilename)
		if filename == "" {
			filename = meta.GetName()
		}

		mt := MediaTypeForFormat(labelString(rlabels, LabelFormat))
		isLFS := labelBool(rlabels, LabelIsLFS) || meta.GetType() == ResourceTypeWeights

		var size int64
		var digest string
		if d := meta.Digest; d != nil {
			digest = d.Value
		}

		am, err := r.AccessMethod()
		if err != nil {
			log.Warn("cannot get access method", slog.String("resource", meta.GetName()), slog.Any("error", err))
		} else {
			ba := am.AsBlobAccess()
			size = ba.Size()
			if mt == "application/octet-stream" && am.MimeType() != "" {
				mt = am.MimeType()
			}
			am.Close()
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

func labelString(labels metav1.Labels, name string) string {
	var s string
	if _, err := labels.GetValue(name, &s); err == nil {
		return s
	}
	if idx := labels.GetIndex(name); idx >= 0 {
		var raw json.RawMessage
		if _, err := labels.GetValue(name, &raw); err == nil {
			var v string
			if err := json.Unmarshal(raw, &v); err == nil {
				return v
			}
		}
	}
	return ""
}

func labelBool(labels metav1.Labels, name string) bool {
	var b bool
	_, _ = labels.GetValue(name, &b)
	return b
}

func allLabels(labels metav1.Labels) map[string]string {
	m := make(map[string]string, len(labels))
	for _, l := range labels {
		var s string
		if err := json.Unmarshal(l.Value, &s); err == nil {
			m[l.Name] = s
		}
	}
	return m
}
