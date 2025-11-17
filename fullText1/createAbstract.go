package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiConfig holds the API key and model name for Gemini.
type GeminiConfig struct {
	APIKey    string `json:"api_key"`
	ModelName string `json:"model_name"`
}

// loadGeminiConfig reads the Gemini configuration from the specified JSON file.
// It returns a *GeminiConfig and an error. If the file is not found or unreadable,
// it returns an error, allowing the caller to decide on fallback behavior.
func loadGeminiConfig(configPath string) (*GeminiConfig, error) {
	// The path is used directly; no home directory expansion is performed.
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

// generateAbstract interacts with the Gemini API to create a story abstract.
func generateAbstract(apiKey, modelName, instruction, language string, numChapters int) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("error creating Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	// The client.GenerativeModel method typically returns a non-nil model struct
	// even for invalid model names, with the error occurring on the API call.
	// This specific 'if model == nil' check might not be hit in practice with genai-go library.
	// Keeping it for consistency with original code, but actual model validation happens server-side.
	if model == nil {
		return "", fmt.Errorf("model '%s' could not be initialized (unexpected client state). Please check the model name.", modelName)
	}

	// Prompt engineering for a concise abstract
	// Dynamically include the number of chapters in the prompt
	prompt := fmt.Sprintf(`Write a concise, compelling story writing plan. 
It need to include the settings, the name of main characters and a detail plan for all %d chapters.
	`, numChapters)

	if instruction != "" {
		prompt += "\nStory Idea: " + instruction
	} else {
		prompt += " Create a detailed story idea."
	}

	// Add language instruction to the prompt
	prompt += fmt.Sprintf("\nOutput the plan in %s.", language)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("error generating content from Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated from Gemini for the given instruction")
	}

	var abstractBuilder strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			abstractBuilder.WriteString(string(txt))
		}
	}
	return abstractBuilder.String(), nil
}

func main() {
	// Default values for flags
	defaultModelFromEnvOrNoConfig := "gemini-2.5-flash" // This model will be used if NO config file is loaded, or if config is loaded but modelName is empty in it.

	// Define command-line flags
	// configPath now defaults to an empty string, meaning no config file is used by default.
	configPath := flag.String("config", "", "Path to Gemini configuration JSON file (optional). If not provided, API key is taken from GEMINI_API_KEY env var and model defaults to '"+defaultModelFromEnvOrNoConfig+"'.")
	outputPath := flag.String("output", "", "Path to save the generated abstract file (default: abstract-yyyy-mm-dd-hh-mm-ss.txt)")

	// Make --instruction optional with a default value
	defaultInstruction := ""
	instruction := flag.String("instruction", defaultInstruction, "Story instruction or idea for which to generate an abstract (optional). ")

	// Add language parameter
	language := flag.String("language", "english", "Specify the desired output language for the abstract (default: english).")

	// Add chapters parameter
	chapters := flag.Int("chapters", 0, "Specify the desired number of chapters for the story plan (optional). If not provided, a random number between 20-40 will be used.")

	flag.Parse()

	var apiKey string
	var modelName string

	// Check if a config path was explicitly provided.
	if *configPath != "" {
		// Attempt to load configuration from the specified file.
		geminiConfig, err := loadGeminiConfig(*configPath)
		if err != nil {
			log.Printf("Warning: Could not load Gemini configuration from '%s': %v. Falling back to environment variable GEMINI_API_KEY and default model '%s'.", *configPath, err, defaultModelFromEnvOrNoConfig)
			// Fall through to environment variable logic.
			apiKey = os.Getenv("GEMINI_API_KEY")
			if apiKey == "" {
				log.Fatalf("Error: GEMINI_API_KEY environment variable is not set, and the specified config file '%s' could not be loaded or was invalid. Please set GEMINI_API_KEY or provide a valid --config file.", *configPath)
			}
			modelName = defaultModelFromEnvOrNoConfig // Use the hardcoded default model
		} else {
			// Configuration file loaded successfully. Use values from it.
			apiKey = geminiConfig.APIKey
			modelName = geminiConfig.ModelName

			// If API key is missing in the config file, try environment variable as a secondary source.
			if apiKey == "" {
				log.Printf("Warning: API Key is missing in the config file '%s'. Attempting to use GEMINI_API_KEY environment variable.", *configPath)
				apiKey = os.Getenv("GEMINI_API_KEY")
				if apiKey == "" {
					log.Fatal("Error: API Key is missing in the config file and GEMINI_API_KEY environment variable is not set. Please provide an API key.")
				}
			}

			// If model name is missing in the config file, use the default.
			if modelName == "" {
				log.Printf("Warning: Model name not specified in config '%s'. Using default: %s", *configPath, defaultModelFromEnvOrNoConfig)
				modelName = defaultModelFromEnvOrNoConfig
			}
		}
	} else {
		// No config path was provided, so directly use environment variable.
		log.Printf("No --config file specified. Attempting to use GEMINI_API_KEY environment variable and default model '%s'.", defaultModelFromEnvOrNoConfig)
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			log.Fatal("Error: GEMINI_API_KEY environment variable is not set. Please set GEMINI_API_KEY or provide a valid --config file.")
		}
		modelName = defaultModelFromEnvOrNoConfig // Use the hardcoded default model
	}

	// Final check to ensure we have an API key and model name
	if apiKey == "" {
		log.Fatal("Error: No API key found after checking config file (if provided) and environment variable. Please provide it.")
	}
	if modelName == "" { // This case should theoretically be covered by the logic above, but good for robustness.
		modelName = defaultModelFromEnvOrNoConfig
		log.Printf("Warning: Model name was somehow still empty, defaulting to %s.", modelName)
	}

	// Determine number of chapters
	numChapters := *chapters
	if numChapters == 0 {
		// Seed the random number generator
		rand.Seed(time.Now().UnixNano())
		// Generate a random number between 20 and 40 (inclusive)
		numChapters = rand.Intn(21) + 20 // rand.Intn(n) generates [0, n-1], so 21 gives [0, 20]. Adding 20 shifts it to [20, 40].
		log.Printf("Number of chapters not specified. Generating a random number: %d", numChapters)
	} else {
		log.Printf("Using specified number of chapters: %d", numChapters)
	}

	// --- Generate Abstract ---
	log.Printf("Initiating abstract generation using Gemini model: %s, output language: %s, chapters: %d", modelName, *language, numChapters)
	abstract, err := generateAbstract(apiKey, modelName, *instruction, *language, numChapters)
	if err != nil {
		log.Fatalf("Error generating abstract: %v", err)
	}

	// --- Determine Output Path ---
	finalOutputPath := *outputPath
	if finalOutputPath == "" {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		finalOutputPath = fmt.Sprintf("abstract-%s.txt", timestamp)
	}

	// --- Save Abstract to File ---
	err = os.WriteFile(finalOutputPath, []byte(abstract), 0644)
	if err != nil {
		log.Fatalf("Error saving abstract to file '%s': %v", finalOutputPath, err)
	}

	fmt.Printf("Abstract successfully generated and saved to: %s\n", finalOutputPath)
	log.Printf("Abstract saved to: %s", finalOutputPath)
}