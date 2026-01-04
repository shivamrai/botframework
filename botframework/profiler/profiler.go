package profiler

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Tier represents the hardware capability tier
type Tier string

const (
	TierElite    Tier = "Elite"    // NVIDIA GPU > 24GB VRAM
	TierHigh     Tier = "High"     // NVIDIA GPU 8-24GB VRAM
	TierApple    Tier = "Apple"    // Apple Silicon
	TierBalanced Tier = "Balanced" // High RAM, Limited/No GPU
	TierLegacy   Tier = "Legacy"   // Low RAM, No GPU
)

// Engine represents the recommended inference backend
type Engine string

const (
	EngineVLLM      Engine = "vllm"
	EngineExLlamaV2 Engine = "exllamav2"
	EngineMLX       Engine = "mlx"
	EngineLlamaCPP  Engine = "llama_cpp"
)

type HardwareProfile struct {
	VRAM_MB      int
	SystemRAM_MB int
	HasCuda      bool
	HasMetal     bool
	ComputeCap   float64 // e.g. 8.6 for RTX 30-series
	CpuAVX512    bool
}

// DetectHardware scans the system to populate the HardwareProfile
func DetectHardware() *HardwareProfile {
	profile := &HardwareProfile{
		HasMetal: false,
		HasCuda:  false,
	}

	// 1. Detect System RAM
	profile.SystemRAM_MB = detectSystemRAM()

	// 2. Detect GPU (Metal vs CUDA)
	switch runtime.GOOS {
	case "darwin":
		// Check for Apple Silicon (Metal)
		// Simple check: uname -m returns arm64
		out, err := exec.Command("uname", "-m").Output()
		if err == nil && strings.TrimSpace(string(out)) == "arm64" {
			profile.HasMetal = true
			// On Unified Memory architecture, VRAM ~= System RAM (minus OS overhead)
			// We'll conservatively estimate 70% of system RAM is available for GPU
			profile.VRAM_MB = int(float64(profile.SystemRAM_MB) * 0.7)
		}
	case "linux", "windows":
		// Check for NVIDIA
		// nvidia-smi --query-gpu=memory.total,compute_cap --format=csv,noheader,nounits
		out, err := exec.Command("nvidia-smi", "--query-gpu=memory.total,compute_cap", "--format=csv,noheader,nounits").Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), ",")
			if len(parts) >= 2 {
				profile.HasCuda = true
				vram, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				profile.VRAM_MB = vram
				cap, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				profile.ComputeCap = cap
			}
		}
	}

	return profile
}

func detectSystemRAM() int {
	// Simplified detection logic
	// In production, use a library like 'github.com/jaypipes/ghw' or 'github.com/shirou/gopsutil'
	// This is a placeholder implementation
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			bytes, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
			return int(bytes / 1024 / 1024)
		}
	}
	// Default fallback
	return 8192 // 8GB
}

// ClassifyTier determines the hardware tier based on the profile
func (p *HardwareProfile) ClassifyTier() Tier {
	if p.HasMetal {
		return TierApple
	}

	vramGB := p.VRAM_MB / 1024
	ramGB := p.SystemRAM_MB / 1024

	if p.HasCuda {
		if vramGB >= 24 {
			return TierElite
		}
		if vramGB >= 8 {
			return TierHigh
		}
	}

	if ramGB >= 32 {
		return TierBalanced
	}

	return TierLegacy
}

// GetRecommendedEngine returns the best engine for a specific model size
func (p *HardwareProfile) GetRecommendedEngine(modelSizeGB float64) Engine {
	// 1. Apple Exception
	if p.HasMetal {
		return EngineMLX
	}

	// 2. NVIDIA Rules
	if p.HasCuda {
		vramGB := float64(p.VRAM_MB) / 1024.0
		
		// "Elite" Rule: If we have massive VRAM headroom (>20% more than model), use vLLM
		if vramGB > (modelSizeGB * 1.2) {
			return EngineVLLM
		}
		
		// "High" Rule: If it fits tightly, ExLlamaV2 is often more memory efficient/fast for single user
		if vramGB >= modelSizeGB {
			return EngineExLlamaV2
		}
	}

	// 3. Fallback (Balanced/Legacy)
	// If it doesn't fit in VRAM, or no GPU, we use llama.cpp for CPU offloading
	return EngineLlamaCPP
}

// String returns a summary of the profile
func (p *HardwareProfile) String() string {
	return fmt.Sprintf("RAM: %dMB, VRAM: %dMB, CUDA: %v, Metal: %v, Compute: %.1f", 
		p.SystemRAM_MB, p.VRAM_MB, p.HasCuda, p.HasMetal, p.ComputeCap)
}
