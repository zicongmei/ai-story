package abstract

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	// "gopkg.in/yaml.v3" // Moved to pkg/abstract/file

	"github.com/zicongmei/ai-story/fullText1/pkg/abstract/file" // New import
	"github.com/zicongmei/ai-story/fullText1/pkg/aiEndpoint"
)

// AbstractOutput structure for YAML output - MOVED to pkg/abstract/file
// type AbstractOutput struct {
// 	Abstract         string `json:"abstract"`
// 	ThoughtSignature []byte `json:"thought_signature"`
// }

// GenerateAbstractInput holds all input parameters for the generateAbstract function.
type GenerateAbstractInput struct {
	APIKey        string
	ModelName     string
	ThinkingLevel string
	Instruction   string
	Language      string
	NumChapters   int
}

// AbstractGenerationResult holds all output parameters for the generateAbstract function.
type AbstractGenerationResult struct {
	Abstract         string
	ThoughtSignature []byte
	InputTokens      int
	OutputTokens     int
	Cost             float64
	Err              error // To propagate errors gracefully
}

// generateAbstract interacts with the Gemini API to create a story abstract.
func generateAbstract(input GenerateAbstractInput) AbstractGenerationResult { // Updated signature
	var result AbstractGenerationResult

	// Prompt engineering for a concise abstract
	// Dynamically include the number of chapters in the prompt
	prompt := fmt.Sprintf(`Write a concise, compelling story writing plan.
It need to include the settings, the name of main characters and a detail plan for all %d chapters.
	`, input.NumChapters)

	if input.Instruction != "" {
		prompt += "\nStory Idea: " + input.Instruction
	} else {
		prompt += " Create a detailed story idea."
	}

	// Add language instruction to the prompt
	prompt += fmt.Sprintf("\nOutput the plan in %s.", input.Language)

	apiInput := aiEndpoint.CallGeminiAPIInput{
		Ctx:           context.Background(),
		APIKey:        input.APIKey,
		ModelName:     input.ModelName,
		Prompt:        prompt,
		ThinkingLevel: input.ThinkingLevel,
		PreviousTurn:  nil,
	}
	apiResponse := aiEndpoint.CallGeminiAPI(apiInput)

	if apiResponse.Err != nil {
		result.Err = fmt.Errorf("error generating content from Gemini: %w", apiResponse.Err)
		return result
	}

	result.Abstract = apiResponse.GeneratedText
	result.ThoughtSignature = apiResponse.ThoughtSignature
	result.InputTokens = apiResponse.InputTokens
	result.OutputTokens = apiResponse.OutputTokens
	result.Cost = apiResponse.Cost
	return result
}

// GetChapterCountInput holds all input parameters for the getChapterCountFromGemini function.
// This is specific to the abstract subcommand's chapter count check.
type GetChapterCountInput struct {
	APIKey        string
	ModelName     string
	ThinkingLevel string
	Abstract      string
}

