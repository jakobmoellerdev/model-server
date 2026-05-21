package ocm

import (
	"encoding/json"
	"testing"

	descriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"

	"github.com/stretchr/testify/assert"
)

func label(name, value string) descriptor.Label {
	b, _ := json.Marshal(value)
	return descriptor.Label{Name: name, Value: b}
}

func boolLabel(name string, value bool) descriptor.Label {
	b, _ := json.Marshal(value)
	return descriptor.Label{Name: name, Value: b}
}

func TestLabelString(t *testing.T) {
	labels := []descriptor.Label{
		label(LabelModelID, "meta-llama/Llama-3-8B"),
		label(LabelTask, "text-generation"),
	}

	assert.Equal(t, "meta-llama/Llama-3-8B", labelString(labels, LabelModelID))
	assert.Equal(t, "text-generation", labelString(labels, LabelTask))
	assert.Equal(t, "", labelString(labels, LabelFamily))
}

func TestLabelString_MissingKey(t *testing.T) {
	assert.Equal(t, "", labelString(nil, LabelModelID))
}

func TestLabelBool_True(t *testing.T) {
	labels := []descriptor.Label{boolLabel(LabelGated, true)}
	assert.True(t, labelBool(labels, LabelGated))
}

func TestLabelBool_False(t *testing.T) {
	labels := []descriptor.Label{boolLabel(LabelGated, false)}
	assert.False(t, labelBool(labels, LabelGated))
}

func TestLabelBool_StringTrue(t *testing.T) {
	b, _ := json.Marshal("true")
	labels := []descriptor.Label{{Name: LabelGated, Value: b}}
	assert.True(t, labelBool(labels, LabelGated))
}

func TestLabelBool_Missing(t *testing.T) {
	assert.False(t, labelBool(nil, LabelGated))
}

func TestAllLabels(t *testing.T) {
	labels := []descriptor.Label{
		label(LabelTask, "text-generation"),
		label(LabelFamily, "llama"),
	}
	m := allLabels(labels)
	assert.Equal(t, "text-generation", m[LabelTask])
	assert.Equal(t, "llama", m[LabelFamily])
}

func TestMediaTypeForFormat(t *testing.T) {
	cases := []struct {
		format string
		want   string
	}{
		{"safetensors", "application/x-safetensors"},
		{"gguf", "application/x-gguf"},
		{"pytorch", "application/x-pytorch"},
		{"onnx", "application/x-onnx"},
		{"json", "application/json"},
		{"markdown", "text/markdown"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, MediaTypeForFormat(c.format), "format=%q", c.format)
	}
}
