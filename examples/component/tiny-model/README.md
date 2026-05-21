# tiny-model

A minimal synthetic model published by [model-server](https://github.com/jakobmoellerdev/model-server) for integration testing and API exploration.

This is **not** a real language model. It contains:

- `config.json` — a minimal transformer-style configuration
- `tokenizer.json` — a stub BPE tokenizer vocabulary
- `weights.bin` — 16 zero bytes (placeholder weights, not usable for inference)
- `README.md` — this file

## Purpose

Use this component to:

- Test that model-server is correctly reading an OCM component from a registry
- Exercise all four API surfaces (HF Hub, Ollama, OpenAI, MLflow) without downloading large real models
- Validate your deployment end-to-end before pointing model-server at production model components

## OCM Component

```
name:     github.com/jakobmoellerdev/model-server/models/tiny-model
version:  1.0.0
registry: ghcr.io/jakobmoellerdev/model-server/models
model-id: example-org/tiny-model
task:     text-generation
```

## License

CC0 1.0 — public domain, no restrictions.
