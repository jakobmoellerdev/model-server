package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDesc(id, component, task, family string) ModelDescriptor {
	return ModelDescriptor{
		ID:        id,
		Component: component,
		Task:      task,
		Family:    family,
	}
}

func TestIndex_AddAndResolve(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("org/model-a", "github.com/org/model-a", "text-generation", "llama"))
	idx.add(makeDesc("org/model-b", "github.com/org/model-b", "embeddings", "bert"))

	assert.Equal(t, "github.com/org/model-a", idx.resolveModelID("org/model-a"))
	assert.Equal(t, "github.com/org/model-b", idx.resolveModelID("org/model-b"))
	assert.Equal(t, "", idx.resolveModelID("nonexistent"))
}

func TestIndex_SearchNoFilter(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("org/a", "comp/a", "text-generation", "llama"))
	idx.add(makeDesc("org/b", "comp/b", "embeddings", "bert"))

	results := idx.search(SearchFilter{})
	assert.Len(t, results, 2)
}

func TestIndex_SearchByTask(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("org/a", "comp/a", "text-generation", "llama"))
	idx.add(makeDesc("org/b", "comp/b", "embeddings", "bert"))

	results := idx.search(SearchFilter{Task: "embeddings"})
	require.Len(t, results, 1)
	assert.Equal(t, "org/b", results[0].ID)
}

func TestIndex_SearchByQuery_ID(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("meta-llama/Llama-3", "comp/a", "text-generation", "llama"))
	idx.add(makeDesc("google/bert-base", "comp/b", "embeddings", "bert"))

	results := idx.search(SearchFilter{Query: "llama"})
	require.Len(t, results, 1)
	assert.Equal(t, "meta-llama/Llama-3", results[0].ID)
}

func TestIndex_SearchByQuery_Family(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("org/model-x", "comp/a", "text-generation", "mistral"))
	idx.add(makeDesc("org/model-y", "comp/b", "text-generation", "llama"))

	results := idx.search(SearchFilter{Query: "mistral"})
	require.Len(t, results, 1)
	assert.Equal(t, "org/model-x", results[0].ID)
}

func TestIndex_SearchLimit(t *testing.T) {
	idx := newIndex()
	for i := 0; i < 10; i++ {
		idx.add(makeDesc("org/model", "comp", "text-generation", "llama"))
	}
	results := idx.search(SearchFilter{Limit: 3})
	assert.Len(t, results, 3)
}

func TestIndex_SearchOffset(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("org/a", "comp/a", "text-generation", "llama"))
	idx.add(makeDesc("org/b", "comp/b", "text-generation", "llama"))
	idx.add(makeDesc("org/c", "comp/c", "text-generation", "llama"))

	results := idx.search(SearchFilter{Offset: 2})
	require.Len(t, results, 1)
	assert.Equal(t, "org/c", results[0].ID)
}

func TestIndex_SearchOffsetBeyondEnd(t *testing.T) {
	idx := newIndex()
	idx.add(makeDesc("org/a", "comp/a", "text-generation", "llama"))

	results := idx.search(SearchFilter{Offset: 5})
	assert.Empty(t, results)
}

func TestIndex_Empty(t *testing.T) {
	idx := newIndex()
	assert.Empty(t, idx.search(SearchFilter{}))
	assert.Equal(t, "", idx.resolveModelID("anything"))
}
