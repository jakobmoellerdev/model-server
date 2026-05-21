# model-server — Proposal

## Problem & Motivation

AI model distribution has no open, supply-chain-aware standard. Teams rely on Hugging Face Hub or ad-hoc cloud buckets — neither provides cryptographic provenance, reproducible deployment, or air-gapped operation.

**Specific pain points:**

| Pain point | Status quo | This proposal |
|---|---|---|
| Model provenance | None — no signing, no SBOM | OCM signatures on every component version |
| Reproducibility | Mutable branch names (`main`) | Immutable versioned OCM components |
| Multi-registry replication | Manual sync scripts | OCM transport (CTF + OCI) handles it |
| Air-gapped deployment | Complex workarounds | CTF archives work fully offline |
| Client compatibility | Lock-in to HF SDK or Ollama | Both APIs served from one server |
| Auditability | HTTP logs only | Component descriptor carries full metadata + signatures |

model-server solves this by acting as a thin translation layer: AI clients see familiar HF Hub and Ollama APIs; behind the scenes every model is an OCM component stored in a standard OCI registry.

---

## Design Decisions

### 1. OCM as sole data plane

All model storage and retrieval goes through OCM bindings. No secondary database, no local model store beyond a blob cache. OCM provides:

- **Versioning** via component versions (semver)
- **Signing** via component signatures (cosign-compatible)
- **Multi-source** via multiple configured repositories, tried in order
- **Transport** via CTF archives for air-gapped scenarios

