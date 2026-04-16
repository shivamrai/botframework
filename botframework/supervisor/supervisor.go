package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type WorkerHealth struct {
	Status      string `json:"status"`
	ModelLoaded bool   `json:"model_loaded"`
	Model       string `json:"model"`
}

type PythonWorker struct {
	ScriptPath string
	Port       string
	Process    *exec.Cmd
	Proxy      *httputil.ReverseProxy
	HTTPClient *http.Client

	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	stopping    bool
	restarting  bool
	maxRestarts int
}

func NewPythonWorker(scriptPath, port string) *PythonWorker {
	targetURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%s", port))
	if err != nil {
		log.Fatalf("invalid worker URL: %v", err)
	}

	return &PythonWorker{
		ScriptPath:  scriptPath,
		Port:        port,
		Proxy:       httputil.NewSingleHostReverseProxy(targetURL),
		HTTPClient:  &http.Client{Timeout: 2 * time.Second},
		maxRestarts: 3,
	}
}

func resolveProjectRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

func (p *PythonWorker) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.cancel != nil {
		p.mu.Unlock()
		return errors.New("worker already started")
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.stopping = false
	p.restarting = false
	p.mu.Unlock()

	if err := p.startProcess(); err != nil {
		return err
	}

	go p.monitorProcess()
	return nil
}

func (p *PythonWorker) startProcess() error {
	fmt.Printf("🚀 Starting Python Engine: %s on port %s\n", p.ScriptPath, p.Port)

	p.mu.RLock()
	ctx := p.ctx
	p.mu.RUnlock()

	if configuredPython := os.Getenv("BOTFRAMEWORK_PYTHON"); configuredPython != "" {
		fmt.Printf("🐍 Using BOTFRAMEWORK_PYTHON=%s\n", configuredPython)
		p.Process = exec.CommandContext(ctx, configuredPython, p.ScriptPath, "--port", p.Port)
	} else if _, err := exec.LookPath("pipenv"); err == nil {
		fmt.Println("🐍 Using pipenv-managed Python environment")
		p.Process = exec.CommandContext(ctx, "pipenv", "run", "python", p.ScriptPath, "--port", p.Port)
	} else {
		fmt.Println("🐍 Using system python3")
		p.Process = exec.CommandContext(ctx, "python3", p.ScriptPath, "--port", p.Port)
	}
	p.Process.Dir = resolveProjectRoot()
	p.Process.Stdout = os.Stdout
	p.Process.Stderr = os.Stderr

	if err := p.Process.Start(); err != nil {
		return fmt.Errorf("failed to start python process: %w", err)
	}

	fmt.Println("⏳ Waiting for worker to initialize...")
	if err := p.waitForHealthy(30 * time.Second); err != nil {
		_ = p.Process.Process.Kill()
		return err
	}
	fmt.Println("✅ Worker is ready!")
	p.mu.Lock()
	p.restarting = false
	p.mu.Unlock()

	return nil
}

func (p *PythonWorker) waitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := p.checkHealth(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("worker failed health check within %s", timeout)
}

func (p *PythonWorker) checkHealth() error {
	resp, err := p.HTTPClient.Get(fmt.Sprintf("http://127.0.0.1:%s/health", p.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func (p *PythonWorker) monitorProcess() {
	for attempt := 0; ; attempt++ {
		p.mu.RLock()
		process := p.Process
		stopping := p.stopping
		p.mu.RUnlock()

		if process == nil || stopping {
			return
		}

		err := process.Wait()

		p.mu.RLock()
		stopping = p.stopping
		ctxErr := p.ctx.Err()
		p.mu.RUnlock()

		if stopping || ctxErr != nil {
			return
		}

		log.Printf("worker exited unexpectedly: %v", err)
		if attempt >= p.maxRestarts {
			log.Printf("worker restart limit reached (%d attempts)", p.maxRestarts)
			return
		}

		backoff := time.Duration(1<<attempt) * time.Second
		log.Printf("restarting worker in %s (attempt %d/%d)", backoff, attempt+1, p.maxRestarts)
		time.Sleep(backoff)

		p.mu.Lock()
		p.restarting = true
		p.mu.Unlock()

		if err := p.startProcess(); err != nil {
			log.Printf("worker restart failed: %v", err)
			continue
		}

		attempt = -1
	}
}

func (p *PythonWorker) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	p.Proxy.ServeHTTP(w, r)
}

func (p *PythonWorker) Health() (*WorkerHealth, error) {
	resp, err := p.HTTPClient.Get(fmt.Sprintf("http://127.0.0.1:%s/health", p.Port))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("worker health returned status %d", resp.StatusCode)
	}

	var health WorkerHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}

func (p *PythonWorker) Stop() error {
	p.mu.Lock()
	p.stopping = true
	process := p.Process
	cancel := p.cancel
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if process != nil && process.Process != nil {
		fmt.Println("🛑 Stopping Python Engine...")
		if err := process.Process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return process.Process.Kill()
		}
		_, err := process.Process.Wait()
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
	}
	return nil
}
