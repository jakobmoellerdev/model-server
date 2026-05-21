package ocm

const (
	LabelModelID   = "ext.ocm.software/model-server.model-id"
	LabelTask      = "ext.ocm.software/model-server.task"
	LabelLibrary   = "ext.ocm.software/model-server.library"
	LabelLicense   = "ext.ocm.software/model-server.license"
	LabelFamily    = "ext.ocm.software/model-server.family"
	LabelBaseModel = "ext.ocm.software/model-server.base-model"
	LabelGated     = "ext.ocm.software/model-server.gated"
	LabelPrivate   = "ext.ocm.software/model-server.private"
	LabelFilename  = "ext.ocm.software/model-server.filename"
	LabelFormat    = "ext.ocm.software/model-server.format"
	LabelIsLFS     = "ext.ocm.software/model-server.lfs"

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
