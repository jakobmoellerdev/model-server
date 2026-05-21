#!/usr/bin/env python3
"""
MLflow Model Registry API usage examples for model-server.

Install deps:
    pip install mlflow

Run against the sample component:
    python3 examples/usage/mlflow.py

The sample component is published at:
    ghcr.io/jakobmoellerdev/model-server/models
Model ID: example-org/tiny-model
"""

import os

BASE_URL = os.environ.get("MODEL_SERVER_URL", "http://localhost:8080")
MODEL_NAME = os.environ.get("MODEL_ID", "example-org/tiny-model")
MODEL_VERSION = os.environ.get("MODEL_VERSION", "1.0.0")

import mlflow  # noqa: E402
from mlflow import MlflowClient  # noqa: E402

mlflow.set_tracking_uri(BASE_URL)
client = MlflowClient(tracking_uri=BASE_URL)

print(f"=== MLflow Model Registry @ {BASE_URL} ===\n")

# 1. Search all registered models
print("--- client.search_registered_models() ---")
models = client.search_registered_models()
for rm in models:
    versions = rm.latest_versions or []
    ver_str = versions[0].version if versions else "—"
    print(f"  {rm.name:40s}  latest_version={ver_str}")
print()

# 2. Search with filter
print(f"--- search_registered_models(filter_string=\"name='{MODEL_NAME}'\") ---")
filtered = client.search_registered_models(filter_string=f"name='{MODEL_NAME}'")
for rm in filtered:
    print(f"  {rm.name}")
    for v in rm.latest_versions or []:
        print(f"    version={v.version}  status={v.status}")
print()

# 3. Get a specific model version
print(f"--- client.get_model_version({MODEL_NAME!r}, {MODEL_VERSION!r}) ---")
try:
    mv = client.get_model_version(MODEL_NAME, MODEL_VERSION)
    print(f"  name:    {mv.name}")
    print(f"  version: {mv.version}")
    print(f"  status:  {mv.status}")
    print(f"  source:  {mv.source or '(none)'}")
except Exception as exc:
    print(f"  error: {exc}")
print()

# 4. Get download URI
print(f"--- client.get_model_version_download_uri({MODEL_NAME!r}, {MODEL_VERSION!r}) ---")
try:
    uri = client.get_model_version_download_uri(MODEL_NAME, MODEL_VERSION)
    print(f"  artifact_uri: {uri}")
    print(f"  (append filename, e.g. {uri}config.json)")
except Exception as exc:
    print(f"  error: {exc}")
print()

# 5. Search model versions
print("--- client.search_model_versions() ---")
try:
    versions = client.search_model_versions()
    for v in versions:
        print(f"  {v.name}@{v.version}  status={v.status}")
except Exception as exc:
    print(f"  error: {exc}")
print()

print("Done.")
