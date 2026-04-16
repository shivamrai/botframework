package engine

import (
	"botframework/profiler"
	"botframework/supervisor"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
)

type InferenceEngine interface {
	Start(ctx context.Context) error
	ProxyRequest(w http.ResponseWriter, r *http.Request)
	Health() (*supervisor.WorkerHealth, error)
	Stop() error
}

type ModelManager struct {
	Engine InferenceEngine
}

func resolveWorkerScript() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("..", "worker", "main.py")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "worker", "main.py")
}

func NewSmartManager() *ModelManager {
	fmt.Println("🔍 Scanning Hardware...")
	profile := profiler.DetectHardware()
	fmt.Printf("📊 Hardware Profile: %s\n", profile.String())

	tier := profile.ClassifyTier()
	fmt.Printf("🏷️  System Tier: %s\n", tier)

	targetModelSizeGB := 5.5
	recommendedEngine := profile.GetRecommendedEngine(targetModelSizeGB)
	fmt.Printf("⚙️  Recommended Engine: %s\n", recommendedEngine)

	workerScript := resolveWorkerScript()
	return NewManagerForEngine(workerScript, "8081", recommendedEngine)
}

func NewManagerForEngine(workerScript, port string, recommendedEngine profiler.Engine) *ModelManager {
	var selectedEngine InferenceEngine

	switch recommendedEngine {
	case profiler.EngineMLX:
		fmt.Println("🍎 Starting MLX Backend (Apple Silicon)")
		selectedEngine = supervisor.NewPythonWorker(workerScript, port)
	case profiler.EngineVLLM:
		fmt.Println("🚀 Starting vLLM Backend (High Performance)")
		selectedEngine = supervisor.NewPythonWorker(workerScript, port)
	case profiler.EngineExLlamaV2:
		fmt.Println("⚡ Starting ExLlamaV2 Backend")
		selectedEngine = supervisor.NewPythonWorker(workerScript, port)
	default:
		fmt.Println("🐢 Starting llama.cpp Backend (Universal/CPU)")
		selectedEngine = supervisor.NewPythonWorker(workerScript, port)
	}

	return &ModelManager{Engine: selectedEngine}
}
