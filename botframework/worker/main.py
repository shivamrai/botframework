"""Worker service entrypoint for chat completions."""
from __future__ import annotations

# pylint: disable=import-error,wrong-import-position
import argparse
import json
import os
import sys
import time
from contextlib import asynccontextmanager
from typing import Optional, Sequence, TYPE_CHECKING

import uvicorn
from fastapi import FastAPI
from fastapi.responses import StreamingResponse

# Add the parent directory to sys.path to allow imports from botframework
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from rest.schemas import (
    ChatCompletionRequest,
    ChatCompletionResponse,
    ChatCompletionResponseChoice,
    ChatCompletionUsage,
    ChatMessage,
    HealthResponse,
    LlamaMessage,
)

# Try importing llama-cpp-python

if TYPE_CHECKING:
    from llama_cpp import ChatCompletionRequestMessage
    from llama_cpp import Llama

try:
    from llama_cpp import Llama as _LlamaRuntime
except ImportError:
    _LlamaRuntime = None


# Global LLM instance (typed strictly as Llama)
llm: Optional["Llama"] = None


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Handle startup and shutdown for the FastAPI app."""
    # Startup logic
    print("🚀 Worker starting up...")
    yield
    # Shutdown logic
    print("🛑 Worker shutting down...")

app = FastAPI(title="BotFramework Worker", lifespan=lifespan)

@app.post("/v1/chat/completions")
async def chat_completions(request: ChatCompletionRequest):
    """Handle chat completion requests."""
    print(f"📥 Received request for model: {request.model}")

    if llm is None:
        # Fallback for mock mode if model failed to load or lib missing
        return mock_response(request)

    # Convert Pydantic messages to list of dicts for llama-cpp
    messages: list[LlamaMessage] = [
        {"role": m.role, "content": m.content} for m in request.messages
    ]

    if request.stream:
        return StreamingResponse(
            stream_chat_response(messages, request),
            media_type="text/event-stream"
        )
    return create_chat_response(messages, request)

def create_chat_response(
    messages: Sequence["ChatCompletionRequestMessage"],
    request: ChatCompletionRequest,
):
    """Create a non-streaming chat completion response."""
    assert llm is not None  # For type checker
    temperature = 0.7 if request.temperature is None else request.temperature
    top_k = 40 if request.top_k is None else request.top_k
    repeat_penalty = 1.1 if request.repeat_penalty is None else request.repeat_penalty
    response = llm.create_chat_completion(
        messages=messages,
        temperature=temperature,
        top_p=request.top_p,
        top_k=top_k,
        max_tokens=request.max_tokens,
        stop=request.stop,
        repeat_penalty=repeat_penalty,
        stream=False
    )
    return response

def stream_chat_response(
    messages: Sequence["ChatCompletionRequestMessage"],
    request: ChatCompletionRequest,
):
    """Stream chat completion chunks as server-sent events."""
    assert llm is not None  # For type checker
    temperature = 0.7 if request.temperature is None else request.temperature
    top_k = 40 if request.top_k is None else request.top_k
    repeat_penalty = 1.1 if request.repeat_penalty is None else request.repeat_penalty
    stream = llm.create_chat_completion(
        messages=messages,
        temperature=temperature,
        top_p=request.top_p,
        top_k=top_k,
        max_tokens=request.max_tokens,
        stop=request.stop,
        repeat_penalty=repeat_penalty,
        stream=True
    )

    for chunk in stream:
        # llama-cpp-python returns dicts that match OpenAI format
        yield f"data: {json.dumps(chunk)}\n\n"

    yield "data: [DONE]\n\n"

def mock_response(request: ChatCompletionRequest) -> ChatCompletionResponse:
    return ChatCompletionResponse(
        id="chatcmpl-mock",
        object="chat.completion",
        created=int(time.time()),
        model=request.model,
        choices=[
            ChatCompletionResponseChoice(
                index=0,
                message=ChatMessage(
                    role="assistant",
                    content=(
                        "⚠️ Mock Response (Model not loaded). You said: "
                        f"{request.messages[-1].content}"
                    ),
                ),
                finish_reason="stop",
            )
        ],
        usage=ChatCompletionUsage(
            prompt_tokens=0,
            completion_tokens=0,
            total_tokens=0,
        ),
    )


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    """Simple health check endpoint."""
    status = "ok" if llm else "mock_mode"
    return HealthResponse(status=status, model_loaded=llm is not None)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--port",
        type=int,
        default=8081,
        help="Port to run the server on",
    )
    parser.add_argument(
        "--model-path",
        type=str,
        default=None,
        help="Path to the GGUF model file",
    )
    parser.add_argument(
        "--n-gpu-layers",
        type=int,
        default=-1,
        help="Number of layers to offload to GPU (-1 for all)",
    )
    parser.add_argument(
        "--n-ctx",
        type=int,
        default=2048,
        help="Context window size",
    )

    args = parser.parse_args()

    if args.model_path and _LlamaRuntime:
        if os.path.exists(args.model_path):
            print(f"📂 Loading model from: {args.model_path}")
            try:
                llm = _LlamaRuntime(
                    model_path=args.model_path,
                    n_gpu_layers=args.n_gpu_layers,
                    n_ctx=args.n_ctx,
                    verbose=True
                )
                print("✅ Model loaded successfully!")
            except Exception as exc:  # pylint: disable=broad-exception-caught
                print(f"❌ Failed to load model: {exc}")
        else:
            print(f"❌ Model path does not exist: {args.model_path}")
    else:
        print(
            "ℹ️  No model path provided or llama-cpp-python missing. "
            "Starting in Mock Mode."
        )

    print(f"Worker starting on port {args.port}...")
    uvicorn.run(app, host="127.0.0.1", port=args.port)
