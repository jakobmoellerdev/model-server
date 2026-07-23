#!/usr/bin/env python3
import os
import sys
import json
import urllib.request
import urllib.error
import subprocess

DEFAULT_MODEL_ID = "hf-internal-testing/tiny-random-gpt2"

def main():
    model_id = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_MODEL_ID
    model_suffix = model_id.split("/")[-1]
    
    print(f"=== Resolving Hugging Face Model: {model_id} ===")
    
    # Query HF API for model details
    api_url = f"https://huggingface.co/api/models/{model_id}"
    try:
        req = urllib.request.Request(api_url, headers={"User-Agent": "Mozilla/5.0"})
        with urllib.request.urlopen(req) as response:
            model_data = json.loads(response.read().decode())
    except urllib.error.URLError as e:
        print(f"Error querying Hugging Face API: {e}", file=sys.stderr)
        print("Please check your internet connection or the model ID.", file=sys.stderr)
        sys.exit(1)

    # Extract files
    siblings = model_data.get("siblings", [])
    if not siblings:
        print(f"No files found for model {model_id}!", file=sys.stderr)
        sys.exit(1)

    # Filter out files we don't need
    # Include .model for SentencePiece tokenizers!
    allowed_extensions = {".json", ".txt", ".bin", ".safetensors", ".md", ".model"}
    excluded_names = {".gitattributes", "tf_model.h5", "flax_model.msgpack", "model.onnx", "coreml"}
    
    files_to_download = []
    for s in siblings:
        filename = s["rfilename"]
        ext = os.path.splitext(filename)[1]
        if filename in excluded_names or filename.startswith("."):
            continue
        if ext in allowed_extensions:
            files_to_download.append(filename)

    print(f"Found {len(files_to_download)} files to download: {files_to_download}")

    # Set up directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_dir = os.path.join(script_dir, model_suffix)
    os.makedirs(model_dir, exist_ok=True)

    # Download files
    for filename in files_to_download:
        dest_path = os.path.join(model_dir, filename)
        os.makedirs(os.path.dirname(dest_path), exist_ok=True)
        
        url = f"https://huggingface.co/{model_id}/resolve/main/{filename}"
        print(f"Downloading {filename}...")
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(req) as resp, open(dest_path, "wb") as out_file:
                out_file.write(resp.read())
        except urllib.error.URLError as e:
            print(f"Error downloading {filename}: {e}", file=sys.stderr)
            sys.exit(1)

    print(f"Successfully downloaded model files to: {model_dir}")

    # Generate component-constructor.yaml
    print("\n=== Generating OCM Component Constructor ===")
    
    # Map model metadata
    pipeline_tag = model_data.get("pipeline_tag", "text-generation")
    library_name = model_data.get("library_name", "transformers")
    license_tag = "apache-2.0"
    for tag in model_data.get("tags", []):
        if tag.startswith("license:"):
            license_tag = tag.split(":")[-1]

    # Component name MUST be lowercase to satisfy OCM CLI regex validation!
    component_name = f"example.org/{model_suffix.lower()}"
    provider_name = model_data.get("author", "huggingface").lower()

    yaml_lines = [
        "components:",
        f"  - name: {component_name}",
        "    version: 1.0.0",
        "    provider:",
        f"      name: {provider_name}",
        "    labels:",
        f"      - name: ext.ocm.software/model-server.model-id",
        f"        value: {json.dumps(model_id)}",
        f"      - name: ext.ocm.software/model-server.task",
        f"        value: {json.dumps(pipeline_tag)}",
        f"      - name: ext.ocm.software/model-server.library",
        f"        value: {json.dumps(library_name)}",
        f"      - name: ext.ocm.software/model-server.license",
        f"        value: {json.dumps(license_tag)}",
        "    resources:"
    ]

    for filename in files_to_download:
        ext = os.path.splitext(filename)[1]
        
        # Resource names MUST be alphanumeric (without hyphens or underscores) to satisfy 
        # OCM's strict lowerCamelCase transformation ID schema.
        r_name = filename.replace(".", "").replace("/", "").replace("_", "").replace("-", "").lower()
        
        r_type = "generic"
        r_format = "bin"
        if filename == "config.json":
            r_type = "modelConfig"
            r_format = "json"
        elif filename == "README.md":
            r_type = "modelCard"
            r_format = "markdown"
        elif filename.endswith(".safetensors"):
            r_type = "modelWeights"
            r_format = "safetensors"
        elif filename == "pytorch_model.bin":
            r_type = "modelWeights"
            r_format = "pytorch"
        elif any(t in filename for t in ["vocab", "tokenizer", "merges", "special_tokens"]) or ext == ".model":
            r_type = "tokenizer"
            r_format = "json" if ext == ".json" else "bin" if ext == ".model" else "text"
            
        media_type = "application/json" if r_format == "json" else "application/octet-stream"

        yaml_lines.extend([
            f"      - name: {r_name}",
            f"        type: {r_type}",
            f"        relation: local",
            f"        input:",
            f"          type: file",
            f"          path: {model_suffix}/{filename}",
            f"          mediaType: {media_type}",
            f"        labels:",
            f"          - name: ext.ocm.software/model-server.filename",
            f"            value: {json.dumps(filename)}",
            f"          - name: ext.ocm.software/model-server.format",
            f"            value: {json.dumps(r_format)}"
        ])

    constructor_path = os.path.join(script_dir, "real-model-constructor.yaml")
    with open(constructor_path, "w") as f:
        f.write("\n".join(yaml_lines) + "\n")
    print(f"Created constructor specification at: {constructor_path}")

    # Build local OCM archive (CTF)
    print("\n=== Compiling OCM Component Version ===")
    archive_dir = os.path.join(script_dir, "local-transport-archive")
    if os.path.exists(archive_dir):
        import shutil
        shutil.rmtree(archive_dir)

    # Detect OCM binary
    ocm_bin = "/Users/D067928/.local/bin/ocm"
    if not os.path.exists(ocm_bin):
        ocm_bin = "ocm"

    try:
        help_output = subprocess.check_output([ocm_bin, "add", "componentversions", "--help"], stderr=subprocess.STDOUT).decode()
        if "--repository" in help_output:
            print(f"Detected OCM CLI ({ocm_bin}) using --repository / --constructor syntax.")
            cmd = [
                ocm_bin, "add", "componentversions",
                "--repository", f"ctf::{archive_dir}",
                "--constructor", "real-model-constructor.yaml"
            ]
        else:
            print(f"Detected OCM CLI ({ocm_bin}) using --create / --file syntax.")
            cmd = [
                ocm_bin, "add", "componentversions",
                "--create",
                "--file", archive_dir,
                "real-model-constructor.yaml"
            ]
    except Exception as e:
        print(f"Failed to auto-detect OCM CLI syntax, using default: {e}")
        cmd = [
            ocm_bin, "add", "componentversions",
            "--create",
            "--file", archive_dir,
            "real-model-constructor.yaml"
        ]

    try:
        print(f"Running: {' '.join(cmd)}")
        subprocess.run(cmd, cwd=script_dir, check=True)
        print(f"Successfully compiled OCM archive at: {archive_dir}")
    except Exception as e:
        print(f"Error compiling OCM archive using OCM CLI: {e}", file=sys.stderr)
        sys.exit(1)

    # Generate custom model-server.yaml pointing to the local CTF archive
    print("\n=== Generating Local model-server.yaml Config ===")
    
    server_yaml = f"""server:
  listen: ":8080"
  readTimeout: 30s
  writeTimeout: 0s
  idleTimeout: 120s
  shutdownTimeout: 30s

auth:
  mode: none

ocm:
  repositories:
    - name: local-ctf-repo
      type: CTF
      url: "{archive_dir}"
  blobCache:
    path: "{os.path.join(script_dir, 'local-cache')}"
    maxSizeBytes: 10737418240
    ttl: 168h
  signatures:
    required: false
    trustedKeys: []
  refreshInterval: 1m
  indexTTL: 10s

apis:
  hfhub:
    enabled: true
  ollama:
    enabled: true
  openai:
    enabled: true
  mlflow:
    enabled: true
"""

    config_path = os.path.join(script_dir, "local-model-server.yaml")
    with open(config_path, "w") as f:
        f.write(server_yaml)
    print(f"Created model-server configuration at: {config_path}")

    # Success output and guidance
    print("\n" + "="*50)
    print("SUCCESS: Real Model Component Built and Packaged!")
    print("="*50)
    print(f"Model ID: {model_id}")
    print(f"OCM Archive: {archive_dir}")
    print(f"Config File: {config_path}")
    print("\nTo run the model-server with this real model component:")
    print(f"  go run ./cmd/model-server/main.go -config {config_path}")
    print("\nTo test with an API request in a separate terminal:")
    print("  1. List models (HuggingFace API):")
    print("     curl http://localhost:8080/api/models")
    print("\n  2. Fetch model details:")
    print(f"     curl http://localhost:8080/api/models/{model_id}")
    print("\n  3. Download the actual config.json file via HuggingFace API:")
    print(f"     curl -L http://localhost:8080/{model_id}/resolve/main/config.json")
    print("\n  4. Download the actual model weights:")
    weight_file = "model.safetensors" if "model.safetensors" in files_to_download else "pytorch_model.bin"
    print(f"     curl -L http://localhost:8080/{model_id}/resolve/main/{weight_file}")
    print("="*50)

if __name__ == "__main__":
    main()
