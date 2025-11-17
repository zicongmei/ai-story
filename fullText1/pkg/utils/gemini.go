package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const DefaultGeminiModel = "gemini-2.5-flash"

// GeminiConfig holds the API key and model name for Gemini.
type GeminiConfig struct {
	APIKey    string `json:"api_key"`
	ModelName string `json:"model_name"`
}

// LoadGeminiConfig reads the Gemini configuration from the specified JSON file.
// It returns a *GeminiConfig and an error. If the file is not found or unreadable,
// it returns an error, allowing the caller to decide on fallback behavior.
func LoadGeminiConfig(configPath string) (*GeminiConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	var config GeminiConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", configPath, err)
	}

	return &config, nil
}

// LoadGeminiConfigWithFallback attempts to load configuration from a file.
// If the file is not provided or fails to load, it falls back to environment variables
// and default model names. It returns the API key and model name, or an error.
func LoadGeminiConfigWithFallback(configPath string) (string, string, error) {
	var apiKey string
	var modelName string

	if configPath != "" {
		geminiConfig, err := LoadGeminiConfig(configPath)
		if err != nil {
			log.Printf("Warning: Could not load Gemini configuration from '%s': %v. Falling back to environment variable GEMINI_API_KEY and default model '%s'.", configPath, err, DefaultGeminiModel)
			// Fall through to environment variable logic.
			apiKey = os.Getenv("GEMINI_API_KEY")
			if apiKey == "" {
				return "", "", fmt.Errorf("GEMINI_API_KEY environment variable is not set, and the specified config file '%s' could not be loaded or was invalid. Please set GEMINI_API_KEY or provide a valid --config file", configPath)
			}
			modelName = DefaultGeminiModel // Use the hardcoded default model
		} else {
			// Configuration file loaded successfully. Use values from it.
			apiKey = geminiConfig.APIKey
			modelName = geminiConfig.ModelName

			// If API key is missing in the config file, try environment variable as a secondary source.
			if apiKey == "" {
				log.Printf("Warning: API Key is missing in the config file '%s'. Attempting to use GEMINI_API_KEY environment variable.", configPath)
				apiKey = os.Getenv("GEMINI_API_KEY")
				if apiKey == "" {
					return "", "", fmt.Errorf("API Key is missing in the config file and GEMINI_API_KEY environment variable is not set. Please provide an API key")
				}
			}

			// If model name is missing in the config file, use the default.
			if modelName == "" {
				log.Printf("Warning: Model name not specified in config '%s'. Using default: %s", configPath, DefaultGeminiModel)
				modelName = DefaultGeminiModel
			}
		}
	} else {
		// No config path was provided, so directly use environment variable.
		log.Printf("No --config file specified. Attempting to use GEMINI_API_KEY environment variable and default model '%s'.", DefaultGeminiModel)
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return "", "", fmt.Errorf("GEMINI_API_KEY environment variable is not set. Please set GEMINI_API_KEY or provide a valid --config file")
		}
		modelName = DefaultGeminiModel // Use the hardcoded default model
	}

	// Final check to ensure we have an API key and model name
	if apiKey == "" {
		return "", "", fmt.Errorf("no API key found after checking config file (if provided) and environment variable. Please provide it")
	}
	if modelName == "" { // This case should theoretically be covered by the logic above, but good for robustness.
		modelName = DefaultGeminiModel
		log.Printf("Warning: Model name was somehow still empty, defaulting to %s.", modelName)
	}

	return apiKey, modelName, nil
}

// CallGeminiAPI sends a prompt to the Gemini API and returns the generated text.
func CallGeminiAPI(ctx context.Context, apiKey, modelName, prompt string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("error creating Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	if model == nil {
		return "", fmt.Errorf("model '%s' could not be initialized (unexpected client state). Please check the model name.", modelName)
	}

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("error generating content from Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated from Gemini for the given instruction")
	}

	var responseBuilder strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseBuilder.WriteString(string(txt))
		}
	}
	return responseBuilder.String(), nil
}
