package registry

import (
	"strings"
	"sync"

	"github.com/open-component-model/model-server/internal/observability"
)

// index is a thread-safe in-memory listing of all known model descriptors.
type index struct {
	mu     sync.RWMutex
	models []ModelDescriptor
	byID   map[string]int // public model ID → slice index
	byComp map[string]int // OCM component name → slice index
}

func newIndex() *index {
	return &index{
		byID:   make(map[string]int),
		byComp: make(map[string]int),
	}
}

func (idx *index) add(d ModelDescriptor) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	pos := len(idx.models)
	idx.models = append(idx.models, d)
	idx.byID[d.ID] = pos
	idx.byComp[d.Component] = pos

	observability.IndexSize.Set(float64(len(idx.models)))
}

// resolveModelID returns the OCM component name for a public model ID.
func (idx *index) resolveModelID(modelID string) string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if pos, ok := idx.byID[modelID]; ok {
		return idx.models[pos].Component
	}
	return ""
}

// search returns models matching f.
func (idx *index) search(f SearchFilter) []ModelDescriptor {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	query := strings.ToLower(f.Query)
	var result []ModelDescriptor

	for _, d := range idx.models {
		if f.Task != "" && d.Task != f.Task {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(d.ID), query) &&
			!strings.Contains(strings.ToLower(d.Family), query) {
			continue
		}
		result = append(result, d)
	}

	if f.Offset > 0 && f.Offset < len(result) {
		result = result[f.Offset:]
	}
	if f.Limit > 0 && f.Limit < len(result) {
		result = result[:f.Limit]
	}
	return result
}
