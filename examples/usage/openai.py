#!/usr/bin/env python3
"""
OpenAI-compatible API usage examples for model-server.

Install deps:
    pip install openai

Run against the sample component:
    python3 examples/usage/openai.py

The sample component is published at:
    ghcr.io/jakobmoellerdev/model-server/models
Model ID: example-org/tiny-model
"""

import os
from datetime import datetime, timezone

BASE_URL = os.environ.get("MODEL_SERVER_URL", "http://localhost:8080")
MODEL_ID = os.environ.get("MODEL_ID", "example-org/tiny-model")

from openai import OpenAI  # noqa: E402

client = OpenAI(
    base_url=f"{BASE_URL}/v1",
    api_key="any",  # model-server accepts any key in auth:none mode
)

print(f"=== OpenAI API @ {BASE_URL}/v1 ===\n")

# 1. List all models
print("--- client.models.list() ---")
page = client.models.list()
for m in page.data:
    created_dt = datetime.fromtimestamp(m.created, tz=timezone.utc).strftime("%Y-%m-%d")
    print(f"  {m.id:40s}  owned_by={m.owned_by}  created={created_dt}")
print()

# 2. Retrieve a specific model
print(f"--- client.models.retrieve({MODEL_ID!r}) ---")
model = client.models.retrieve(MODEL_ID)
print(f"  id:       {model.id}")
print(f"  object:   {model.object}")
print(f"  owned_by: {model.owned_by}")
print(f"  created:  {datetime.fromtimestamp(model.created, tz=timezone.utc).isoformat()}")
print()

# 3. Retrieve by two-segment path (owner/model)
owner, name = MODEL_ID.split("/", 1) if "/" in MODEL_ID else ("", MODEL_ID)
if owner:
    print(f"--- client.models.retrieve('{owner}/{name}') ---")
    m2 = client.models.retrieve(f"{owner}/{name}")
    print(f"  id: {m2.id}")
    print()

print("Done.")
