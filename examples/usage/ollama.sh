#!/usr/bin/env bash
# Ollama API usage examples for model-server.
#
# Requires:
#   ollama CLI  https://ollama.com/download
#
# Run:
#   bash examples/usage/ollama.sh
#
# The sample component is published at:
#   ghcr.io/jakobmoellerdev/model-server/models
# Model ID: example-org/tiny-model

set -euo pipefail

BASE_URL="${MODEL_SERVER_URL:-http://localhost:8080}"
MODEL="${MODEL_ID:-example-org/tiny-model}"

export OLLAMA_HOST="$BASE_URL"

echo "=== Ollama API @ $BASE_URL ==="
echo

# 1. List available models
echo "--- ollama list ---"
ollama list
echo

# 2. Show model details
echo "--- ollama show $MODEL ---"
ollama show "$MODEL" || echo "(model not yet pulled)"
echo

# 3. Pull model files to local cache
echo "--- ollama pull $MODEL ---"
ollama pull "$MODEL"
echo

echo "=== curl equivalents ==="
echo

# 4. List via raw HTTP
echo "--- GET /api/tags ---"
curl -s "$BASE_URL/api/tags" | python3 -m json.tool
echo

# 5. Show via raw HTTP
echo "--- POST /api/show ---"
curl -s -X POST "$BASE_URL/api/show" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$MODEL\"}" | python3 -m json.tool
echo

# 6. Pull (streaming NDJSON) via raw HTTP
echo "--- POST /api/pull (NDJSON stream) ---"
curl -s -X POST "$BASE_URL/api/pull" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$MODEL\"}"
echo

echo "Done."
