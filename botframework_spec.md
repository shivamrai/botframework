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
POST /v1/models/list
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
systemctl enable bot.framework
systemctl start bot.framework
journalctl -fu bot.framework
\`\`\`

---

## 📐 Design & Implementation Strategy

### 1️⃣ Manager-Worker Architecture (Go + Python)

To ensure robustness and scalability, we adopt a distributed system pattern locally:

- **The Manager (Go):** Acts as the "Control Plane". It handles HTTP requests, manages processes, checks hardware, and routes traffic. It starts instantly and uses minimal RAM.
- **The Worker (Python):** Acts as the "Data Plane". It loads the heavy AI libraries (`llama-cpp-python`, `torch`).
- **Benefit:** Decouples service stability from inference stability. If the Python worker crashes (OOM), the Go manager stays alive to restart it or report errors.

### 2️⃣ Smart Hardware Provisioning (Sliding Window)

Instead of forcing users to pick model parameters, the system auto-configures based on a "Weights & Rank" logic:

- **Phase A: Discovery:** On startup, the Go Manager probes the system (CPU AVX support, GPU Vendor, Total VRAM).
- **Phase B: Tiers:**
  - *Tier 1 (<8GB RAM):* Selects highly quantized 3B-7B models.
  - *Tier 2 (8-16GB RAM):* Selects 7B-13B models (Q4/Q5).
  - *Tier 3 (>16GB + GPU):* Selects larger models or higher precision.
- **Phase C: Dynamic Re-configuration:** The Manager re-runs profiling on every startup. If hardware changes (e.g., eGPU added), it triggers a re-provisioning flow.

### 1️⃣ Manager-Worker Architecture (Go + Python)

To ensure robustness and scalability, we adopt a distributed system pattern locally:

- **The Manager (Go):** Acts as the "Control Plane". It handles HTTP requests, manages processes, checks hardware, and routes traffic. It starts instantly and uses minimal RAM.
- **The Worker (Python):** Acts as the "Data Plane". It loads the heavy AI libraries (`llama-cpp-python`, `torch`).
- **Benefit:** Decouples service stability from inference stability. If the Python worker crashes (OOM), the Go manager stays alive to restart it or report errors.

### 2️⃣ Smart Model Discovery & Selection

Instead of forcing a single "best" model, the system empowers the user with smart defaults:

- **Phase A: Discovery:** On startup, the Go Manager probes the system (CPU AVX support, GPU Vendor, Total VRAM).
- **Phase B: Filtering:** The system generates a list of *compatible* models from the registry.
  - *Example:* If VRAM is 8GB, it filters out 70B models but highlights 7B (Q4) and 13B (Q2).
- **Phase C: User Selection:** The user selects their preferred model via the client UI (VS Code). The system defaults to the highest-ranked model that fits comfortably in VRAM.
- **Phase D: Dynamic Re-configuration:** If hardware changes (e.g., eGPU added), the available model list is automatically refreshed.

### 3️⃣ Native Service Architecture (Strategy #2)

To balance **isolation** with **native performance**, the middleware runs as a standalone process on the host OS, not inside a container.

- **Why:** Avoids Docker virtualization overhead (especially on macOS/Metal).
- **Isolation:** Achieved via self-contained binary distribution (no system Python dependency).

### 2️⃣ Single Binary Distribution

- **Mechanism:** PyInstaller (Python) or Go compilation.
- **Benefit:** "Click-to-run" experience. No `pip install` or CUDA toolkit configuration required for end-users.
- **Pathing:** Writes config/logs to standard user data directories (e.g., `~/.config/botframework`, `~/Library/Application Support/`).

### 3️⃣ Hardware Abstraction Layer (HAL)

The runtime acts as a HAL for LLM inference, maximizing resource usage:

- **macOS:** Auto-detects Apple Silicon and loads `Metal (MPS)` backend.
- **Windows/Linux:** Auto-detects NVIDIA GPUs (`CUDA`) or falls back to `AVX/AVX2` CPU instructions.
- **Memory Mapping:** Uses `mmap` for model loading to allow OS-level paging and prevent OOM crashes.

### 4️⃣ Editor-Managed Lifecycle (LSP Model)

For VS Code/IDE integration, we adopt the Language Server Protocol lifecycle pattern:

- **Daemon Mode:** The extension spawns the `botframework` binary in the background on startup.
- **Heartbeat:** Server reports status (VRAM usage, model ready) to the editor status bar.
- **Cleanup:** Extension terminates the process on exit (configurable).

### 5️⃣ Jupyter & Notebook Integration

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
| v0.1 | Local Python server + `/v1/chat` + streaming |
| v0.2 | Python client + VS Code demo extension |
| v0.3 | Go backend + OpenAI compatible endpoints |
| v0.4 | Function calling + JSON mode |
| v1.0 | Published library + docs + template repos |

---
