package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
)

const DefaultGeminiModel = "gemini-2.5-flash"

// Pricing constants per 1 million tokens
const (
	// Gemini 2.5 Pro (prompts <= 200k tokens)
	Gemini25ProInputPriceLowTierPerMillion  float64 = 1.25
	Gemini25ProOutputPriceLowTierPerMillion float64 = 10.00

	// Gemini 2.5 Pro (prompts > 200k tokens)
	Gemini25ProInputPriceHighTierPerMillion  float64 = 2.50
	Gemini25ProOutputPriceHighTierPerMillion float64 = 15.00

	// Gemini 3 Pro Preview (prompts <= 200k tokens)
	Gemini3ProPreviewInputPriceLowTierPerMillion  float64 = 2.00
	Gemini3ProPreviewOutputPriceLowTierPerMillion float64 = 12.00

	// Gemini 3 Pro Preview (prompts > 200k tokens)
	Gemini3ProPreviewInputPriceHighTierPerMillion  float64 = 4.00
	Gemini3ProPreviewOutputPriceHighTierPerMillion float64 = 18.00

	Gemini25ProPromptTokenThreshold = 200000

	// Gemini 2.5 Flash
	Gemini25FlashInputPricePerMillion  float64 = 0.30
	Gemini25FlashOutputPricePerMillion float64 = 2.50

	// Gemini 2.5 Flash Lite
	Gemini25FlashLiteInputPricePerMillion  float64 = 0.10
	Gemini25FlashLiteOutputPricePerMillion float64 = 0.40

	TokensPerMillion float64 = 1_000_000.0
)

// ModelPrices holds the per-million token pricing for a specific model tier.
type ModelPrices struct {
	InputPricePerMillion  float64
	OutputPricePerMillion float64
}

// GetModelPrices returns the input and output prices per 1 million tokens for a given model and input token count.
// The inputTokens parameter is crucial for determining the pricing tier for models like gemini-2.5-pro.
func GetModelPrices(modelName string, inputTokens int) (*ModelPrices, error) {
	switch modelName {
	case "gemini-2.5-pro", "gemini-1.5-pro", "gemini-pro": // Treat "gemini-1.5-pro" and "gemini-pro" as "gemini-2.5-pro" for pricing based on available rates.
		if inputTokens <= Gemini25ProPromptTokenThreshold {
			return &ModelPrices{
				InputPricePerMillion:  Gemini25ProInputPriceLowTierPerMillion,
				OutputPricePerMillion: Gemini25ProOutputPriceLowTierPerMillion,
			}, nil
		} else {
			return &ModelPrices{
				InputPricePerMillion:  Gemini25ProInputPriceHighTierPerMillion,
				OutputPricePerMillion: Gemini25ProOutputPriceHighTierPerMillion,
			}, nil
		}
	case "gemini-3-pro-preview":
		if inputTokens <= Gemini25ProPromptTokenThreshold {
			return &ModelPrices{
				InputPricePerMillion:  Gemini3ProPreviewInputPriceLowTierPerMillion,
				OutputPricePerMillion: Gemini3ProPreviewOutputPriceLowTierPerMillion,
			}, nil
		} else {
			return &ModelPrices{
				InputPricePerMillion:  Gemini3ProPreviewInputPriceHighTierPerMillion,
				OutputPricePerMillion: Gemini3ProPreviewOutputPriceHighTierPerMillion,
			}, nil
		}
	case "gemini-2.5-flash":
		return &ModelPrices{
			InputPricePerMillion:  Gemini25FlashInputPricePerMillion,
			OutputPricePerMillion: Gemini25FlashOutputPricePerMillion,
		}, nil
	case "gemini-2.5-flash-lite":
		return &ModelPrices{
			InputPricePerMillion:  Gemini25FlashLiteInputPricePerMillion,
			OutputPricePerMillion: Gemini25FlashLiteOutputPricePerMillion,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported model for pricing: %s", modelName)
	}
}

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

// CallGeminiAPI sends a prompt to the Gemini API and returns the generated text,
// along with the input and output token counts, and the calculated cost.
func CallGeminiAPI(ctx context.Context, apiKey, modelName, prompt string) (string, int, int, float64, error) { // Added float64 for cost
	log.Printf("Gemini API Call: Initiating call to model '%s'. Prompt length: %d characters.", modelName, len(prompt))

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("error creating Gemini client: %w", err)
	}

	apiPrompt := genai.Text(prompt)
	thinking := int32(-1)
	genCofig := &genai.GenerateContentConfig{
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &thinking,
		},
	}

	// First, count input tokens to determine pricing tier
	countResp, err := client.Models.CountTokens(ctx, modelName, apiPrompt, &genai.CountTokensConfig{})
	if err != nil {
		log.Printf("Warning: Failed to count input tokens for prompt (length %d): %v. Proceeding with generation and assuming 0 input tokens for cost calculation.", len(prompt), err)
		// Don't return error here, proceed with generation but log 0 for input tokens
	}
	inputTokens := 0
	if countResp != nil {
		inputTokens = int(countResp.TotalTokens)
	}
	log.Printf("Gemini API Call: Input token count: %d", inputTokens)

	// Get model prices based on model name and input tokens
	modelPrices, err := GetModelPrices(modelName, inputTokens)
	if err != nil {
		log.Printf("Warning: Could not get pricing for model '%s': %v. Cost will be reported as 0.", modelName, err)
		modelPrices = &ModelPrices{} // Default to zero prices if not found
	}

	resp, err := client.Models.GenerateContent(ctx, modelName, apiPrompt, genCofig)

	if err != nil {
		log.Printf("Gemini API Call: Error generating content: %v", err)
		return "", inputTokens, 0, 0, fmt.Errorf("error generating content from Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("Gemini API Call: No content generated for the given instruction.")
		return "", inputTokens, 0, 0, fmt.Errorf("no content generated from Gemini for the given instruction")
	}

	generatedText := resp.Text()

	outputTokens := 0
	if resp.UsageMetadata != nil {
		outputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	} else {
		log.Println("failed to count output token, output tokens will be 0 for cost calculation.")
	}

	// Calculate cost
	cost := (float64(inputTokens)/TokensPerMillion)*modelPrices.InputPricePerMillion +
		(float64(outputTokens)/TokensPerMillion)*modelPrices.OutputPricePerMillion

	log.Printf("Gemini API Call: Call to model '%s' completed. Input tokens: %d, Output tokens: %d, Cost: $%.6f", modelName, inputTokens, outputTokens, cost)

	return generatedText, inputTokens, outputTokens, cost, nil
}