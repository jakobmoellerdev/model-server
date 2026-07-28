package hfhub

import (
	"time"

	"github.com/open-component-model/model-server/internal/registry"
)

// hfTime marshals as the UTC format the HF SDK expects: 2006-01-02T15:04:05.000Z
type hfTime struct{ time.Time }

func (t hfTime) MarshalJSON() ([]byte, error) {
	s := t.UTC().Format(`"2006-01-02T15:04:05.000Z"`)
	return []byte(s), nil
}

// ModelInfo is the HF Hub-compatible model metadata response shape.
type ModelInfo struct {
	ID           string    `json:"id"`
	ModelID      string    `json:"modelId"`
	Sha          string    `json:"sha,omitempty"`
	Author       string    `json:"author,omitempty"`
	LastModified hfTime    `json:"lastModified"`
	Private      bool      `json:"private"`
	Gated        bool      `json:"gated"`
	Tags         []string  `json:"tags,omitempty"`
	PipelineTag  string    `json:"pipeline_tag,omitempty"`
	LibraryName  string    `json:"library_name,omitempty"`
	CardData     *CardData `json:"cardData,omitempty"`
	Siblings     []Sibling `json:"siblings"`
	CreatedAt    hfTime    `json:"createdAt"`
}

// CardData holds model card metadata (license, language, etc.).
type CardData struct {
	License string `json:"license,omitempty"`
}

// Sibling represents a single file in the model repository.
type Sibling struct {
	Rfilename string   `json:"rfilename"`
	Size      int64    `json:"size,omitempty"`
	BlobID    string   `json:"blobId,omitempty"`
	LFS       *LFSInfo `json:"lfs,omitempty"`
}

// LFSInfo marks a large file served via redirect.
type LFSInfo struct {
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	PointerSize int    `json:"pointerSize"`
}

// TreeEntry is a file tree entry for /tree endpoints.
type TreeEntry struct {
	Type   string `json:"type"` // "blob"
	Path   string `json:"path"`
	Size   int64  `json:"size,omitempty"`
	BlobID string `json:"oid,omitempty"`
}

func toModelInfo(d registry.ModelDescriptor) ModelInfo {
	siblings := make([]Sibling, 0, len(d.Files))
	for _, f := range d.Files {
		sib := Sibling{Rfilename: f.Path, Size: f.Size, BlobID: f.Digest}
		if f.IsLFS {
			sib.LFS = &LFSInfo{Size: f.Size, SHA256: f.Digest}
		}
		siblings = append(siblings, sib)
	}

	var author string
	for i, c := range d.ID {
		if c == '/' {
			author = d.ID[:i]
			break
		}
	}

	tags := []string{d.Task}
	if d.Family != "" {
		tags = append(tags, d.Family)
	}

	return ModelInfo{
		ID: d.ID, ModelID: d.ID, Sha: d.Version, Author: author,
		LastModified: hfTime{d.ModifiedAt}, Private: d.Private, Gated: d.Gated,
		Tags: tags, PipelineTag: d.Task, LibraryName: d.Library,
		CardData: &CardData{License: d.License},
		Siblings: siblings, CreatedAt: hfTime{d.CreatedAt},
	}
}
