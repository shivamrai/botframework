package engine

import (
	"botframework/profiler"
	"botframework/supervisor"
	"testing"
)

func TestPythonWorkerSatisfiesInferenceEngine(t *testing.T) {
	var _ InferenceEngine = (*supervisor.PythonWorker)(nil)
}

func TestNewManagerForEngineCreatesEngine(t *testing.T) {
	tests := []struct {
		name   string
		engine profiler.Engine
	}{
		{name: "mlx", engine: profiler.EngineMLX},
		{name: "vllm", engine: profiler.EngineVLLM},
		{name: "exllama", engine: profiler.EngineExLlamaV2},
		{name: "default_llama_cpp", engine: profiler.EngineLlamaCPP},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewManagerForEngine("/tmp/fake_worker.py", "9001", tc.engine)
			if mgr == nil || mgr.Engine == nil {
				t.Fatal("expected non-nil manager and engine")
			}
			if _, ok := mgr.Engine.(*supervisor.PythonWorker); !ok {
				t.Fatalf("expected *supervisor.PythonWorker, got %T", mgr.Engine)
			}
		})
	}
}
