package models

import (
	"fmt"
	"strings"
)

// ModelInfo contains metadata about an LLM model
type ModelInfo struct {
	ID              string   // Unique identifier (e.g., "hunyuan-mt-7b")
	Name            string   // Human-readable name
	Description     string   // Description of model capabilities
	Parameters      uint64   // Number of parameters (e.g., 7_000_000_000 for 7B)
	MinRAM          uint64   // Minimum RAM in bytes (for Q4 quantization)
	RecommendedRAM  uint64   // Recommended RAM in bytes (for Q8 quantization)
	QuantType       string   // Quantization type (Q4, Q8, F16, etc.)
	SourceURL       string   // HuggingFace or other source URL
	Languages       []string // Supported languages
	OptimizedFor    string   // What this model is optimized for
	Quality         string   // Quality rating: excellent, good, moderate
	LicenseType     string   // License (Apache-2.0, MIT, etc.)
	RequiresGPU     bool     // Whether GPU is required
	ContextLength   int      // Maximum context length in tokens
}

// ModelRegistry manages available translation models
type ModelRegistry struct {
	models map[string]*ModelInfo
}

// NewRegistry creates a new model registry with pre-configured translation models
func NewRegistry() *ModelRegistry {
	registry := &ModelRegistry{
		models: make(map[string]*ModelInfo),
	}

	registry.registerDefaultModels()
	return registry
}

