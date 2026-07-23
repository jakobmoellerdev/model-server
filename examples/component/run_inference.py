#!/usr/bin/env python3
import os
import sys
import time
import json
import socket
import subprocess
import urllib.request

# Set global socket timeout to 300 seconds to prevent any client-side timeouts
socket.setdefaulttimeout(300)

# Ensure we are using our virtual environment if available
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(os.path.dirname(script_dir))

def main():
    config_file = sys.argv[2] if len(sys.argv) > 2 else "local-model-server.yaml"
    if os.path.isabs(config_file) or "/" in config_file:
        config_path = config_file
    else:
        config_path = os.path.join(script_dir, config_file)
    
    # 1. Build model-server binary (skip if already built)
    binary_path = os.path.join(project_root, "model-server")
    if not os.path.exists(binary_path):
        print("=== Building model-server binary ===")
        subprocess.run(["go", "build", "-o", "model-server", "./cmd/model-server/main.go"], cwd=project_root, check=True)
    else:
        print("=== model-server binary exists, skipping build ===")

    # 2. Start model-server in the background
    print("\n=== Starting model-server ===")
    env = os.environ.copy()
    env["GODEBUG"] = "netdns=go"
    
    # Authenticate via temporary DOCKER_CONFIG if credentials are provided in env
    ghcr_username = os.environ.get("GHCR_USERNAME")
    ghcr_token = os.environ.get("GHCR_TOKEN") or os.environ.get("GHCR_PASSWORD")
    if ghcr_username and ghcr_token:
        print("Detected GHCR credentials. Creating temporary authenticated DOCKER_CONFIG...")
        import base64
        auth_str = f"{ghcr_username}:{ghcr_token}"
        auth_b64 = base64.b64encode(auth_str.encode()).decode()
        temp_docker_dir = os.path.join(script_dir, "temp_docker_config")
        os.makedirs(temp_docker_dir, exist_ok=True)
        docker_config_data = {
            "auths": {
                "ghcr.io": {
                    "auth": auth_b64
                }
            }
        }
        with open(os.path.join(temp_docker_dir, "config.json"), "w") as f:
            json.dump(docker_config_data, f)
        env["DOCKER_CONFIG"] = temp_docker_dir
        print("DOCKER_CONFIG configured to use plaintext credentials.")

    server_process = subprocess.Popen(
        ["./model-server", "-config", config_path],
        cwd=project_root,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        env=env,
        text=True
    )
    
    # Wait for readyz
    print("Waiting for model-server to be ready...")
    for _ in range(15):
        try:
            with urllib.request.urlopen("http://localhost:8080/readyz") as resp:
                if resp.status == 200:
                    print("model-server is ready!")
                    break
        except Exception:
            pass
        time.sleep(1)
    else:
        print("Error: model-server failed to start.", file=sys.stderr)
        server_process.terminate()
        sys.exit(1)

    # 3. Configure HF_ENDPOINT and HF_HUB_TIMEOUT
    print("\n=== Configuring HF_ENDPOINT to local model-server ===")
    os.environ["HF_ENDPOINT"] = "http://localhost:8080"
    os.environ["HF_HUB_TIMEOUT"] = "300"
    os.environ["HF_HUB_DOWNLOAD_TIMEOUT"] = "300"
    
    # Clear HF cache for this model to force it to download from our server (only if FORCE_REDOWNLOAD=1)
    model_id = sys.argv[1] if len(sys.argv) > 1 else "arnir0/Tiny-LLM"
    cache_dir = os.path.expanduser(f"~/.cache/huggingface/hub/models--{model_id.replace('/', '--')}")
    if os.path.exists(cache_dir) and os.environ.get("FORCE_REDOWNLOAD") in ("1", "true"):
        print(f"Clearing local HF cache to force download through proxy: {cache_dir}")
        import shutil
        shutil.rmtree(cache_dir)

    try:
        # Import transformers (must be done after environment variables are set)
        print("\n=== Importing Hugging Face Transformers ===")
        from transformers import AutoTokenizer, AutoModelForCausalLM
        import torch

        # 4. Load the Model and Tokenizer from model-server
        print(f"\n=== Loading {model_id} (fetching files through model-server proxy) ===")
        tokenizer = AutoTokenizer.from_pretrained(model_id, revision="1.0.0")
        model = AutoModelForCausalLM.from_pretrained(model_id, revision="1.0.0", dtype=torch.float16)

        # 5. Run inference
        print("\n=== Running Model Inference ===")
        prompt = "Once upon a time, there was a tiny LLM"
        print(f"Prompt: {prompt}")
        
        inputs = tokenizer(prompt, return_tensors="pt")
        
        # Generate text
        with torch.no_grad():
            outputs = model.generate(
                **inputs,
                max_new_tokens=40,
                do_sample=True,
                temperature=0.8,
                top_k=50,
                top_p=0.9,
                pad_token_id=tokenizer.eos_token_id
            )
            
        generated_text = tokenizer.decode(outputs[0], skip_special_tokens=True)
        print("\n" + "="*50)
        print("GENERATED INFERENCE RESULT:")
        print("="*50)
        print(generated_text)
        print("="*50)

    except Exception as e:
        print(f"\nError running inference: {e}", file=sys.stderr)
    finally:
        # 6. Shutdown server and print model-server console logs
        print("\n=== Shutting down model-server ===")
        server_process.terminate()
        try:
            stdout, _ = server_process.communicate(timeout=5)
            print("\nmodel-server console output:")
            print("-"*50)
            print(stdout)
            print("-"*50)
        except Exception:
            pass
            
        # Clean up temporary DOCKER_CONFIG directory
        temp_docker_dir = os.path.join(script_dir, "temp_docker_config")
        if os.path.exists(temp_docker_dir):
            import shutil
            shutil.rmtree(temp_docker_dir)

if __name__ == "__main__":
    main()
