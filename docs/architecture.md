# Architecture Diagrams

Diagrams generated with [fireworks-tech-graph](https://github.com/yizhiyanhua-ai/fireworks-tech-graph), blueprint style.
Source SVGs in `docs/diagrams/` — PNG exports at 2x resolution.

---

## 1. System Context

How model-server sits between AI clients and OCM-backed OCI registries.

![System Context](diagrams/01-system-context.png)

---

## 2. Package / Component Structure

Internal Go package dependency graph, grouped by layer.

![Package Structure](diagrams/02-package-structure.png)

---

## 3. Request Flow — HF SDK `list_models()`

Sequence from client call through middleware, handler, registry, and the background index-build goroutine.

![Request Flow](diagrams/03-request-flow.png)

---

## 4. Data Model — OCM Component → ModelDescriptor

How an OCM component descriptor (labels + resources) maps to model-server's internal `ModelDescriptor` and `FileEntry` types.

![Data Model](diagrams/04-data-model.png)

---

## Label Schema

| Label | Required | Example |
|---|---|---|
| `ext.ocm.software/model-server.model-id` | yes | `meta-llama/Llama-3-8B` |
| `ext.ocm.software/model-server.task` | yes | `text-generation` |
| `ext.ocm.software/model-server.library` | no | `transformers` |
| `ext.ocm.software/model-server.license` | no | `apache-2.0` |
| `ext.ocm.software/model-server.family` | no | `llama` |
| `ext.ocm.software/model-server.base-model` | no | `meta-llama/Llama-3` |
| `ext.ocm.software/model-server.gated` | no | `true` |
| `ext.ocm.software/model-server.private` | no | `false` |

## Format → Media Type

| Format | Media Type |
|---|---|
| `safetensors` | `application/x-safetensors` |
| `gguf` | `application/x-gguf` |
| `pytorch` | `application/x-pytorch` |
| `onnx` | `application/x-onnx` |
| `json` | `application/json` |
| `markdown` | `text/markdown` |
| _(default)_ | `application/octet-stream` |

---

## API Surface

### Hugging Face Hub

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/models` | List models (`?search=`, `?task=`, `?limit=`, `?skip=`) |
| `GET` | `/api/models/{owner}/{model}` | Model metadata + file list |
| `GET` | `/api/models/{model}` | Single-segment model ID |
| `GET` | `/api/models/{owner}/{model}/tree/{revision}` | File tree |
| `GET/HEAD` | `/{owner}/{model}/resolve/{revision}/{file}` | Download or stat a file |
| `GET/HEAD` | `/{owner}/{model}/raw/{revision}/{file}` | Raw file access (alias for resolve) |

### Ollama

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/tags` | List stored models |
| `POST` | `/api/show` | Model info + Modelfile |
| `POST` | `/api/pull` | Pull model, streams NDJSON progress |

### OpenAI

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/models` | List models (OpenAI-compatible object: list) |
| `GET` | `/v1/models/{owner}/{model}` | Get model by two-segment ID |
| `GET` | `/v1/models/{model}` | Get model by single-segment ID |

### MLflow Model Registry

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/2.0/mlflow/registered-models/search` | Search registered models (`?filter=`, `?max_results=`) |
| `GET` | `/api/2.0/mlflow/registered-models/get` | Get model by name (`?name=`) |
| `GET` | `/api/2.0/mlflow/model-versions/search` | Search model versions (`?filter=`, `?max_results=`) |
| `GET` | `/api/2.0/mlflow/model-versions/get` | Get version (`?name=`, `?version=`) |
| `GET` | `/api/2.0/mlflow/model-versions/get-download-uri` | Get download URI (`?name=`, `?version=`) |

### Health / Observability

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness probe — always 200 |
| `GET` | `/readyz` | Readiness probe — 503 until index built |
| `GET` | `/metrics` | Prometheus metrics |
