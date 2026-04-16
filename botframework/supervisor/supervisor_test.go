package supervisor

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func extractPort(t *testing.T, serverURL string) string {
	t.Helper()

	hostPort := serverURL[len("http://"):]
	_, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("split host/port: %v", err)
	}
	return port
}

func TestCheckHealthOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	worker := NewPythonWorker("unused.py", extractPort(t, ts.URL))
	worker.HTTPClient = ts.Client()

	if err := worker.checkHealth(); err != nil {
		t.Fatalf("expected healthy, got error: %v", err)
	}
}

func TestCheckHealthNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	worker := NewPythonWorker("unused.py", extractPort(t, ts.URL))
	worker.HTTPClient = ts.Client()

	if err := worker.checkHealth(); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestWaitForHealthyEventually(t *testing.T) {
	var count atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if count.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	worker := NewPythonWorker("unused.py", extractPort(t, ts.URL))
	worker.HTTPClient = ts.Client()

	if err := worker.waitForHealthy(2 * time.Second); err != nil {
		t.Fatalf("expected eventual health success, got: %v", err)
	}
}

func TestWaitForHealthyTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	worker := NewPythonWorker("unused.py", extractPort(t, ts.URL))
	worker.HTTPClient = ts.Client()

	if err := worker.waitForHealthy(300 * time.Millisecond); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestHealthDecode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"ok","model_loaded":true,"model":"qwen.gguf"}`)
	}))
	defer ts.Close()

	worker := NewPythonWorker("unused.py", extractPort(t, ts.URL))
	worker.HTTPClient = ts.Client()

	health, err := worker.Health()
	if err != nil {
		t.Fatalf("unexpected health decode error: %v", err)
	}
	if health.Status != "ok" || !health.ModelLoaded || health.Model != "qwen.gguf" {
		t.Fatalf("unexpected health payload: %+v", health)
	}
}