// registerDefaultModels adds translation-optimized models to the registry
func (r *ModelRegistry) registerDefaultModels() {
	// PRIORITY 1: Translation-Specialized Models

	// Hunyuan-MT-7B: Best 7B translation model
	r.Register(&ModelInfo{
		ID:             "hunyuan-mt-7b-q4",
		Name:           "Hunyuan-MT 7B (Q4)",
		Description:    "Translation-optimized 7B model with commercial-grade quality for 33 languages",
		Parameters:     7_000_000_000,
		MinRAM:         6 * 1024 * 1024 * 1024,   // 6GB
		RecommendedRAM: 8 * 1024 * 1024 * 1024,   // 8GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/Tencent/Hunyuan-MT-7B-GGUF",
		Languages:      []string{"en", "ru", "sr", "zh", "es", "fr", "de", "ja", "ko"},
		OptimizedFor:   "Professional Translation",
		Quality:        "excellent",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  8192,
	})

	r.Register(&ModelInfo{
		ID:             "hunyuan-mt-7b-q8",
		Name:           "Hunyuan-MT 7B (Q8)",
		Description:    "High-quality translation with Q8 quantization for better accuracy",
		Parameters:     7_000_000_000,
		MinRAM:         9 * 1024 * 1024 * 1024,   // 9GB
		RecommendedRAM: 12 * 1024 * 1024 * 1024,  // 12GB
		QuantType:      "Q8_0",
		SourceURL:      "https://huggingface.co/Tencent/Hunyuan-MT-7B-GGUF",
		Languages:      []string{"en", "ru", "sr", "zh", "es", "fr", "de", "ja", "ko"},
		OptimizedFor:   "Professional Translation",
		Quality:        "excellent",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  8192,
	})

	// Aya-23: Multilingual translation model
	r.Register(&ModelInfo{
		ID:             "aya-23-8b-q4",
		Name:           "Aya 23 8B (Q4)",
		Description:    "Multilingual model supporting 23 languages with strong translation",
		Parameters:     8_000_000_000,
		MinRAM:         7 * 1024 * 1024 * 1024,   // 7GB
		RecommendedRAM: 10 * 1024 * 1024 * 1024,  // 10GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/CohereForAI/aya-23-8B-GGUF",
		Languages:      []string{"en", "ru", "sr", "ar", "zh", "cs", "de", "es", "fr", "hi"},
		OptimizedFor:   "Multilingual Translation",
		Quality:        "excellent",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  8192,
	})

	// PRIORITY 2: General-Purpose Models with Strong Translation

	// Qwen2.5: Excellent for multilingual tasks
	r.Register(&ModelInfo{
		ID:             "qwen2.5-7b-instruct-q4",
		Name:           "Qwen 2.5 7B Instruct (Q4)",
		Description:    "Multilingual model with strong Russian and Serbian support",
		Parameters:     7_000_000_000,
		MinRAM:         6 * 1024 * 1024 * 1024,   // 6GB
		RecommendedRAM: 8 * 1024 * 1024 * 1024,   // 8GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/Qwen/Qwen2.5-7B-Instruct-GGUF",
		Languages:      []string{"en", "ru", "sr", "zh", "ja", "ko", "de", "es", "fr"},
		OptimizedFor:   "General + Translation",
		Quality:        "excellent",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  32768,
	})

	// Mistral: Good general-purpose with translation capability
	r.Register(&ModelInfo{
		ID:             "mistral-7b-instruct-q4",
		Name:           "Mistral 7B Instruct v0.3 (Q4)",
		Description:    "High-quality general-purpose model with good translation",
		Parameters:     7_000_000_000,
		MinRAM:         6 * 1024 * 1024 * 1024,   // 6GB
		RecommendedRAM: 8 * 1024 * 1024 * 1024,   // 8GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/mistralai/Mistral-7B-Instruct-v0.3-GGUF",
		Languages:      []string{"en", "ru", "de", "es", "fr", "it"},
		OptimizedFor:   "General + Translation",
		Quality:        "good",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  8192,
	})

	// PRIORITY 3: Larger Models (for high-RAM systems)

	// Qwen2.5 14B: Better quality for systems with 16GB+ RAM
	r.Register(&ModelInfo{
		ID:             "qwen2.5-14b-instruct-q4",
		Name:           "Qwen 2.5 14B Instruct (Q4)",
		Description:    "Larger model for high-quality translation on capable systems",
		Parameters:     14_000_000_000,
		MinRAM:         12 * 1024 * 1024 * 1024,  // 12GB
		RecommendedRAM: 16 * 1024 * 1024 * 1024,  // 16GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/Qwen/Qwen2.5-14B-Instruct-GGUF",
		Languages:      []string{"en", "ru", "sr", "zh", "ja", "ko", "de", "es", "fr"},
		OptimizedFor:   "High-Quality Translation",
		Quality:        "excellent",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  32768,
	})

	// Qwen2.5 27B: Professional-grade for high-end systems
	r.Register(&ModelInfo{
		ID:             "qwen2.5-27b-instruct-q4",
		Name:           "Qwen 2.5 27B Instruct (Q4)",
		Description:    "Professional-grade translation for systems with 32GB+ RAM",
		Parameters:     27_000_000_000,
		MinRAM:         24 * 1024 * 1024 * 1024,  // 24GB
		RecommendedRAM: 32 * 1024 * 1024 * 1024,  // 32GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/Qwen/Qwen2.5-27B-Instruct-GGUF",
		Languages:      []string{"en", "ru", "sr", "zh", "ja", "ko", "de", "es", "fr"},
		OptimizedFor:   "Professional Translation",
		Quality:        "excellent",
		LicenseType:    "Apache-2.0",
		RequiresGPU:    false,
		ContextLength:  32768,
	})

	// PRIORITY 4: Compact Models (for low-resource systems)

	// Phi-3: Efficient small model
	r.Register(&ModelInfo{
		ID:             "phi-3-mini-4k-q4",
		Name:           "Phi-3 Mini 3.8B (Q4)",
		Description:    "Compact model for resource-constrained systems",
		Parameters:     3_800_000_000,
		MinRAM:         4 * 1024 * 1024 * 1024,   // 4GB
		RecommendedRAM: 6 * 1024 * 1024 * 1024,   // 6GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/microsoft/Phi-3-mini-4k-instruct-gguf",
		Languages:      []string{"en", "ru", "de", "es", "fr"},
		OptimizedFor:   "Low-Resource Translation",
		Quality:        "moderate",
		LicenseType:    "MIT",
		RequiresGPU:    false,
		ContextLength:  4096,
	})

	// Gemma 2: Google's efficient model
	r.Register(&ModelInfo{
		ID:             "gemma-2-9b-it-q4",
		Name:           "Gemma 2 9B Instruct (Q4)",
		Description:    "Google's efficient multilingual model",
		Parameters:     9_000_000_000,
		MinRAM:         8 * 1024 * 1024 * 1024,   // 8GB
		RecommendedRAM: 12 * 1024 * 1024 * 1024,  // 12GB
		QuantType:      "Q4_K_M",
		SourceURL:      "https://huggingface.co/google/gemma-2-9b-it-GGUF",
		Languages:      []string{"en", "ru", "de", "es", "fr", "it", "pt", "zh", "ja", "ko"},
		OptimizedFor:   "Balanced Translation",
		Quality:        "good",
		LicenseType:    "Gemma",
		RequiresGPU:    false,
		ContextLength:  8192,
	})
}

// Register adds a model to the registry
func (r *ModelRegistry) Register(model *ModelInfo) {
	r.models[model.ID] = model
}

// Get retrieves a model by ID
func (r *ModelRegistry) Get(id string) (*ModelInfo, bool) {
	model, exists := r.models[id]
	return model, exists
}