// getChapterCountFromGemini sends the abstract to Gemini to get a pure chapter count.
func getChapterCountFromGemini(input GetChapterCountInput) aiEndpoint.ChapterCountResult { // Updated signature, uses aiEndpoint.ChapterCountResult
	var result aiEndpoint.ChapterCountResult

	prompt := fmt.Sprintf(`Given the following complete story abstract (plan), please return ONLY the total number of chapters planned within it.
Do not include any other text, explanation, or formatting. Just the pure number.

--- Story Abstract ---
%s
--- End Story Abstract ---
`, input.Abstract)

	apiInput := aiEndpoint.CallGeminiAPIInput{
		Ctx:           context.Background(),
		APIKey:        input.APIKey,
		ModelName:     input.ModelName,
		Prompt:        prompt,
		ThinkingLevel: input.ThinkingLevel,
		PreviousTurn:  nil,
	}
	apiResponse := aiEndpoint.CallGeminiAPI(apiInput)

	if apiResponse.Err != nil {
		result.Err = fmt.Errorf("error calling Gemini to get chapter count: %w", apiResponse.Err)
		return result
	}

	// Clean up the response to ensure it's a pure number
	countStr := strings.TrimSpace(apiResponse.GeneratedText)
	// Take only the first line in case Gemini adds extra text after the number
	countStr = strings.Split(countStr, "\n")[0]

	count, err := strconv.Atoi(countStr)
	if err != nil {
		result.InputTokens = apiResponse.InputTokens
		result.OutputTokens = apiResponse.OutputTokens
		result.Cost = apiResponse.Cost
		result.Err = fmt.Errorf("could not parse chapter count '%s' from Gemini response: %w", countStr, err)
		return result
	}

	result.Count = count
	result.InputTokens = apiResponse.InputTokens
	result.OutputTokens = apiResponse.OutputTokens
	result.Cost = apiResponse.Cost
	return result
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
	outputPath := cmd.String("output", "", "Path to save the generated abstract file (default: abstract-yyyy-mm-dd-hh-mm-ss.yaml)") // Changed default extension

	defaultInstruction := ""
	instruction := cmd.String("instruction", defaultInstruction, "Story instruction or idea for which to generate an abstract (optional). ")

	language := cmd.String("language", "english", "Specify the desired output language for the abstract (default: english).")

	chapters := cmd.Int("chapters", 0, "Specify the desired number of chapters for the story plan (optional). If not provided, a random number between 20-40 will be used.")

	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("failed to parse abstract subcommand flags: %w", err)
	}

	// Load Gemini config using the utility function
	geminiConfigDetails := aiEndpoint.LoadGeminiConfigWithFallback(*configPath) // Updated call
	if geminiConfigDetails.Err != nil {
		return geminiConfigDetails.Err // aiEndpoint.LoadGeminiConfigWithFallback already logs detailed errors.
	}
	apiKey := geminiConfigDetails.APIKey
	modelName := geminiConfigDetails.ModelName
	thinkingLevel := geminiConfigDetails.ThinkingLevel

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
	generateAbstractInput := GenerateAbstractInput{
		APIKey:        apiKey,
		ModelName:     modelName,
		ThinkingLevel: thinkingLevel,
		Instruction:   *instruction,
		Language:      *language,
		NumChapters:   numChapters,
	}
	abstractResult := generateAbstract(generateAbstractInput) // Updated call
	if abstractResult.Err != nil {
		return fmt.Errorf("error generating abstract: %w", abstractResult.Err)
	}
	abstract := abstractResult.Abstract
	signature := abstractResult.ThoughtSignature
	accumulatedInputTokens += abstractResult.InputTokens
	accumulatedOutputTokens += abstractResult.OutputTokens
	accumulatedCost += abstractResult.Cost
	log.Printf("Abstract generation complete. Input tokens: %d, Output tokens: %d, Cost: $%.6f", abstractResult.InputTokens, abstractResult.OutputTokens, abstractResult.Cost)

	// --- Determine Output Path ---
	finalOutputPath := *outputPath
	if finalOutputPath == "" {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		finalOutputPath = fmt.Sprintf("abstract-%s.yaml", timestamp) // Changed to .yaml
	}

	// --- Save Abstract and Thought Signature to YAML File ---
	err := file.WriteAbstractFile(finalOutputPath, abstract, signature)
	if err != nil {
		return fmt.Errorf("error saving abstract: %w", err)
	}

	fmt.Printf("Abstract successfully generated and saved to: %s\n", finalOutputPath)
	log.Printf("Abstract saved to: %s", finalOutputPath)

	// --- New Step: Get pure chapter count from Gemini ---
	log.Printf("Sending abstract to Gemini to get pure chapter count...")
	getChapterCountInput := GetChapterCountInput{
		APIKey:        apiKey,
		ModelName:     modelName,
		ThinkingLevel: thinkingLevel,
		Abstract:      abstract,
	}
	chapterCountResult := getChapterCountFromGemini(getChapterCountInput) // Updated call
	if chapterCountResult.Err != nil {
		log.Printf("Warning: Failed to get pure chapter count from Gemini: %v. Proceeding without this information.", chapterCountResult.Err)
	} else {
		accumulatedInputTokens += chapterCountResult.InputTokens
		accumulatedOutputTokens += chapterCountResult.OutputTokens
		accumulatedCost += chapterCountResult.Cost
		fmt.Printf("Pure chapter count from Gemini: %d\n", chapterCountResult.Count)
		log.Printf("Pure chapter count from Gemini: %d. Input tokens: %d, Output tokens: %d, Cost: $%.6f", chapterCountResult.Count, chapterCountResult.InputTokens, chapterCountResult.OutputTokens, chapterCountResult.Cost)
	}

	fmt.Printf("Total accumulated cost for abstract generation process: $%.6f\n", accumulatedCost)
	log.Printf("Total accumulated tokens for abstract generation process: Input %d, Output %d. Total accumulated cost: $%.6f",
		accumulatedInputTokens, accumulatedOutputTokens, accumulatedCost)

	return nil
}
