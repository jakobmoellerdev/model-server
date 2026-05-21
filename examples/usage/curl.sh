#!/usr/bin/env bash
# Comprehensive curl examples for all model-server API surfaces.
#
# Requires: curl, jq
#
# Run:
#   bash examples/usage/curl.sh
#
# The sample component is published at:
#   ghcr.io/jakobmoellerdev/model-server/models
# Model ID: example-org/tiny-model

set -euo pipefail

BASE="${MODEL_SERVER_URL:-http://localhost:8080}"
MODEL="${MODEL_ID:-example-org/tiny-model}"
VERSION="${MODEL_VERSION:-1}"

# Derive owner/name
OWNER="${MODEL%%/*}"
NAME="${MODEL#*/}"

section() { echo; echo "━━━ $* ━━━"; echo; }

echo "=== model-server curl examples @ $BASE ==="

# ─────────────────────────────────────────────
section "Hugging Face Hub API"

echo "# List all models"
curl -fsSL "$BASE/api/models" | jq '.[].id'

echo
echo "# Filter by task"
curl -fsSL "$BASE/api/models?task=text-generation" | jq '.[].id'

echo
echo "# Model info"
curl -fsSL "$BASE/api/models/$MODEL" | jq '{id, pipeline_tag, siblings: [.siblings[].rfilename]}'

echo
echo "# File tree"
curl -fsSL "$BASE/api/models/$MODEL/tree/main" | jq '.[].path'

echo
echo "# Download config.json"
curl -fsSL "$BASE/$OWNER/$NAME/resolve/main/config.json"

echo
echo "# HEAD stat (ETag, Content-Length, X-Repo-Commit)"
curl -sI "$BASE/$OWNER/$NAME/resolve/main/config.json" | grep -iE 'etag|content-length|x-repo-commit'

# ─────────────────────────────────────────────
section "Ollama API"

echo "# List models"
curl -fsSL "$BASE/api/tags" | jq '.models[] | {name, size}'

echo
echo "# Show model details"
curl -fsSL -X POST "$BASE/api/show" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$MODEL\"}" | jq '{name: .model_info.name, details}'

echo
echo "# Pull (NDJSON progress stream)"
curl -fsSL -X POST "$BASE/api/pull" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$MODEL\"}"

# ─────────────────────────────────────────────
section "OpenAI API"

echo "# List all models"
curl -fsSL "$BASE/v1/models" | jq '.data[] | {id, owned_by}'

echo
echo "# Retrieve by two-segment ID"
curl -fsSL "$BASE/v1/models/$OWNER/$NAME" | jq '{id, object, owned_by, created}'

echo
echo "# Retrieve by full slash-joined ID (URL-encoded slash)"
curl -fsSL "$BASE/v1/models/$MODEL" | jq '{id, object}'

# ─────────────────────────────────────────────
section "MLflow Model Registry API"

echo "# Search registered models"
curl -fsSL "$BASE/api/2.0/mlflow/registered-models/search" \
  | jq '.registered_models[] | {name, latest_versions: [.latest_versions[].version]}'

echo
echo "# Get registered model"
curl -fsSL "$BASE/api/2.0/mlflow/registered-models/get?name=$MODEL" \
  | jq '.registered_model | {name, description}'

echo
echo "# Search model versions"
curl -fsSL "$BASE/api/2.0/mlflow/model-versions/search" \
  | jq '.model_versions[] | {name, version, status}'

echo
echo "# Get specific version"
curl -fsSL "$BASE/api/2.0/mlflow/model-versions/get?name=$MODEL&version=$VERSION" \
  | jq '.model_version | {name, version, status}'

echo
echo "# Get download URI"
curl -fsSL "$BASE/api/2.0/mlflow/model-versions/get-download-uri?name=$MODEL&version=$VERSION" \
  | jq '.artifact_uri'

echo
echo "# Search with MLflow filter expression"
curl -fsSL \
  --get --data-urlencode "filter=name = '$MODEL'" \
  "$BASE/api/2.0/mlflow/registered-models/search" \
  | jq '.registered_models[].name'

# ─────────────────────────────────────────────
section "Health & Observability"

echo "# Liveness"
curl -fsSL "$BASE/healthz"

echo
echo "# Readiness"
curl -fsSL "$BASE/readyz"

echo
echo "# Prometheus metrics (first 10 lines)"
curl -fsSL "$BASE/metrics" | head -10

echo
echo "Done."