// List returns all registered models
func (r *ModelRegistry) List() []*ModelInfo {
	models := make([]*ModelInfo, 0, len(r.models))
	for _, model := range r.models {
		models = append(models, model)
	}
	return models
}

// FindBestModel finds the best model for given constraints
func (r *ModelRegistry) FindBestModel(maxRAM uint64, preferredLangs []string, hasGPU bool) (*ModelInfo, error) {
	var candidates []*ModelInfo

	// Filter models that fit in available RAM and match GPU availability
	// If hasGPU=true: include all models (GPU-optional and GPU-required)
	// If hasGPU=false: exclude models that require GPU
	for _, model := range r.models {
		if model.MinRAM <= maxRAM {
			if !model.RequiresGPU || hasGPU {
				candidates = append(candidates, model)
			}
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no models found within RAM constraint of %d GB", maxRAM/(1024*1024*1024))
	}

	// Score each model
	type scoredModel struct {
		model *ModelInfo
		score float64
	}

	var scored []scoredModel

	for _, model := range candidates {
		score := r.scoreModel(model, preferredLangs, maxRAM)
		scored = append(scored, scoredModel{model, score})
	}

	// Find highest scoring model
	best := scored[0]
	for _, sm := range scored[1:] {
		if sm.score > best.score {
			best = sm
		}
	}

	return best.model, nil
}

// scoreModel calculates a score for a model based on preferences
func (r *ModelRegistry) scoreModel(model *ModelInfo, preferredLangs []string, maxRAM uint64) float64 {
	score := 0.0

	// Quality bonus
	switch model.Quality {
	case "excellent":
		score += 10.0
	case "good":
		score += 7.0
	case "moderate":
		score += 4.0
	}

	// Translation optimization bonus
	if strings.Contains(strings.ToLower(model.OptimizedFor), "translation") {
		score += 5.0
	}

	// Language support bonus
	langSupport := 0
	for _, lang := range preferredLangs {
		for _, supported := range model.Languages {
			if lang == supported {
				langSupport++
				break
			}
		}
	}
	score += float64(langSupport) * 2.0

	// Size efficiency bonus (prefer models that use RAM efficiently)
	ramUsagePercent := float64(model.RecommendedRAM) / float64(maxRAM)
	if ramUsagePercent <= 0.6 { // Uses 60% or less of available RAM
		score += 3.0
	} else if ramUsagePercent <= 0.8 {
		score += 1.0
	}

	// Parameter size bonus (larger is generally better, up to a point)
	paramB := float64(model.Parameters) / 1_000_000_000
	if paramB >= 7 && paramB <= 14 {
		score += 2.0 // Sweet spot for translation
	} else if paramB > 14 {
		score += 1.0
	}

	// Context length bonus
	if model.ContextLength >= 8192 {
		score += 1.0
	}

	return score
}

// FilterByLanguages returns models that support all specified languages
func (r *ModelRegistry) FilterByLanguages(languages []string) []*ModelInfo {
	var filtered []*ModelInfo

	for _, model := range r.models {
		supportsAll := true
		for _, reqLang := range languages {
			found := false
			for _, modelLang := range model.Languages {
				if reqLang == modelLang {
					found = true
					break
				}
			}
			if !found {
				supportsAll = false
				break
			}
		}
		if supportsAll {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

// FilterByRAM returns models that fit within the specified RAM limit
func (r *ModelRegistry) FilterByRAM(maxRAM uint64) []*ModelInfo {
	var filtered []*ModelInfo

	for _, model := range r.models {
		if model.MinRAM <= maxRAM {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

// GetRecommendationsForHardware returns recommended models for specific hardware
func (r *ModelRegistry) GetRecommendationsForHardware(ramGB float64, hasGPU bool) []*ModelInfo {
	ramBytes := uint64(ramGB * 1024 * 1024 * 1024)

	var recommendations []*ModelInfo

	// Get all models that fit in RAM
	candidates := r.FilterByRAM(ramBytes)

	// Prioritize translation-optimized models
	for _, model := range candidates {
		if strings.Contains(strings.ToLower(model.OptimizedFor), "translation") {
			recommendations = append(recommendations, model)
		}
	}

	// Add general-purpose models if we don't have enough recommendations
	if len(recommendations) < 3 {
		for _, model := range candidates {
			isAlreadyAdded := false
			for _, rec := range recommendations {
				if rec.ID == model.ID {
					isAlreadyAdded = true
					break
				}
			}
			if !isAlreadyAdded {
				recommendations = append(recommendations, model)
			}
		}
	}

	return recommendations
}
