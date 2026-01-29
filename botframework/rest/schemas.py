"""Pydantic schemas for REST API payloads."""
# pylint: disable=too-few-public-methods,import-error
import time
from typing import Dict, List, Optional, Union

from pydantic import BaseModel, Field

class ChatMessage(BaseModel):
    """Represents a single chat message."""
    role: str
    content: str

class ChatCompletionRequest(BaseModel):
    """Request body for chat completion."""
    model: str
    messages: List[ChatMessage]
    temperature: Optional[float] = 0.7
    top_p: Optional[float] = 1.0
    n: Optional[int] = 1
    max_tokens: Optional[int] = None
    stream: Optional[bool] = False
    stop: Optional[Union[str, List[str]]] = None
    presence_penalty: Optional[float] = 0.0
    frequency_penalty: Optional[float] = 0.0
    logit_bias: Optional[Dict[str, float]] = None
    user: Optional[str] = None
    # Additional parameters for llama.cpp
    top_k: Optional[int] = 40
    repeat_penalty: Optional[float] = 1.1

class ChatCompletionResponseChoice(BaseModel):
    """A single choice in a chat completion response."""
    index: int
    message: ChatMessage
    finish_reason: Optional[str] = None

class ChatCompletionUsage(BaseModel):
    """Token usage details for a completion."""
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int

class ChatCompletionResponse(BaseModel):
    """Full chat completion response."""
    id: str
    object: str = "chat.completion"
    created: int = Field(default_factory=lambda: int(time.time()))
    model: str
    choices: List[ChatCompletionResponseChoice]
    usage: ChatCompletionUsage

class ChatCompletionChunkDelta(BaseModel):
    """Delta content for a streaming chunk."""
    role: Optional[str] = None
    content: Optional[str] = None

class ChatCompletionChunkChoice(BaseModel):
    """A single choice in a streaming chunk."""
    index: int
    delta: ChatCompletionChunkDelta
    finish_reason: Optional[str] = None

class ChatCompletionChunk(BaseModel):
    """Streaming chunk response."""
    id: str
    object: str = "chat.completion.chunk"
    created: int = Field(default_factory=lambda: int(time.time()))
    model: str
    choices: List[ChatCompletionChunkChoice]
