package ocm

const (
	LabelModelID   = "ai.modelserver.io/model-id"
	LabelTask      = "ai.modelserver.io/task"
	LabelLibrary   = "ai.modelserver.io/library"
	LabelLicense   = "ai.modelserver.io/license"
	LabelFamily    = "ai.modelserver.io/family"
	LabelBaseModel = "ai.modelserver.io/base-model"
	LabelGated     = "ai.modelserver.io/gated"
	LabelPrivate   = "ai.modelserver.io/private"
	LabelFilename  = "ai.modelserver.io/filename"
	LabelFormat    = "ai.modelserver.io/format"
	LabelIsLFS     = "ai.modelserver.io/lfs"

	ResourceTypeWeights   = "modelWeights"
	ResourceTypeConfig    = "modelConfig"
	ResourceTypeTokenizer = "tokenizer"
	ResourceTypeModelCard = "modelCard"
)

// MediaTypeForFormat maps a format string to a MIME type.
func MediaTypeForFormat(format string) string {
	switch format {
	case "safetensors":
		return "application/x-safetensors"
	case "gguf":
		return "application/x-gguf"
	case "pytorch":
		return "application/x-pytorch"
	case "onnx":
		return "application/x-onnx"
	case "json":
		return "application/json"
	case "markdown":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}
