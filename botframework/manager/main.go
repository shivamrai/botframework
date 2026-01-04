package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"time"
	"botframework/profiler"
)

// 1. Define the Interface (The "Contract")
type InferenceEngine interface {
	Start(ctx context.Context) error
	// ProxyRequest forwards the HTTP request to the underlying worker
	ProxyRequest(w http.ResponseWriter, r *http.Request)
	Stop() error
}

// 2. Implementation: PythonWorker (Wraps a Python/FastAPI process)
type PythonWorker struct {
	ScriptPath string
	Port       string
	Process    *exec.Cmd
	Proxy      *httputil.ReverseProxy
}

func NewPythonWorker(scriptPath string, port string) *PythonWorker {
	// The worker will run on localhost at the specified port
	targetURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%s", port))
	if err != nil {
		log.Fatalf("Invalid worker URL: %v", err)
	}

	return &PythonWorker{
		ScriptPath: scriptPath,
		Port:       port,
		Proxy:      httputil.NewSingleHostReverseProxy(targetURL),
	}
}

func (p *PythonWorker) Start(ctx context.Context) error {
	fmt.Printf("🚀 Starting Python Engine: %s on port %s\n", p.ScriptPath, p.Port)

	// In a real app, we would use the specific venv python executable
	// For now, we assume 'python3' is available in the path
	p.Process = exec.CommandContext(ctx, "python3", p.ScriptPath, "--port", p.Port)
	
	// Pipe stdout/stderr to the parent process for debugging
	p.Process.Stdout = os.Stdout
	p.Process.Stderr = os.Stderr

	if err := p.Process.Start(); err != nil {
		return fmt.Errorf("failed to start python process: %w", err)
	}

	// TODO: Implement a proper healthcheck loop (ping /health) instead of sleep
	fmt.Println("⏳ Waiting for worker to initialize...")
	time.Sleep(2 * time.Second)
	fmt.Println("✅ Worker is ready!")

	return nil
}

func (p *PythonWorker) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	// The ReverseProxy handles the streaming of the response body automatically
	p.Proxy.ServeHTTP(w, r)
}

func (p *PythonWorker) Stop() error {
	if p.Process != nil && p.Process.Process != nil {
		fmt.Println("🛑 Stopping Python Engine...")
		return p.Process.Process.Kill()
	}
	return nil
}

// 3. The Orchestrator (The "Smart" part)
type ModelManager struct {
	Engine InferenceEngine
}

func NewSmartManager() *ModelManager {
	// 1. Run Hardware Profiling
	fmt.Println("🔍 Scanning Hardware...")
	profile := profiler.DetectHardware()
	fmt.Printf("📊 Hardware Profile: %s\n", profile.String())

	tier := profile.ClassifyTier()
	fmt.Printf("🏷️  System Tier: %s\n", tier)

	// 2. Select Engine based on a hypothetical model size (e.g., 7B Q4 ~ 5.5GB)
	// In the future, this will come from the user's selected model in the UI
	targetModelSizeGB := 5.5 
	recommendedEngine := profile.GetRecommendedEngine(targetModelSizeGB)
	fmt.Printf("⚙️  Recommended Engine: %s\n", recommendedEngine)

	var selectedEngine InferenceEngine

	// Path to the worker script
	workerScript := "botframework/worker/main.py"

	// 3. Provision the correct worker
	switch recommendedEngine {
	case profiler.EngineMLX:
		fmt.Println("🍎 Starting MLX Backend (Apple Silicon)")
		// In reality, we might pass a flag like --backend=mlx to the python script
		selectedEngine = NewPythonWorker(workerScript, "8081")
	case profiler.EngineVLLM:
		fmt.Println("🚀 Starting vLLM Backend (High Performance)")
		selectedEngine = NewPythonWorker(workerScript, "8081")
	case profiler.EngineExLlamaV2:
		fmt.Println("⚡ Starting ExLlamaV2 Backend")
		selectedEngine = NewPythonWorker(workerScript, "8081")
	default:
		fmt.Println("🐢 Starting llama.cpp Backend (Universal/CPU)")
		selectedEngine = NewPythonWorker(workerScript, "8081")
	}

	return &ModelManager{Engine: selectedEngine}
}

func main() {
	// Create a context that we can cancel to stop the worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize our smart backend
	manager := NewSmartManager()

	// Lifecycle: Start the engine
	err := manager.Engine.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}
	
	// Ensure we stop the worker when the manager exits
	defer func() {
		if err := manager.Engine.Stop(); err != nil {
			log.Printf("Error stopping engine: %v", err)
		}
	}()

	// Set up the HTTP server for the Manager
	// This acts as the API Gateway
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		fmt.Printf("📥 Request: %s %s\n", r.Method, r.URL.Path)
		
		// Forward everything to the worker
		manager.Engine.ProxyRequest(w, r)
	})

	port := "8080"
	fmt.Printf("🌟 BotFramework Manager listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
