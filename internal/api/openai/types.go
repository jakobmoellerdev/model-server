package openai

import (
	"time"

	"github.com/open-component-model/model-server/internal/registry"
)

// ModelObject is the OpenAI API model object returned by /v1/models.
// https://platform.openai.com/docs/api-reference/models/object
type ModelObject struct {
	ID      string `json:"id"`
	Object  string `json:"object"`   // always "model"
	Created int64  `json:"created"`  // unix timestamp
	OwnedBy string `json:"owned_by"` // provider / organisation
}

// ModelList is the paginated list returned by GET /v1/models.
type ModelList struct {
	Object string        `json:"object"` // always "list"
	Data   []ModelObject `json:"data"`
}

func toModelObject(d registry.ModelDescriptor) ModelObject {
	owner := d.Labels["ext.ocm.software/model-server.library"]
	if owner == "" {
		// derive owner from the model ID prefix (e.g. "meta-llama/Llama-3" → "meta-llama")
		for i, c := range d.ID {
			if c == '/' {
				owner = d.ID[:i]
				break
			}
		}
	}
	if owner == "" {
		owner = "unknown"
	}

	created := d.CreatedAt
	if created.IsZero() {
		created = time.Unix(0, 0)
	}

	return ModelObject{
		ID:      d.ID,
		Object:  "model",
		Created: created.Unix(),
		OwnedBy: owner,
	}
}
