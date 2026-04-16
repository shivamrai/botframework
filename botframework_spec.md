# bot.framework – Local LLM Middleware (Draft Specification v0.1)

## 🎯 Vision & Purpose

`bot.framework` is a middleware runtime and developer API that makes **local LLM inference** as easy as calling the **OpenAI API**, but **offline**, **secure**, and **lightweight** — powered internally by `llama.cpp`.

It enables software like:

- VS Code extensions
- JetBrains plugins
- CLI tools
- Background AI assistants
- Offline chatbots
- Enterprise apps requiring private inference

…to integrate with LLMs **without needing GPU cloud services**.

---

## 🧩 Core Problem

Current llama.cpp ecosystem is fragmented:

- Apps must re-implement model loading, batching, KV cache, threads, streaming
- No unified API surface
- No structured JSON output or “tool calling”
- No developer-friendly SDK for Python or Go
- No VS Code–ready embed-and-serve module
- No middleware that feels like “OpenAI API but local”

`bot.framework` solves this.

---

## 🏗️ High-Level Architecture

\`\`\`
          ┌───────────────────────────┐
          │  VS Code / Client App     │
          │  - TypeScript SDK         │
          │  - Chat UI / features     │
          └────────────┬──────────────┘
                       │  HTTP / WS / RPC
                       │
   ┌───────────────────▼───────────────────────┐
   │         bot.framework Manager             │
   │             (Go Proxy)                    │
   │                                           │
   │ - Hardware Discovery & Profiling          │
   │ - Process Lifecycle (Start/Stop Workers)  │
   │ - Request Routing & Load Balancing        │
   │ - API Gateway (OpenAI Compatible)         │
   │                                           │
   └───────────────────┬───────────────────────┘
                       │  Internal HTTP / gRPC
                       │
   ┌───────────────────▼───────────────────────┐
   │         Inference Worker (Python)         │
   │        (FastAPI + llama.cpp)              │
   │                                           │
   │ - Model Loading (mmap)                    │
   │ - Token Streaming                         │
   │ - Context/KV Cache Management             │
   │ - Structured Output Generation            │
   │                                           │
   └───────────────────┬───────────────────────┘
                       │
               Local PC Hardware
\`\`\`

---

## 🧱 Scope – What the Middleware Provides

### 1️⃣ Unified API

**Python**
\`\`\`python
from botframework import LocalLLM
llm = LocalLLM("models/qwen.gguf")
resp = llm.chat("Rewrite this code using asyncio.")
print(resp.text)
\`\`\`

**Go**
\`\`\`go
llm := botframework.New("models/qwen.gguf")
res, _ := llm.Chat("Summarize this module", nil)
fmt.Println(res.Text)
\`\`\`

---

### 2️⃣ OpenAI-Compatible REST Endpoints

\`\`\`http
POST /v1/chat/completions
POST /v1/completions
POST /v1/embeddings
GET  /v1/models
GET  /v1/sessions
GET  /v1/health
POST /v1/cancel/{session}
\`\`\`

---

### 3️⃣ Model Runtime Layer

- llama.cpp backend
- Automatic thread/CPU selection
- Auto-detect quantization best match
- Preloading + lazy-load modes
- Supports:
  - GGUF quantized models
  - CPU-only or GPU-offload

---

### 4️⃣ Session & KV-Cache Management

Sessions allow:

- multi-turn chat
- incremental code edits
- reuse of context
- stream cancellation
- automatic trimming

---

### 5️⃣ Structured Output Modes

\`\`\`json
{
  "mode": "structured",
  "schema": {
     "type": "object",
     "properties": { "summary": { "type": "string" } }
  }
}
\`\`\`

---

### 6️⃣ Token Streaming & Cancellation

\`\`\`http
data: "token"
data: "token"
...
\`\`\`

Cancellation:
\`\`\`http
POST /v1/cancel/{session}
\`\`\`

---

### 7️⃣ Optional Systemd Service Deployment

Example:
\`\`\`
systemctl --user enable botframework
systemctl --user start botframework
journalctl --user -fu botframework
\`\`\`

The service file is optional. The daemon can also run in the foreground for local development or be spawned directly by an editor extension.

---

### 8️⃣ Language SDKs

Thin SDKs wrap the local daemon rather than embedding inference directly:

- **Python SDK:** First published SDK for notebooks, scripts, and automation
- **Go SDK:** Native integration for local services and CLIs
- **TypeScript SDK:** Editor and desktop app integration

Each SDK should provide:

- daemon discovery (`localhost` by default, overridable by config)
- typed request/response models
- streaming helpers
- health and model discovery helpers

---

## 📐 Design & Implementation Strategy

### 1️⃣ Manager-Worker Architecture (Go + Python)

To ensure robustness and scalability, we adopt a distributed system pattern locally:

- **The Manager (Go):** Acts as the "Control Plane". It handles HTTP requests, manages processes, checks hardware, and routes traffic. It starts instantly and uses minimal RAM.
- **The Worker (Python):** Acts as the "Data Plane". It loads the heavy AI libraries (`llama-cpp-python`, `torch`).
- **Benefit:** Decouples service stability from inference stability. If the Python worker crashes (OOM), the Go manager stays alive to restart it or report errors.

### 2️⃣ Smart Hardware Provisioning & Model Discovery

Instead of forcing users to pick low-level runtime parameters, the manager should combine hardware discovery with ranked model selection:

- **Phase A: Discovery:** On startup, the Go manager probes the system for CPU features, GPU vendor, VRAM, and available system RAM.
- **Phase B: Tiering:**
  - *Tier 1 (<8GB RAM):* Prefer highly quantized 3B-7B models.
  - *Tier 2 (8-16GB RAM):* Prefer 7B-13B models in Q4/Q5 ranges.
  - *Tier 3 (>16GB RAM or dedicated GPU):* Allow larger models or higher precision variants.
- **Phase C: Filtering:** The model registry is filtered to variants that fit available memory and backend constraints.
- **Phase D: Selection:** The client can choose from compatible models, while the middleware defaults to the highest-ranked model that fits comfortably.
- **Phase E: Reconfiguration:** Profiling runs on every startup and can refresh available models when hardware changes.

### 3️⃣ Three-Layer Middleware Architecture

The runtime should be shaped as a local daemon with clearly separated interfaces:

- **Core daemon:** The Go manager is the long-lived host process. It owns hardware detection, worker lifecycle, session coordination, health checks, and restart policy.
- **Public inference API:** A stable OpenAI-compatible surface exposed locally for any programming language:
  - `POST /v1/chat/completions`
  - `POST /v1/completions`
  - `POST /v1/embeddings`
  - `GET /v1/models`
  - `GET /v1/sessions`
  - `GET /v1/health`
- **Tool/control API:** A separate, explicitly opt-in local interface for trusted IDE-style actions such as reading files, listing workspaces, applying patches, and fetching diagnostics. This is intentionally deferred until after the core inference middleware reaches a stable `v1.0`.

### 4️⃣ Native Service Architecture

To balance **isolation** with **native performance**, the middleware runs as a standalone process on the host OS, not inside a container.

- **Why:** Avoids Docker virtualization overhead (especially on macOS/Metal).
- **Isolation:** Achieved via self-contained binary distribution (no system Python dependency).

### 5️⃣ Single Binary Distribution

- **Mechanism:** PyInstaller (Python) or Go compilation.
- **Benefit:** "Click-to-run" experience. No `pip install` or CUDA toolkit configuration required for end-users.
- **Pathing:** Writes config/logs to standard user data directories (e.g., `~/.config/botframework`, `~/Library/Application Support/`).

### 6️⃣ Hardware Abstraction Layer (HAL)

The runtime acts as a HAL for LLM inference, maximizing resource usage:

- **macOS:** Auto-detects Apple Silicon and loads `Metal (MPS)` backend.
- **Windows/Linux:** Auto-detects NVIDIA GPUs (`CUDA`) or falls back to `AVX/AVX2` CPU instructions.
- **Memory Mapping:** Uses `mmap` for model loading to allow OS-level paging and prevent OOM crashes.

### 7️⃣ Editor-Managed Lifecycle (LSP Model)

For VS Code/IDE integration, we adopt the Language Server Protocol lifecycle pattern:

- **Daemon Mode:** The extension spawns the `botframework` binary in the background on startup.
- **Heartbeat:** Server reports status (VRAM usage, model ready) to the editor status bar.
- **Cleanup:** Extension terminates the process on exit (configurable).

### 8️⃣ Jupyter & Notebook Integration

- **Decoupled State:** The model runs in the background server, independent of the notebook kernel.
- **Benefit:** Restarting the notebook kernel does *not* unload the model (zero-latency iteration).
- **Usage:** Standard OpenAI Python SDK pointing to `localhost` or custom `%chat` magic commands.

---

## 🧪 Testing & Validation Strategy

- Integration tests
- Stress / memory-leak detection
- Synthetic load tests
- LLM correctness smoke tests
- VS Code extension failure handling

---

## 📊 Model Scoring & Recommendation Logic

### 1. Efficiency Density Score
To calculate a "Score out of 100" for each model/precision pair, we use an **Efficiency Density** formula:

$$
Score = \left( \frac{\text{Benchmark Accuracy}}{\text{Baseline Accuracy}} \times W_a \right) + \left( \frac{\text{Memory Efficiency}}{\text{Theoretical Max}} \times W_m \right)
$$

Where:
- **$W_a$ (Accuracy Weight - 70%):** Normalized MMLU-Pro or GSM8K scores. FP16 is the 100% baseline.
- **$W_m$ (Memory Weight - 30%):** Rewards models that pack more "punch" into fewer gigabytes.
- **Penalty Factor:** Subtract points if the model requires specialized hardware (e.g., FP8 needing an H100) that the local machine lacks.

### 2. KV Cache Math (GQA)
For Grouped Query Attention (GQA) models like Llama 3, we calculate the required VRAM for context:

$$
VRAM_{KV} \approx \text{ContextLen} \times \text{Layers} \times (2 \times \text{Heads}_{kv} \times \text{HeadDim}) \times \text{BytesPerParam}
$$

**The "Smart Rule":** If the remaining VRAM after loading weights is $V_{rem}$, then:

$$
Score_{KV} = \max \left(0, \min \left(100, \frac{V_{rem}}{VRAM_{KV\_Target}} \times 100 \right) \right)
$$

### 3. Scoring Matrix
This table defines the logic for our profiling engine:

| Score Type | Logic for 100 (Most Likely) | Logic for 0 (Least Likely) |
| :--- | :--- | :--- |
| **Size Score** | Total Model Weights < 50% of available VRAM/RAM. | Weights > 100% of available VRAM/RAM. |
| **FP16 Score** | VRAM ≥ (Weights × 1.2) + 2GB buffer. | VRAM < Weights. |
| **INT8 Score** | VRAM ≥ (Weights$_{8bit} \times$ 1.2) + 1GB buffer. | VRAM < Weights$_{8bit}$. |
| **INT4 Score** | VRAM ≥ (Weights$_{4bit} \times$ 1.2) + 500MB buffer. | VRAM < Weights$_{4bit}$. |
| **KV Cache Score** | VRAM has >4GB left after loading weights. | VRAM is full after loading weights (0 context). |
| **Perf Score** | GPU Memory Bandwidth >500GB/s (e.g., RTX 3090+). | CPU-only with DDR4 RAM. |

---

## 🏁 Roadmap

| Version | Milestone |
|--------|-----------|
| v0.1 | Python worker + `/v1/chat/completions` + streaming |
| v0.2 | Go manager daemon + health-check loop + worker restart + `/v1/models` + `/v1/health` |
| v0.3 | Sessions + KV cache reuse + `/v1/sessions` + `/v1/embeddings` |
| v0.4 | Structured output (JSON schema) + function calling |
| v0.5 | Python SDK + `systemd --user` service file |
| v0.6 | Go SDK + TypeScript SDK |
| v1.0 | Published packages + docs + template repos |
| v1.1 | Tool/control API for IDE integration |

---
