# BotFramework Project

## Overview
BotFramework is a local LLM middleware that provides an OpenAI-compatible API for running inference on consumer hardware. It uses a **Manager-Worker** architecture where a Go process manages hardware profiling and lifecycle, while a Python worker handles the actual inference using `llama.cpp`.

## Project Structure
```
/
├── ideapad-vscode/       # Client: VS Code Extension
├── botframework/         # Middleware Core
│   ├── setup/            # Hardware detection & installation scripts
│   ├── manager/          # Go: Process manager & API Gateway
│   ├── worker/           # Python: Inference engine (FastAPI + llama.cpp)
│   ├── profiler/         # Go: Hardware profiling & Model Scoring logic
│   └── rest/             # API definitions
└── scripts/              # Utility scripts (e.g., registry generator)
```

## Prerequisites
- **Go 1.21+**
- **Python 3.12+**
- **pipenv**

## Setup Instructions

### Open The Project
You can open the repository directly as a folder, or use the workspace file to keep editor settings scoped to this project:

```bash
code botframework.code-workspace
```

The workspace file is optional.

### 1. Native Bootstrap (Recommended)
BotFramework is designed to probe the host machine directly and choose the right inference backend for that hardware. The recommended development path is native execution with Go on the host and Python isolated through `pipenv`.

**Bootstrap the repo:**
```bash
./up
```

The bootstrap script installs `pipenv` if needed, detects the available acceleration backend, and installs the Python dependencies with the correct `CMAKE_ARGS` for `llama-cpp-python`.

### 2. Manual Setup

#### Python Environment
```bash
python3.12 -m pip install --user pipenv
CMAKE_ARGS="-DLLAMA_HIPBLAS=on" pipenv install --python python3.12
```

Use the `CMAKE_ARGS` that matches your host hardware:
- ROCm: `-DLLAMA_HIPBLAS=on`
- NVIDIA: `-DLLAMA_CUBLAS=on`
- Apple Silicon: `-DLLAMA_METAL=on`
- CPU only: `-DLLAMA_BLAS=off`

#### Go Manager
```bash
cd botframework
go mod tidy
go run ./manager
```

## Usage
1.  Start the Manager:
    ```bash
    cd botframework
    go run ./manager
    ```
2.  The Manager will:
    *   Detect your hardware (RAM, GPU).
    *   Recommend the best inference engine.
    *   Start the Python worker by preferring the project `pipenv` environment.
    *   Serve an OpenAI-compatible API at `http://localhost:8080`.

## Development Scripts
- **Generate Model Registry**:
    ```bash
    python3 scripts/generate_model_registry.py
    ```
    Updates `botframework/profiler/model_classification.json` with the latest model data.
