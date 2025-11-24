package abstract

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zicongmei/ai-story/fullText1/pkg/utils"
)

// AbstractOutput structure for JSON output
type AbstractOutput struct {
	Abstract         string `json:"abstract"`
	ThoughtSignature []byte `json:"thought_signature"`
}

// generateAbstract interacts with the Gemini API to create a story abstract.
func generateAbstract(apiKey, modelName, thinkingLevel, instruction, language string, numChapters int) (string, []byte, int, int, float64, error) { // Updated signature
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

	// Note: We pass nil for history here as this is the initial request
	abstract, signature, inputTokens, outputTokens, cost, err := utils.CallGeminiAPI(context.Background(), apiKey, modelName, prompt, thinkingLevel, nil) // Updated call
	if err != nil {
		return "", nil, 0, 0, 0, fmt.Errorf("error generating content from Gemini: %w", err)
	}

	return abstract, signature, inputTokens, outputTokens, cost, nil
}

// getChapterCountFromGemini sends the abstract to Gemini to get a pure chapter count.
func getChapterCountFromGemini(apiKey, modelName, thinkingLevel, abstract string) (int, int, int, float64, error) { // Updated signature
	prompt := fmt.Sprintf(`Given the following complete story abstract (plan), please return ONLY the total number of chapters planned within it.
Do not include any other text, explanation, or formatting. Just the pure number.

--- Story Abstract ---
%s
--- End Story Abstract ---
`, abstract)

	// Note: We pass nil for history here as this is a standalone request
	countStr, _, inputTokens, outputTokens, cost, err := utils.CallGeminiAPI(context.Background(), apiKey, modelName, prompt, thinkingLevel, nil) // Updated call
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("error calling Gemini to get chapter count: %w", err)
	}

	// Clean up the response to ensure it's a pure number
	countStr = strings.TrimSpace(countStr)
	// Take only the first line in case Gemini adds extra text after the number
	countStr = strings.Split(countStr, "\n")[0]

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, inputTokens, outputTokens, cost, fmt.Errorf("could not parse chapter count '%s' from Gemini response: %w", countStr, err)
	}

	return count, inputTokens, outputTokens, cost, nil
}

// Execute is the main entry point for the 'abstract' subcommand.
func Execute(args []string) error {
	cmd := flag.NewFlagSet("abstract", flag.ContinueOnError) // Use ContinueOnError to allow main to handle errors
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "Usage of %s abstract:\n", os.Args[0])
		cmd.PrintDefaults()
	}

	// Define command-line flags
	configPath := cmd.String("config", "", "Path to Gemini configuration JSON file (optional). If not provided, API key is taken from GEMINI_API_KEY env var and model defaults to 'gemini-pro'.")
	outputPath := cmd.String("output", "", "Path to save the generated abstract file (default: abstract-yyyy-mm-dd-hh-mm-ss.json)")

	defaultInstruction := ""
	instruction := cmd.String("instruction", defaultInstruction, "Story instruction or idea for which to generate an abstract (optional). ")

	language := cmd.String("language", "english", "Specify the desired output language for the abstract (default: english).")

	chapters := cmd.Int("chapters", 0, "Specify the desired number of chapters for the story plan (optional). If not provided, a random number between 20-40 will be used.")

	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("failed to parse abstract subcommand flags: %w", err)
	}

	// Load Gemini config using the utility function
	apiKey, modelName, thinkingLevel, err := utils.LoadGeminiConfigWithFallback(*configPath) // Updated to receive thinkingLevel
	if err != nil {
		return err // utils.LoadGeminiConfigWithFallback already logs detailed errors.
	}

	// Determine number of chapters for the *initial* abstract generation
	numChapters := *chapters
	if numChapters == 0 {
		// Seed the random number generator
		rand.Seed(time.Now().UnixNano())
		// Generate a random number between 20 and 40 (inclusive)
		numChapters = rand.Intn(21) + 20 // rand.Intn(n) generates [0, 20]. Adding 20 shifts it to [20, 40].
		log.Printf("Number of chapters not specified for abstract generation. Generating a random number: %d", numChapters)
	} else {
		log.Printf("Using specified number of chapters: %d for abstract generation", numChapters)
	}

	var accumulatedInputTokens int
	var accumulatedOutputTokens int
	var accumulatedCost float64

	// --- Generate Abstract ---
	log.Printf("Initiating abstract generation using Gemini model: %s, output language: %s, chapters: %d", modelName, *language, numChapters)
	// Updated call to include thinkingLevel
	abstract, signature, inputTokensAbstract, outputTokensAbstract, costAbstract, err := generateAbstract(apiKey, modelName, thinkingLevel, *instruction, *language, numChapters)
	if err != nil {
		return fmt.Errorf("error generating abstract: %w", err)
	}
	accumulatedInputTokens += inputTokensAbstract
	accumulatedOutputTokens += outputTokensAbstract
	accumulatedCost += costAbstract
	log.Printf("Abstract generation complete. Input tokens: %d, Output tokens: %d, Cost: $%.6f", inputTokensAbstract, outputTokensAbstract, costAbstract)


	// --- Determine Output Path ---
	finalOutputPath := *outputPath
	if finalOutputPath == "" {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		finalOutputPath = fmt.Sprintf("abstract-%s.json", timestamp)
	}

	// --- Save Abstract and Thought Signature to JSON File ---
	outputData := AbstractOutput{
		Abstract:         abstract,
		ThoughtSignature: signature,
	}
	jsonBytes, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling abstract output to JSON: %w", err)
	}

	err = os.WriteFile(finalOutputPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("error saving abstract to file '%s': %w", finalOutputPath, err)
	}

	fmt.Printf("Abstract successfully generated and saved to: %s\n", finalOutputPath)
	log.Printf("Abstract saved to: %s", finalOutputPath)

	// --- New Step: Get pure chapter count from Gemini ---
	log.Printf("Sending abstract to Gemini to get pure chapter count...")
	// Updated call to include thinkingLevel
	pureChapterCount, inputTokensCount, outputTokensCount, costCount, err := getChapterCountFromGemini(apiKey, modelName, thinkingLevel, abstract)
	if err != nil {
		log.Printf("Warning: Failed to get pure chapter count from Gemini: %v. Proceeding without this information.", err)
	} else {
		accumulatedInputTokens += inputTokensCount
		accumulatedOutputTokens += outputTokensCount
		accumulatedCost += costCount
		fmt.Printf("Pure chapter count from Gemini: %d\n", pureChapterCount)
		log.Printf("Pure chapter count from Gemini: %d. Input tokens: %d, Output tokens: %d, Cost: $%.6f", pureChapterCount, inputTokensCount, outputTokensCount, costCount)
	}

	fmt.Printf("Total accumulated cost for abstract generation process: $%.6f\n", accumulatedCost)
	log.Printf("Total accumulated tokens for abstract generation process: Input %d, Output %d. Total accumulated cost: $%.6f",
		accumulatedInputTokens, accumulatedOutputTokens, accumulatedCost)

	return nil
}