*Trade-off:* listing is only possible for CTF-backed repos (OCI registries don't expose a component list). OCI repos are still fully usable for lookup by name.

### 2. In-memory index with periodic refresh

At startup model-server builds an in-memory index by calling `ListComponents` across all repos with listers, then fetching each component's descriptor. The index is rebuilt on a configurable interval (`ocm.refreshInterval`, default 5m).

*Trade-off:* stale window between refresh ticks. Acceptable for model registries where write frequency is low. `Describe` always hits OCM live; only `Search`/`List` reads the index.

### 3. Four API surfaces (HF Hub, Ollama, OpenAI, MLflow), no inference

model-server is a **registry**, not an inference server. It serves model files and metadata only. All four API surfaces share one `ModelRegistry` interface — the same `OCMRegistry` instance handles all requests.

- **HF Hub** — full compatibility with `huggingface_hub` Python SDK and `transformers`
- **Ollama** — `ollama list`, `ollama pull`, `ollama show` via `OLLAMA_HOST`
- **OpenAI** — `/v1/models` list/retrieve for any client using the OpenAI SDK
- **MLflow** — model registry read path; downstream tools can discover and download artifacts via the MLflow client

*Trade-off:* Ollama `pull` streams blobs through the server rather than redirecting (OCI registries support redirects but the implementation currently streams). Future work: redirect to OCI CDN URL for `IsLFS=true` resources.

### 4. Middleware-first auth

Authentication is enforced in a single middleware layer before any handler runs. Three modes:

- `none` — open; useful for internal / air-gapped deployments
- `bearer` — static token list from file; constant-time compare
- `oidc` — configured but not yet wired (placeholder)

No per-handler auth logic. Handlers assume the request is authenticated.

### 5. New modular OCM bindings (`ocm.software/open-component-model/bindings/go/*`)

Chose the new fine-grained module set over the old monolith (`ocm.software/ocm`):

| | Old monolith | New bindings |
|---|---|---|
| Transitive deps | docker, grpcgcp, cosign, k8s, OPA | Minimal, per-concern modules |
| Binary size | ~80 MB | ~15 MB |
| Build time | ~90 s cold | ~20 s cold |
| API stability | Stable but large | New, may change |

*Risk:* new bindings are still `v0.x`. Tracked with the `ocm.software/open-component-model` org.

---

## API Compatibility Matrix

### HF Hub endpoints

| Endpoint | Status | Notes |
|---|---|---|
| `GET /api/models` | Implemented | Supports `?task=`, `?search=`, `?limit=`, `?skip=` |
| `GET /api/models/{owner}/{model}` | Implemented | Returns `ModelInfo` with siblings |
| `GET /api/models/{model}` | Implemented | Single-segment model ID |
| `GET /api/models/{owner}/{model}/tree/{revision}` | Implemented | Returns `[]TreeEntry` |
| `GET /{owner}/{model}/resolve/{revision}/*` | Implemented | Streams blob from OCM; sets `X-Repo-Commit`, `ETag`, `Content-Length` |
| `GET /{owner}/{model}/raw/{revision}/*` | Implemented | Alias for resolve |
| `POST /api/models` (create) | Not implemented | Read-only registry |
| `DELETE /api/models/{model}` | Not implemented | Read-only registry |

### Ollama endpoints

| Endpoint | Status | Notes |
|---|---|---|
| `GET /api/tags` | Implemented | Lists index contents |
| `POST /api/show` | Implemented | Returns model details + Modelfile |
| `POST /api/pull` | Implemented | Streams NDJSON progress |
| `DELETE /api/delete` | 405 Not Allowed | Read-only registry |
| `POST /api/push` | Stub | Not yet implemented |
| `POST /api/copy` | Stub | Not yet implemented |
| `POST /api/blobs/{digest}` | Not implemented | Upload path for push |
| `HEAD /api/blobs/{digest}` | Not implemented | Upload path for push |

### OpenAI endpoints

| Endpoint | Status | Notes |
|---|---|---|
| `GET /v1/models` | Implemented | Returns `{"object":"list","data":[...]}` |
| `GET /v1/models/{owner}/{model}` | Implemented | Two-segment model ID |
| `GET /v1/models/{model}` | Implemented | Single-segment model ID |

### MLflow Model Registry endpoints

| Endpoint | Status | Notes |
|---|---|---|
| `GET /api/2.0/mlflow/registered-models/search` | Implemented | `?filter=`, `?max_results=` |
| `GET /api/2.0/mlflow/registered-models/get` | Implemented | `?name=` required |
| `GET /api/2.0/mlflow/model-versions/search` | Implemented | `?filter=`, `?max_results=` |
| `GET /api/2.0/mlflow/model-versions/get` | Implemented | `?name=`, `?version=` |
| `GET /api/2.0/mlflow/model-versions/get-download-uri` | Implemented | Returns HF-Hub-compatible `/resolve/` URI |
| Write endpoints (create, update, delete, transition) | Not implemented | Read-only registry |

### Health / Observability

| Endpoint | Status |
|---|---|
| `GET /healthz` | Implemented — always 200 |
| `GET /readyz` | Implemented — 503 until index built |
| `GET /metrics` | Implemented — Prometheus |

---

## Security & Supply Chain

### Threat model

model-server is a **read gateway** to an OCM registry. Writes happen via OCM tooling (`ocm` CLI, CI pipelines) external to this server. The attack surface is:

1. Unauthenticated read of model files
2. Serving a tampered/backdoored model
3. Credential exposure in config

### Controls in place

| Threat | Control |
|---|---|
| Unauthenticated reads | `auth.mode: bearer` or `oidc` in config; middleware blocks all routes |
| Tampered model | `ocm.signatures.required: true` — `signing.go` verifies OCM component signature before serving |
| Credential exposure | Credentials use `${ENV_VAR}` interpolation only; never inline in YAML; `CredentialSpec` not logged |
| Timing attacks on tokens | `crypto/subtle.ConstantTimeCompare` in bearer middleware |
| Panic / crash | `chiMiddleware.Recoverer()` recovers panics, returns 500, logs stack |

### OCM supply-chain model

```
Model author
    │
    ▼
ocm CLI / GitHub Action
    │  signs with cosign key
    ▼
OCI Registry ──── signed OCM component
                        │
                        │ model-server verifies signature
                        ▼
                  Client receives model files
```

Each OCM component version carries:
- **Component descriptor** — name, version, provider, labels, resource list
- **Resource digests** — sha256 of every blob, committed in the descriptor
- **Signatures** — over the component descriptor digest

When `signatures.required: true`, model-server calls `ocm.signing.Verify(cv)` before `ExtractInfo`. A component with no valid signature or a mismatched digest is rejected with 403.

### Secrets in config

```yaml
credentials:
  ghcr-creds:
    username: ${GHCR_USERNAME}   # expanded at load time from env
    password: ${GHCR_TOKEN}      # never stored in config file
```

Config loading (`internal/config/config.go`) expands `${VAR}` patterns via `os.Expand` before YAML parsing. Raw config bytes are never logged.

---

## Repository Layout

```
model-server/
├── cmd/model-server/main.go        entry point, wires all layers
├── internal/
│   ├── config/                     YAML config + validation
│   ├── server/                     chi router, HTTP server, graceful shutdown
│   │   └── middleware/             auth · logging · metrics
│   ├── api/
│   │   ├── hfhub/                  Hugging Face Hub compatible handlers
│   │   ├── ollama/                 Ollama compatible handlers
│   │   ├── openai/                 OpenAI /v1/models handlers
│   │   ├── mlflow/                 MLflow Model Registry handlers
│   │   └── health/                 /healthz · /readyz
│   ├── registry/                   ModelRegistry interface + OCMRegistry impl
│   │   ├── registry.go             interface
│   │   ├── resolver.go             OCMRegistry implementation
│   │   ├── index.go                in-memory search index
│   │   └── types.go                ModelDescriptor · FileEntry · SearchFilter
│   ├── ocm/
│   │   ├── client.go               multi-repo OCM client
│   │   ├── descriptor.go           descriptor → ComponentInfo
│   │   ├── resource.go             blob access
│   │   └── annotations.go          label constants
│   └── observability/              slog setup · Prometheus metrics
├── test/integration/               full-stack tests against in-memory CTF repo
├── examples/
│   ├── component/                  example OCM component YAML
│   └── config/                     example model-server.yaml
└── docs/
    ├── architecture.md             diagrams + API surface reference
    └── proposal.md                 this document
```

---

## Running Locally

```bash
# Start against a local CTF archive
cat > model-server.yaml <<EOF
server:
  listen: ":8080"
auth:
  mode: none
ocm:
  repositories:
    - name: local
      type: CTF
      url: ./my-models.ctf
apis:
  hfhub:
    enabled: true
  ollama:
    enabled: true
  openai:
    enabled: true
  mlflow:
    enabled: true
EOF

go run ./cmd/model-server -config model-server.yaml

# Verify
curl http://localhost:8080/api/models                                  # HF Hub
curl http://localhost:8080/api/tags                                    # Ollama
curl http://localhost:8080/v1/models                                   # OpenAI
curl "http://localhost:8080/api/2.0/mlflow/registered-models/search"  # MLflow
curl -X POST http://localhost:8080/api/show -d '{"name":"org/my-model"}'
```

```bash
# Run tests
go test ./... -race -count=1
go test ./test/integration/... -v
```
