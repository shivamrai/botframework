package api

import (
	"botframework/engine"
	"encoding/json"
	"net/http"
)

type ModelListResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

func HandleHealth(workerEngine engine.InferenceEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		health, err := workerEngine.Health()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(health); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	}
}

func HandleModels(workerEngine engine.InferenceEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		health, err := workerEngine.Health()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		response := ModelListResponse{Object: "list"}
		if health.Model != "" {
			response.Data = append(response.Data, ModelInfo{
				ID:      health.Model,
				Object:  "model",
				OwnedBy: "botframework",
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	}
}
