package ollama

import (
	"time"

	"github.com/open-component-model/model-server/internal/registry"
)

type TagsResponse struct {
	Models []ModelSummary `json:"models"`
}

type ModelSummary struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

type Details struct {
	Format            string `json:"format,omitempty"`
	Family            string `json:"family,omitempty"`
	ParameterSize     string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
}

type ShowRequest struct {
	Name string `json:"name"`
}

type ShowResponse struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
	License    string    `json:"license,omitempty"`
}

type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure"`
	Stream   *bool  `json:"stream"`
}

type PullEvent struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

func toSummary(d registry.ModelDescriptor) ModelSummary {
	var total int64
	for _, f := range d.Files {
		total += f.Size
	}
	return ModelSummary{
		Name: ollamaName(d), ModifiedAt: d.ModifiedAt,
		Size: total, Digest: d.Digest,
		Details: Details{Family: d.Family},
	}
}

func toShowResponse(d registry.ModelDescriptor) ShowResponse {
	var total int64
	for _, f := range d.Files {
		total += f.Size
	}
	return ShowResponse{
		Name: ollamaName(d), ModifiedAt: d.ModifiedAt,
		Size: total, Digest: d.Digest, License: d.License,
		Details: Details{Family: d.Family},
	}
}

func ollamaName(d registry.ModelDescriptor) string {
	if d.Version != "" {
		return d.ID + ":" + d.Version
	}
	return d.ID + ":latest"
}
