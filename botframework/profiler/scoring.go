package profiler

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
)

// ModelRegistry represents the JSON structure of available models
type ModelRegistry struct {
	Models []Model `json:"models"`
}

type Model struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Family        string    `json:"family"`
	ParamsB       float64   `json:"params_b"`
	ContextWindow int       `json:"context_window"`
	Benchmarks    Benchmarks `json:"benchmarks"`
	Variants      []Variant `json:"variants"`
}

type Benchmarks struct {
	MMLU  float64 `json:"mmlu"`
	GSM8K float64 `json:"gsm8k"`
}

type Variant struct {
	Quant             string  `json:"quant"`
	SizeGB            float64 `json:"size_gb"`
	AccuracyRetention float64 `json:"accuracy_retention"`
}

// ScoredVariant wraps a variant with its calculated score
type ScoredVariant struct {
	ModelID   string
	ModelName string
	Variant   Variant
	Score     float64
	Reason    string
}

// LoadRegistry reads the model classification JSON
func LoadRegistry(path string) (*ModelRegistry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var registry ModelRegistry
	if err := json.Unmarshal(bytes, &registry); err != nil {
		return nil, err
	}

	return &registry, nil
}

// RecommendModels ranks models based on the hardware profile
func (p *HardwareProfile) RecommendModels(registry *ModelRegistry) []ScoredVariant {
	var recommendations []ScoredVariant

	for _, model := range registry.Models {
		for _, variant := range model.Variants {
			score, reason := p.CalculateScore(model, variant)
			if score > 0 {
				recommendations = append(recommendations, ScoredVariant{
					ModelID:   model.ID,
					ModelName: model.Name,
					Variant:   variant,
					Score:     score,
					Reason:    reason,
				})
			}
		}
	}

	// Sort by Score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// CalculateScore implements the scoring logic defined in the spec
func (p *HardwareProfile) CalculateScore(model Model, variant Variant) (float64, string) {
	// 1. Size Score (Can we even load it?)
	// Available memory for model (leaving buffer for OS)
	// If Metal, we use VRAM (which is shared RAM). If CUDA, VRAM.
	// If CPU only (Legacy), we use System RAM.
	
	availableMemGB := float64(p.VRAM_MB) / 1024.0
	if !p.HasCuda && !p.HasMetal {
		// Fallback to System RAM for CPU inference
		availableMemGB = float64(p.SystemRAM_MB) / 1024.0
	}

	// Buffer: 2GB for OS/Display
	safeMemGB := availableMemGB - 2.0
	if safeMemGB < 0 {
		safeMemGB = 0.5 // Minimal fallback
	}

	// Hard cutoff: If model is bigger than available memory, score 0
	if variant.SizeGB > availableMemGB {
		return 0, "Insufficient Memory"
	}

	// 2. Efficiency Density Score
	// Score = (Benchmark / Baseline * Wa) + (MemEfficiency * Wm)
	// Simplified for this implementation:
	// Base Score = Benchmark MMLU (normalized to 0-100 scale roughly) * AccuracyRetention
	baseScore := model.Benchmarks.MMLU * variant.AccuracyRetention

	// 3. Memory Fit Bonus/Penalty
	// If it fits comfortably (leaving room for KV cache), boost score.
	// If it fits tightly, penalize.
	
	// KV Cache estimation (simplified from spec formula for 4k context)
	// VRAM_KV approx 0.5GB for 7B model at 4k context (very rough estimate)
	kvCacheEstGB := 0.5 
	if model.ParamsB > 10 {
		kvCacheEstGB = 1.0
	}

	remainingHeadroom := safeMemGB - variant.SizeGB - kvCacheEstGB
	
	memoryScore := 0.0
	if remainingHeadroom > 2.0 {
		// Lots of room, great for long context
		memoryScore = 20.0 
	} else if remainingHeadroom > 0.5 {
		// Fits okay
		memoryScore = 10.0
	} else {
		// Very tight, risk of OOM
		memoryScore = -30.0
	}

	// 4. Hardware Specific Bonuses
	hwBonus := 0.0
	if p.HasMetal && variant.Quant == "Q4_K_M" {
		// Apple Silicon loves Q4_K_M
		hwBonus += 10.0
	}
	if p.HasCuda && variant.Quant == "Q8_0" && remainingHeadroom > 4.0 {
		// CUDA with lots of VRAM handles INT8 well
		hwBonus += 5.0
	}

	finalScore := baseScore + memoryScore + hwBonus

	// Cap at 100, min 0
	finalScore = math.Min(100, math.Max(0, finalScore))

	reason := fmt.Sprintf("Base: %.1f, MemBonus: %.1f, HWBonus: %.1f (Headroom: %.1fGB)", 
		baseScore, memoryScore, hwBonus, remainingHeadroom)

	return finalScore, reason
}
