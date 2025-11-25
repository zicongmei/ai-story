package aiEndpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath" // Added
	"time"          // Added

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

// HistoryTurn represents a single turn in the conversation history used for preserving thought chains.
type HistoryTurn struct {
	UserPrompt       string
	ModelResponse    string
	ThoughtSignature []byte
}

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

// GeminiConfig holds the API key, model name, and thinking level for Gemini.
type GeminiConfig struct {
	APIKey        string `json:"api_key"`
	ModelName     string `json:"model_name"`
	ThinkingLevel string `json:"thinking_level"`
}

// GeminiConfigDetails holds configuration loaded or derived for Gemini API access.
type GeminiConfigDetails struct {
	APIKey        string
	ModelName     string
	ThinkingLevel string
	Err           error // To propagate errors gracefully from LoadGeminiConfigWithFallback
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
// and default model names. It returns GeminiConfigDetails.
func LoadGeminiConfigWithFallback(configPath string) GeminiConfigDetails { // Changed return signature
	var details GeminiConfigDetails

	if configPath != "" {
		geminiConfig, err := LoadGeminiConfig(configPath)
		if err != nil {
			log.Printf("Warning: Could not load Gemini configuration from '%s': %v. Falling back to environment variable GEMINI_API_KEY and default model '%s'.", configPath, err, DefaultGeminiModel)
			details.APIKey = os.Getenv("GEMINI_API_KEY")
			if details.APIKey == "" {
				details.Err = fmt.Errorf("GEMINI_API_KEY environment variable is not set, and the specified config file '%s' could not be loaded or was invalid. Please set GEMINI_API_KEY or provide a valid --config file", configPath)
				return details
			}
			details.ModelName = DefaultGeminiModel
		} else {
			details.APIKey = geminiConfig.APIKey
			details.ModelName = geminiConfig.ModelName
			details.ThinkingLevel = geminiConfig.ThinkingLevel

			// If API key is missing in the config file, try environment variable as a secondary source.
			if details.APIKey == "" {
				log.Printf("Warning: API Key is missing in the config file '%s'. Attempting to use GEMINI_API_KEY environment variable.", configPath)
				details.APIKey = os.Getenv("GEMINI_API_KEY")
				if details.APIKey == "" {
					details.Err = fmt.Errorf("API Key is missing in the config file and GEMINI_API_KEY environment variable is not set. Please provide an API key")
					return details
				}
			}

			// If model name is missing in the config file, use the default.
			if details.ModelName == "" {
				log.Printf("Warning: Model name not specified in config '%s'. Using default: %s", configPath, DefaultGeminiModel)
				details.ModelName = DefaultGeminiModel
			}
		}
	} else {
		// No config path was provided, so directly use environment variable.
		log.Printf("No --config file specified. Attempting to use GEMINI_API_KEY environment variable and default model '%s'.", DefaultGeminiModel)
		details.APIKey = os.Getenv("GEMINI_API_KEY")
		if details.APIKey == "" {
			details.Err = fmt.Errorf("GEMINI_API_KEY environment variable is not set. Please set GEMINI_API_KEY or provide a valid --config file")
			return details
		}
		details.ModelName = DefaultGeminiModel // Use the hardcoded default model
	}

	// Final check to ensure we have an API key and model name
	if details.APIKey == "" {
		details.Err = fmt.Errorf("no API key found after checking config file (if provided) and environment variable. Please provide it")
		return details
	}
	if details.ModelName == "" { // This case should theoretically be covered by the logic above, but good for robustness.
		details.ModelName = DefaultGeminiModel
		log.Printf("Warning: Model name was somehow still empty, defaulting to %s.", details.ModelName)
	}

	return details
}

// CallGeminiAPIInput holds all input parameters for the CallGeminiAPI function.
type CallGeminiAPIInput struct {
	Ctx              context.Context
	APIKey           string
	ModelName        string
	Prompt           string
	ThinkingLevel    string
	PreviousTurn     *HistoryTurn
	ThoughtSignature []byte
}

// GeminiAPIResponse holds all output parameters for the CallGeminiAPI function.
type GeminiAPIResponse struct {
	GeneratedText    string
	ThoughtSignature []byte
	InputTokens      int
	OutputTokens     int
	Cost             float64
	Err              error // To propagate errors gracefully from the API call
}

// ChapterCountResult holds the result of chapter count operations.
type ChapterCountResult struct {
	Count        int
	InputTokens  int
	OutputTokens int
	Cost         float64
	Err          error // To propagate errors gracefully
}

// CallGeminiAPI sends a prompt to the Gemini API and returns the generated text, thought signature,
// along with the input and output token counts, and the calculated cost.
// It supports an optional thinkingLevel and previous conversation history for thought chain continuity.
func CallGeminiAPI(input CallGeminiAPIInput) GeminiAPIResponse { // Updated signature
	log.Printf("Gemini API Call: Initiating call to model '%s'. Thinking Level: '%s'. Prompt length: %d characters.", input.ModelName, input.ThinkingLevel, len(input.Prompt))

	var response GeminiAPIResponse

	client, err := genai.NewClient(input.Ctx, &genai.ClientConfig{APIKey: input.APIKey})
	if err != nil {
		response.Err = fmt.Errorf("error creating Gemini client: %w", err)
		return response
	}

	// Construct request contents, potentially including history
	var reqContents []*genai.Content

	if input.PreviousTurn != nil {
		reqContents = append(reqContents, &genai.Content{
			Role: "user",
			Parts: []*genai.Part{{
				Text: input.PreviousTurn.UserPrompt,
			}},
		})
		reqContents = append(reqContents, &genai.Content{
			Role: "model",
			Parts: []*genai.Part{{
				Text:             input.PreviousTurn.ModelResponse,
				ThoughtSignature: input.PreviousTurn.ThoughtSignature,
			}},
		})
	}

	// Add current prompt
	reqContents = append(reqContents, &genai.Content{
		Role: "user",
		Parts: []*genai.Part{{
			Text:             input.Prompt,
			ThoughtSignature: input.ThoughtSignature,
		}},
	})

	var genConfig *genai.GenerateContentConfig

	if input.ModelName == "gemini-3-pro-preview" && input.ThinkingLevel != "" {
		// If thinking level is set for the supported model, use it and do NOT set thinking budget.
		genConfig = &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingLevel: genai.ThinkingLevel(input.ThinkingLevel),
			},
		}
	} else {
		// Default behavior: Use dynamic thinking budget (-1).
		thinking := int32(-1)
		genConfig = &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingBudget: &thinking,
			},
		}
	}

	// --- Log Request Body ---
	timestamp := time.Now().Format("20060102_150405.000000") // More precise timestamp
	reqFileName := filepath.Join(os.TempDir(), fmt.Sprintf("gemini_req_%s.json", timestamp))
	respFileName := filepath.Join(os.TempDir(), fmt.Sprintf("gemini_resp_%s.json", timestamp))

	reqBodyBytes, errMarshalReq := json.MarshalIndent(reqContents, "", "  ")
	if errMarshalReq != nil {
		log.Printf("Warning: Failed to marshal Gemini request contents for logging: %v", errMarshalReq)
	} else {
		if errWriteReq := os.WriteFile(reqFileName, reqBodyBytes, 0644); errWriteReq != nil {
			log.Printf("Warning: Failed to write Gemini request body to '%s': %v", reqFileName, errWriteReq)
		} else {
			log.Printf("Gemini API Call: Request body saved to: %s", reqFileName)
		}
	}
	// --- End Log Request Body ---

	// First, count input tokens to determine pricing tier
	countResp, err := client.Models.CountTokens(input.Ctx, input.ModelName, reqContents, &genai.CountTokensConfig{})
	if err != nil {
		log.Printf("Warning: Failed to count input tokens for prompt: %v. Proceeding with generation and assuming 0 input tokens for cost calculation.", err)
		// Don't return error here, proceed with generation but log 0 for input tokens
	}
	response.InputTokens = 0
	if countResp != nil {
		response.InputTokens = int(countResp.TotalTokens)
	}

	// Get model prices based on model name and input tokens
	modelPrices, err := GetModelPrices(input.ModelName, response.InputTokens)
	if err != nil {
		log.Printf("Warning: Could not get pricing for model '%s': %v. Cost will be reported as 0.", input.ModelName, err)
		modelPrices = &ModelPrices{} // Default to zero prices if not found
	}

	// Generate content
	resp, err := client.Models.GenerateContent(input.Ctx, input.ModelName, reqContents, genConfig)

	// --- Log Response Body ---
	if resp != nil {
		respBodyBytes, errMarshalResp := json.MarshalIndent(resp, "", "  ")
		if errMarshalResp != nil {
			log.Printf("Warning: Failed to marshal Gemini response for logging: %v", errMarshalResp)
		} else {
			if errWriteResp := os.WriteFile(respFileName, respBodyBytes, 0644); errWriteResp != nil {
				log.Printf("Warning: Failed to write Gemini response body to '%s': %v", respFileName, errWriteResp)
			} else {
				log.Printf("Gemini API Call: Response body saved to: %s", respFileName)
			}
		}
	} else {
		log.Printf("Gemini API Call: No response object to log.")
	}
	// --- End Log Response Body ---

	if err != nil {
		log.Printf("Gemini API Call: Error generating content: %v", err)
		response.Err = fmt.Errorf("error generating content from Gemini: %w", err)
		return response
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("Gemini API Call: No content generated for the given instruction.")
		response.Err = fmt.Errorf("no content generated from Gemini for the given instruction")
		return response
	}

	response.GeneratedText = resp.Text()
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		response.ThoughtSignature = resp.Candidates[0].Content.Parts[0].ThoughtSignature
	}

	response.OutputTokens = 0
	if resp.UsageMetadata != nil {
		response.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	} else {
		log.Println("failed to count output token, output tokens will be 0 for cost calculation.")
	}

	// Calculate cost
	response.Cost = (float64(response.InputTokens)/TokensPerMillion)*modelPrices.InputPricePerMillion +
		(float64(response.OutputTokens)/TokensPerMillion)*modelPrices.OutputPricePerMillion

	return response
}
