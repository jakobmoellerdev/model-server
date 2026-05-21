package mlflow

import (
	"fmt"
	"time"

	"github.com/open-component-model/model-server/internal/registry"
)

// All response types follow the MLflow REST API schema.
// https://mlflow.org/docs/latest/rest-api.html

// RegisteredModel mirrors mlflow.entities.model_registry.RegisteredModel.
type RegisteredModel struct {
	Name              string         `json:"name"`
	CreationTimestamp int64          `json:"creation_timestamp"`
	LastUpdatedTimestamp int64       `json:"last_updated_timestamp"`
	Description       string         `json:"description,omitempty"`
	LatestVersions    []ModelVersion `json:"latest_versions,omitempty"`
	Tags              []Tag          `json:"tags,omitempty"`
}

// ModelVersion mirrors mlflow.entities.model_registry.ModelVersion.
type ModelVersion struct {
	Name                 string `json:"name"`
	Version              string `json:"version"`
	CreationTimestamp    int64  `json:"creation_timestamp"`
	LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
	Description          string `json:"description,omitempty"`
	Source               string `json:"source,omitempty"`
	Status               string `json:"status"` // READY | PENDING_REGISTRATION | FAILED_REGISTRATION
	RunID                string `json:"run_id,omitempty"`
	Tags                 []Tag  `json:"tags,omitempty"`
}

// Tag is an MLflow key-value metadata tag.
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SearchRegisteredModelsResponse wraps a page of RegisteredModel results.
type SearchRegisteredModelsResponse struct {
	RegisteredModels  []RegisteredModel `json:"registered_models"`
	NextPageToken     string            `json:"next_page_token,omitempty"`
}

// GetRegisteredModelResponse wraps a single RegisteredModel.
type GetRegisteredModelResponse struct {
	RegisteredModel RegisteredModel `json:"registered_model"`
}

// SearchModelVersionsResponse wraps a page of ModelVersion results.
type SearchModelVersionsResponse struct {
	ModelVersions []ModelVersion `json:"model_versions"`
	NextPageToken string         `json:"next_page_token,omitempty"`
}

// GetModelVersionResponse wraps a single ModelVersion.
type GetModelVersionResponse struct {
	ModelVersion ModelVersion `json:"model_version"`
}

// GetModelVersionDownloadURIResponse holds the artifact download URI.
type GetModelVersionDownloadURIResponse struct {
	ArtifactURI string `json:"artifact_uri"`
}

// ErrorResponse is the MLflow error envelope.
type ErrorResponse struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func toRegisteredModel(d registry.ModelDescriptor) RegisteredModel {
	ts := toMillis(d.CreatedAt)
	updated := toMillis(d.ModifiedAt)

	tags := labelsToTags(d.Labels)

	ver := toModelVersion(d)
	return RegisteredModel{
		Name:                 d.ID,
		CreationTimestamp:    ts,
		LastUpdatedTimestamp: updated,
		Description:          fmt.Sprintf("%s model (%s)", d.Family, d.Task),
		LatestVersions:       []ModelVersion{ver},
		Tags:                 tags,
	}
}

func toModelVersion(d registry.ModelDescriptor) ModelVersion {
	ver := d.Version
	if ver == "" {
		ver = "1"
	}
	return ModelVersion{
		Name:                 d.ID,
		Version:              ver,
		CreationTimestamp:    toMillis(d.CreatedAt),
		LastUpdatedTimestamp: toMillis(d.ModifiedAt),
		Description:          d.License,
		Status:               "READY",
		Tags:                 labelsToTags(d.Labels),
	}
}

func labelsToTags(labels map[string]string) []Tag {
	tags := make([]Tag, 0, len(labels))
	for k, v := range labels {
		tags = append(tags, Tag{Key: k, Value: v})
	}
	return tags
}

func toMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
