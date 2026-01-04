import argparse
import uvicorn
import sys
import os
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import StreamingResponse
from contextlib import asynccontextmanager
from typing import Optional

# Add the parent directory to sys.path to allow imports from botframework
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from rest.schemas import ChatCompletionRequest, ChatCompletionResponse, ChatCompletionChunk

# Try importing llama-cpp-python
try:
    from llama_cpp import Llama
except ImportError:
    Llama = None
    print("⚠️  llama-cpp-python not installed. Running in mock mode.")

# Global LLM instance
llm: Optional[Llama] = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup logic
    print("🚀 Worker starting up...")
    yield
    # Shutdown logic
    print("🛑 Worker shutting down...")

app = FastAPI(title="BotFramework Worker", lifespan=lifespan)

@app.post("/v1/chat/completions")
async def chat_completions(request: ChatCompletionRequest):
    global llm
    
    print(f"📥 Received request for model: {request.model}")

    if llm is None:
        # Fallback for mock mode if model failed to load or lib missing
        return mock_response(request)

    # Convert Pydantic messages to list of dicts for llama-cpp
    messages = [{"role": m.role, "content": m.content} for m in request.messages]

    if request.stream:
        return StreamingResponse(
            stream_chat_response(messages, request),
            media_type="text/event-stream"
        )
    else:
        return create_chat_response(messages, request)

def create_chat_response(messages, request: ChatCompletionRequest):
    response = llm.create_chat_completion(
        messages=messages,
        temperature=request.temperature,
        top_p=request.top_p,
        top_k=request.top_k,
        max_tokens=request.max_tokens,
        stop=request.stop,
        repeat_penalty=request.repeat_penalty,
        stream=False
    )
    return response

def stream_chat_response(messages, request: ChatCompletionRequest):
    stream = llm.create_chat_completion(
        messages=messages,
        temperature=request.temperature,
        top_p=request.top_p,
        top_k=request.top_k,
        max_tokens=request.max_tokens,
        stop=request.stop,
        repeat_penalty=request.repeat_penalty,
        stream=True
    )
    
    for chunk in stream:
        # llama-cpp-python returns dicts that match OpenAI format
        import json
        yield f"data: {json.dumps(chunk)}\n\n"
    
    yield "data: [DONE]\n\n"

def mock_response(request: ChatCompletionRequest):
    import time
    return {
        "id": "chatcmpl-mock",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": request.model,
        "choices": [{
            "index": 0,
            "message": {
                "role": "assistant",
                "content": f"⚠️ Mock Response (Model not loaded). You said: {request.messages[-1].content}"
            },
            "finish_reason": "stop"
        }],
        "usage": {
            "prompt_tokens": 0,
            "completion_tokens": 0,
            "total_tokens": 0
        }
    }

@app.get("/health")
async def health():
    status = "ok" if llm else "mock_mode"
    return {"status": status, "model_loaded": llm is not None}

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=8081, help="Port to run the server on")
    parser.add_argument("--model-path", type=str, default=None, help="Path to the GGUF model file")
    parser.add_argument("--n-gpu-layers", type=int, default=-1, help="Number of layers to offload to GPU (-1 for all)")
    parser.add_argument("--n-ctx", type=int, default=2048, help="Context window size")
    
    args = parser.parse_args()

    if args.model_path and Llama:
        if os.path.exists(args.model_path):
            print(f"📂 Loading model from: {args.model_path}")
            try:
                llm = Llama(
                    model_path=args.model_path,
                    n_gpu_layers=args.n_gpu_layers,
                    n_ctx=args.n_ctx,
                    verbose=True
                )
                print("✅ Model loaded successfully!")
            except Exception as e:
                print(f"❌ Failed to load model: {e}")
        else:
            print(f"❌ Model path does not exist: {args.model_path}")
    else:
        print("ℹ️  No model path provided or llama-cpp-python missing. Starting in Mock Mode.")

    print(f"Worker starting on port {args.port}...")
    uvicorn.run(app, host="127.0.0.1", port=args.port)
