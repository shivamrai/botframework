import json
import os

# This script generates the model_classification.json file used by the Go profiler.
# In a real scenario, this could crawl HuggingFace or Ollama library for metadata.

REGISTRY_PATH = "../botframework/profiler/model_classification.json"

def generate_registry():
    registry = {
        "models": [
            {
                "id": "llama-3-8b-instruct",
                "name": "Llama 3 (8B)",
                "family": "llama",
                "params_b": 8.0,
                "context_window": 8192,
                "benchmarks": {
                    "mmlu": 68.4,
                    "gsm8k": 79.6
                },
                "variants": [
                    {
                        "quant": "Q4_K_M",
                        "size_gb": 4.9,
                        "accuracy_retention": 0.98
                    },
                    {
                        "quant": "Q8_0",
                        "size_gb": 8.5,
                        "accuracy_retention": 0.999
                    },
                    {
                        "quant": "F16",
                        "size_gb": 16.0,
                        "accuracy_retention": 1.0
                    }
                ]
            },
            {
                "id": "mistral-7b-v0.3",
                "name": "Mistral (7B)",
                "family": "mistral",
                "params_b": 7.2,
                "context_window": 32768,
                "benchmarks": {
                    "mmlu": 62.5,
                    "gsm8k": 55.0
                },
                "variants": [
                    {
                        "quant": "Q4_K_M",
                        "size_gb": 4.3,
                        "accuracy_retention": 0.97
                    },
                    {
                        "quant": "Q8_0",
                        "size_gb": 7.7,
                        "accuracy_retention": 0.99
                    }
                ]
            },
            {
                "id": "phi-3-mini-4k",
                "name": "Phi-3 Mini (3.8B)",
                "family": "phi",
                "params_b": 3.8,
                "context_window": 4096,
                "benchmarks": {
                    "mmlu": 69.0,
                    "gsm8k": 82.0
                },
                "variants": [
                    {
                        "quant": "Q4_K_M",
                        "size_gb": 2.4,
                        "accuracy_retention": 0.96
                    },
                    {
                        "quant": "F16",
                        "size_gb": 7.6,
                        "accuracy_retention": 1.0
                    }
                ]
            },
             {
                "id": "gemma-2-9b",
                "name": "Gemma 2 (9B)",
                "family": "gemma",
                "params_b": 9.0,
                "context_window": 8192,
                "benchmarks": {
                    "mmlu": 71.0,
                    "gsm8k": 75.0
                },
                "variants": [
                    {
                        "quant": "Q4_K_M",
                        "size_gb": 5.4,
                        "accuracy_retention": 0.97
                    },
                    {
                        "quant": "Q8_0",
                        "size_gb": 9.8,
                        "accuracy_retention": 0.99
                    }
                ]
            }
        ]
    }

    # Ensure directory exists
    os.makedirs(os.path.dirname(REGISTRY_PATH), exist_ok=True)

    with open(REGISTRY_PATH, "w") as f:
        json.dump(registry, f, indent=2)
    
    print(f"✅ Generated model registry at {REGISTRY_PATH}")

if __name__ == "__main__":
    generate_registry()
