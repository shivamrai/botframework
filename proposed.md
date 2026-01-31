# Proposals from recent prompts

- Keep a single FastAPI worker process for local VS Code/Jupyter usage to avoid duplicating model memory.
- Use llama.cpp CPU threads for speedups on CPU inference; keep Uvicorn workers at 1.
- Add a simple request queue or serialization to prevent concurrent model calls in a single worker.
- Use goroutines in the Go manager for worker lifecycle tasks (health checks, restarts, log streaming, and timeouts), not for inference speedups.
