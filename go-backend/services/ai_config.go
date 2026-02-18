package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ModelConfig contains configuration for a single model
type ModelConfig struct {
	Provider          string  `yaml:"provider"`
	PromptPer1K       float64 `yaml:"prompt_per_1k"`
	CompletionPer1K   float64 `yaml:"completion_per_1k"`
	ContextWindow     int     `yaml:"context_window"`
	SupportsTools     bool    `yaml:"supports_tools"`
	SupportsStreaming bool    `yaml:"supports_streaming"`
	Description       string  `yaml:"description"`
}

// Defaults contains default configuration values
type Defaults struct {
	ChatModel  string  `yaml:"chat_model"`
	MaxTokens  int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// AIConfig contains the complete AI configuration
type AIConfig struct {
	Defaults Defaults               `yaml:"defaults"`
	Models   map[string]ModelConfig `yaml:"models"`
}

// ModelPricing contains pricing information for a model (per 1k tokens in USD)
// This is kept for backward compatibility
type ModelPricing struct {
	PromptPer1K     float64
	CompletionPer1K float64
}

var (
	// globalConfig holds the loaded AI configuration
	globalConfig *AIConfig

	// configMutex protects access to globalConfig
	configMutex sync.RWMutex

	// configOnce ensures config is loaded only once
	configOnce sync.Once

	// configLoaded indicates whether the config has been successfully loaded
	configLoaded bool

	// ValidChatModels is maintained for backward compatibility
	// This map is populated from the YAML config
	ValidChatModels map[string]ModelPricing
)

// LoadAIConfig loads the AI configuration from a YAML file
// The path should be relative to the project root or an absolute path
func LoadAIConfig(path string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// If path is empty, try default locations
	if path == "" {
		// Try relative to project root (go-backend/../config/)
		path = "../config/models.yaml"
	}

	// Read the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		// If file not found, use hardcoded defaults
		if os.IsNotExist(err) {
			return initDefaultConfig()
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML
	var config AIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate the config
	if err := validateConfig(&config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Store the config
	globalConfig = &config
	configLoaded = true

	// Populate ValidChatModels for backward compatibility
	populateValidChatModels(&config)

	return nil
}

// initDefaultConfig initializes the configuration with hardcoded defaults
// This is used when the config file is not found
// Note: This should only be called while holding configMutex to ensure thread safety
func initDefaultConfig() error {
	config := &AIConfig{
		Defaults: Defaults{
			ChatModel:    "google/gemini-2.5-flash",
			MaxTokens:    4096,
			Temperature: 0.7,
		},
		Models: map[string]ModelConfig{
			"google/gemini-2.5-flash": {
				Provider:          "google",
				PromptPer1K:       0.0003,
				CompletionPer1K:   0.0025,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Fast, efficient model for quick responses",
			},
			"google/gemini-2.5-pro": {
				Provider:          "google",
				PromptPer1K:       0.00125,
				CompletionPer1K:   0.010,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "High-quality model for complex reasoning",
			},
			"google/gemini-2.5-flash-lite": {
				Provider:          "google",
				PromptPer1K:       0.0001,
				CompletionPer1K:   0.0004,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Ultra-fast, lightweight model for simple tasks",
			},
			"openai/gpt-5-chat": {
				Provider:          "openai",
				PromptPer1K:       0.00125,
				CompletionPer1K:   0.010,
				ContextWindow:     128000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "OpenAI's flagship reasoning model",
			},
			"openai/gpt-5.1-chat": {
				Provider:          "openai",
				PromptPer1K:       0.00125,
				CompletionPer1K:   0.010,
				ContextWindow:     128000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "OpenAI GPT-5.1 with improved reasoning",
			},
			"openai/gpt-5.2-chat": {
				Provider:          "openai",
				PromptPer1K:       0.00175,
				CompletionPer1K:   0.014,
				ContextWindow:     128000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "OpenAI GPT-5.2 with enhanced capabilities",
			},
			"openai/gpt-4o-mini": {
				Provider:          "openai",
				PromptPer1K:       0.00015,
				CompletionPer1K:   0.0006,
				ContextWindow:     128000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Fast and cost-effective small model",
			},
			"anthropic/claude-sonnet-4.5": {
				Provider:          "anthropic",
				PromptPer1K:       0.003,
				CompletionPer1K:   0.015,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Anthropic's balanced performance model",
			},
			"anthropic/claude-sonnet-4.6": {
				Provider:          "anthropic",
				PromptPer1K:       0.003,
				CompletionPer1K:   0.015,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Anthropic's balanced performance model",
			},
			"anthropic/claude-opus-4.6": {
				Provider:          "anthropic",
				PromptPer1K:       0.003,
				CompletionPer1K:   0.025,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Anthropic's balanced performance model",
			},
			"google/gemini-3-pro-preview": {
				Provider:          "google",
				PromptPer1K:       0.002,
				CompletionPer1K:   0.012,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Gemini 3 Pro preview with advanced capabilities",
			},
			"google/gemini-3-flash-preview": {
				Provider:          "google",
				PromptPer1K:       0.0005,
				CompletionPer1K:   0.003,
				ContextWindow:     1000000,
				SupportsTools:     true,
				SupportsStreaming: true,
				Description:       "Gemini 3 Flash preview for fast responses",
			},
		},
	}

	globalConfig = config
	configLoaded = true
	populateValidChatModels(config)

	return nil
}

// validateConfig validates the AI configuration
func validateConfig(config *AIConfig) error {
	if config.Models == nil || len(config.Models) == 0 {
		return fmt.Errorf("no models defined in configuration")
	}

	for modelID, modelConfig := range config.Models {
		if modelConfig.Provider == "" {
			return fmt.Errorf("model %s: missing provider", modelID)
		}
		if modelConfig.PromptPer1K < 0 {
			return fmt.Errorf("model %s: invalid prompt pricing", modelID)
		}
		if modelConfig.CompletionPer1K < 0 {
			return fmt.Errorf("model %s: invalid completion pricing", modelID)
		}
		if modelConfig.ContextWindow <= 0 {
			return fmt.Errorf("model %s: invalid context window", modelID)
		}
	}

	return nil
}

// populateValidChatModels populates the ValidChatModels map from AIConfig
// This maintains backward compatibility with existing code
func populateValidChatModels(config *AIConfig) {
	ValidChatModels = make(map[string]ModelPricing)
	for modelID, modelConfig := range config.Models {
		ValidChatModels[modelID] = ModelPricing{
			PromptPer1K:     modelConfig.PromptPer1K,
			CompletionPer1K: modelConfig.CompletionPer1K,
		}
	}
}

// GetModelConfig retrieves the configuration for a specific model
// Returns an error if the model is not found
func GetModelConfig(model string) (ModelConfig, error) {
	// Ensure config is loaded first
	EnsureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	modelConfig, ok := globalConfig.Models[model]
	if !ok {
		return ModelConfig{}, fmt.Errorf("model not found: %s", model)
	}

	return modelConfig, nil
}

// ValidateChatModel checks if a model ID is valid
// Returns an error if the model is not in the configuration
func ValidateChatModel(model string) error {
	// Ensure config is loaded first
	EnsureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if _, ok := globalConfig.Models[model]; !ok {
		return fmt.Errorf("invalid model: %s", model)
	}

	return nil
}

// GetAllModels returns a map of all configured models
func GetAllModels() map[string]ModelConfig {
	// Ensure config is loaded first
	EnsureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	result := make(map[string]ModelConfig, len(globalConfig.Models))
	for k, v := range globalConfig.Models {
		result[k] = v
	}
	return result
}

// GetDefaultChatModel returns the default chat model ID
func GetDefaultChatModel() string {
	// Ensure config is loaded first
	EnsureConfigLoaded()

	configMutex.RLock()
	defer configMutex.RUnlock()

	if globalConfig.Defaults.ChatModel != "" {
		return globalConfig.Defaults.ChatModel
	}

	return "google/gemini-2.5-flash"
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	// Check for environment variable first
	if envPath := os.Getenv("ZETTELGARDEN_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Try to find the config file in relative locations
	possiblePaths := []string{
		"../config/models.yaml",
		"../../config/models.yaml",
	}

	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Return the default relative path if none found
	return "../config/models.yaml"
}

// EnsureConfigLoaded ensures the configuration is loaded
// This is called by functions that need the config to be initialized
// Uses sync.Once to ensure thread-safe initialization
func EnsureConfigLoaded() {
	configOnce.Do(func() {
		// Try to load from config file, fall back to defaults on error
		if err := LoadAIConfig(GetConfigPath()); err != nil {
			// LoadAIConfig already initializes defaults on file not found
			// Other errors are logged but we proceed with defaults
			log.Printf("Warning: Failed to load AI config: %v (using defaults)", err)
		}
	})
}
