//go:build integration

package main

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const managerBaseURL = "http://127.0.0.1:8080"

func TestManagerIntegrationHealthAndModels(t *testing.T) {
	if os.Getenv("BOTFRAMEWORK_RUN_INTEGRATION") != "1" {
		t.Skip("set BOTFRAMEWORK_RUN_INTEGRATION=1 to run integration tests")
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine current file")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./manager")
	cmd.Dir = moduleRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start manager: %v", err)
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	if err := waitForHTTP200(client, managerBaseURL+"/v1/health", 25*time.Second); err != nil {
		t.Fatalf("health endpoint did not become ready: %v", err)
	}
	if err := waitForHTTP200(client, managerBaseURL+"/v1/models", 10*time.Second); err != nil {
		t.Fatalf("models endpoint did not become ready: %v", err)
	}
}

func waitForHTTP200(client *http.Client, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
