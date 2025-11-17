package abstract

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv" // Added for parsing the chapter number from Gemini
	"strings" // Added for trimming whitespace
	"time"

	"github.com/zicongmei/ai-story/fullText1/pkg/utils"
)

// generateAbstract interacts with the Gemini API to create a story abstract.
func generateAbstract(apiKey, modelName, instruction, language string, numChapters int) (string, error) {
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

	abstract, err := utils.CallGeminiAPI(context.Background(), apiKey, modelName, prompt)
	if err != nil {
		return "", fmt.Errorf("error generating content from Gemini: %w", err)
	}

	return abstract, nil
}

// getChapterCountFromGemini sends the abstract to Gemini to get a pure chapter count.
func getChapterCountFromGemini(apiKey, modelName, abstract string) (int, error) {
	prompt := fmt.Sprintf(`Given the following complete story abstract (plan), please return ONLY the total number of chapters planned within it.
Do not include any other text, explanation, or formatting. Just the pure number.

--- Story Abstract ---
%s
--- End Story Abstract ---
`, abstract)

	countStr, err := utils.CallGeminiAPI(context.Background(), apiKey, modelName, prompt)
	if err != nil {
		return 0, fmt.Errorf("error calling Gemini to get chapter count: %w", err)
	}

	// Clean up the response to ensure it's a pure number
	countStr = strings.TrimSpace(countStr)
	// Take only the first line in case Gemini adds extra text after the number
	countStr = strings.Split(countStr, "\n")[0]

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, fmt.Errorf("could not parse chapter count '%s' from Gemini response: %w", countStr, err)
	}

	return count, nil
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
	outputPath := cmd.String("output", "", "Path to save the generated abstract file (default: abstract-yyyy-mm-dd-hh-mm-ss.txt)")

	defaultInstruction := ""
	instruction := cmd.String("instruction", defaultInstruction, "Story instruction or idea for which to generate an abstract (optional). ")

	language := cmd.String("language", "english", "Specify the desired output language for the abstract (default: english).")

	chapters := cmd.Int("chapters", 0, "Specify the desired number of chapters for the story plan (optional). If not provided, a random number between 20-40 will be used.")

	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("failed to parse abstract subcommand flags: %w", err)
	}

	// Load Gemini config using the utility function
	apiKey, modelName, err := utils.LoadGeminiConfigWithFallback(*configPath)
	if err != nil {
		return err // utils.LoadGeminiConfigWithFallback already logs detailed errors.
	}

	// Determine number of chapters for the *initial* abstract generation
	numChapters := *chapters
	if numChapters == 0 {
		// Seed the random number generator
		rand.Seed(time.Now().UnixNano())
		// Generate a random number between 20 and 40 (inclusive)
		numChapters = rand.Intn(21) + 20 // rand.Intn(n) generates [0, n-1], so 21 gives [0, 20]. Adding 20 shifts it to [20, 40].
		log.Printf("Number of chapters not specified for abstract generation. Generating a random number: %d", numChapters)
	} else {
		log.Printf("Using specified number of chapters: %d for abstract generation", numChapters)
	}

	// --- Generate Abstract ---
	log.Printf("Initiating abstract generation using Gemini model: %s, output language: %s, chapters: %d", modelName, *language, numChapters)
	abstract, err := generateAbstract(apiKey, modelName, *instruction, *language, numChapters)
	if err != nil {
		return fmt.Errorf("error generating abstract: %w", err)
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
		return fmt.Errorf("error saving abstract to file '%s': %w", finalOutputPath, err)
	}

	fmt.Printf("Abstract successfully generated and saved to: %s\n", finalOutputPath)
	log.Printf("Abstract saved to: %s", finalOutputPath)

	// --- New Step: Get pure chapter count from Gemini ---
	log.Printf("Sending abstract to Gemini to get pure chapter count...")
	pureChapterCount, err := getChapterCountFromGemini(apiKey, modelName, abstract)
	if err != nil {
		log.Printf("Warning: Failed to get pure chapter count from Gemini: %v. Proceeding without this information.", err)
	} else {
		fmt.Printf("Pure chapter count from Gemini: %d\n", pureChapterCount)
		log.Printf("Pure chapter count from Gemini: %d", pureChapterCount)
	}

	return nil
}