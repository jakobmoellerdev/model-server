#!/usr/bin/env python3
"""
Hugging Face Hub API usage examples for model-server.

Install deps:
    pip install huggingface_hub transformers

Run against the sample component:
    python3 examples/usage/hfhub.py

The sample component is published at:
    ghcr.io/jakobmoellerdev/model-server/models
Model ID: example-org/tiny-model
"""

import os

BASE_URL = os.environ.get("HF_ENDPOINT", "http://localhost:8080")
MODEL_ID = os.environ.get("MODEL_ID", "example-org/tiny-model")
TOKEN = os.environ.get("HF_TOKEN", "any")  # model-server accepts any token in auth:none mode

os.environ["HF_ENDPOINT"] = BASE_URL

from huggingface_hub import HfApi, hf_hub_download  # noqa: E402

api = HfApi(endpoint=BASE_URL, token=TOKEN)

print(f"=== model-server @ {BASE_URL} ===\n")

# 1. List all models
print("--- list_models() ---")
models = list(api.list_models())
for m in models:
    print(f"  {m.id}  pipeline_tag={m.pipeline_tag}")
print()

# 2. Filter by task
print("--- list_models(filter='text-generation') ---")
for m in api.list_models(filter="text-generation"):
    print(f"  {m.id}")
print()

# 3. Model info — metadata + file list
print(f"--- model_info({MODEL_ID!r}) ---")
info = api.model_info(MODEL_ID)
print(f"  id:           {info.id}")
print(f"  pipeline_tag: {info.pipeline_tag}")
if info.card_data:
    print(f"  license:      {info.card_data.license}")
print(f"  siblings:")
for s in info.siblings or []:
    print(f"    {s.rfilename}  size={s.size}")
print()

# 4. File tree
print(f"--- list_repo_tree({MODEL_ID!r}) ---")
for entry in api.list_repo_tree(MODEL_ID):
    print(f"  {entry.path}  type={entry.type}")
print()

# 5. Download a single file
print(f"--- hf_hub_download({MODEL_ID!r}, 'config.json') ---")
local_path = hf_hub_download(
    repo_id=MODEL_ID,
    filename="config.json",
    endpoint=BASE_URL,
    token=TOKEN,
)
print(f"  cached at: {local_path}")
with open(local_path) as f:
    print(f"  content:   {f.read().strip()[:120]}")
print()

# 6. transformers — load config directly from server
print(f"--- AutoConfig.from_pretrained({MODEL_ID!r}) ---")
try:
    from transformers import AutoConfig

    cfg = AutoConfig.from_pretrained(MODEL_ID, token=TOKEN)
    print(f"  model_type: {cfg.model_type}")
except Exception as exc:
    print(f"  (transformers not available or config incompatible: {exc})")
print()

print("Done.")
