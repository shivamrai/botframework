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
- **Python 3.10+**
- **Docker** (Optional, for Dev Container)

## Setup Instructions

### 1. Dev Container (Recommended)
This project is configured to use a Dev Container with isolated SSH credentials.
1.  **SSH Keys**: Ensure your GitHub SSH keys are in the `localssh` directory.
    *   Required files: `id_rsa` (private key) and `id_rsa.pub` (public key).
2.  **Open in Container**: Use "Dev Containers: Reopen in Container" in VS Code.

### 2. Local Setup (Manual)

#### Python Worker
```bash
cd botframework/worker
python3 -m venv venv
source venv/bin/activate
pip install fastapi uvicorn llama-cpp-python
```
*Note: For Apple Silicon, install `llama-cpp-python` with Metal support:*
```bash
CMAKE_ARGS="-DLLAMA_METAL=on" pip install llama-cpp-python
```

#### Go Manager
```bash
cd botframework/manager
go mod init botframework
go mod tidy
go run main.go
```

## Usage
1.  Start the Manager:
    ```bash
    cd botframework/manager
    go run main.go
    ```
2.  The Manager will:
    *   Detect your hardware (RAM, GPU).
    *   Recommend the best inference engine.
    *   Start the Python worker.
    *   Serve an OpenAI-compatible API at `http://localhost:8080`.

## Development Scripts
- **Generate Model Registry**:
    ```bash
    python3 scripts/generate_model_registry.py
    ```
    Updates `botframework/profiler/model_classification.json` with the latest model data.

