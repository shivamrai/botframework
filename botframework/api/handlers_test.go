package api

import (
	"botframework/supervisor"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockEngine struct {
	health *supervisor.WorkerHealth
	err    error
}

func (m *mockEngine) Start(_ context.Context) error { return nil }
func (m *mockEngine) ProxyRequest(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusTeapot)
}
func (m *mockEngine) Stop() error { return nil }
func (m *mockEngine) Health() (*supervisor.WorkerHealth, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.health, nil
}

func TestHandleHealthMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
	rr := httptest.NewRecorder()

	h := HandleHealth(&mockEngine{health: &supervisor.WorkerHealth{Status: "ok"}})
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestHandleHealthSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()

	h := HandleHealth(&mockEngine{health: &supervisor.WorkerHealth{Status: "ok", ModelLoaded: true, Model: "qwen"}})
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Body.String(); got == "" || got[0] != '{' {
		t.Fatalf("expected json response, got %q", got)
	}
}

func TestHandleHealthFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()

	h := HandleHealth(&mockEngine{err: errors.New("boom")})
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rr.Code)
	}
}

func TestHandleModelsSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()

	h := HandleModels(&mockEngine{health: &supervisor.WorkerHealth{Status: "ok", ModelLoaded: true, Model: "qwen"}})
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Body.String(); got == "" || got[0] != '{' {
		t.Fatalf("expected json response, got %q", got)
	}
}
