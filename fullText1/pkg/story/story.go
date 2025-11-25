package story

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zicongmei/ai-story/fullText1/pkg/abstract/file"
	"github.com/zicongmei/ai-story/fullText1/pkg/aiEndpoint"
)

// GetChapterCountForStoryInput holds input parameters for getChapterCountFromGeminiForStory.
type GetChapterCountForStoryInput struct {
	APIKey        string
	ModelName     string
	ThinkingLevel string
	Abstract      string
}

// getChapterCountFromGeminiForStory sends the abstract to Gemini to get a pure chapter count for story generation.
func getChapterCountFromGeminiForStory(input GetChapterCountForStoryInput) aiEndpoint.ChapterCountResult { // Updated signature
	var result aiEndpoint.ChapterCountResult

	prompt := fmt.Sprintf(`Given the following complete story abstract (plan), please return ONLY the total number of chapters planned within it.
Do not include any other text, explanation, or formatting. Just the pure number.
If no chapters are explicitly outlined, return 0.

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
		result.Err = fmt.Errorf("error calling Gemini to get chapter count for story: %w", apiResponse.Err)
		return result
	}

	countStr := strings.TrimSpace(apiResponse.GeneratedText)
	countStr = strings.Split(countStr, "\n")[0]

	count, err := strconv.Atoi(countStr)
	if err != nil {
		result.InputTokens = apiResponse.InputTokens
		result.OutputTokens = apiResponse.OutputTokens
		result.Cost = apiResponse.Cost
		result.Err = fmt.Errorf("could not parse chapter count '%s' from Gemini response for story: %w", countStr, err)
		return result
	}

	result.Count = count
	result.InputTokens = apiResponse.InputTokens
	result.OutputTokens = apiResponse.OutputTokens
	result.Cost = apiResponse.Cost
	return result
}

// GetWrittenChapterCountInput holds input parameters for getWrittenChapterCountFromGemini.
type GetWrittenChapterCountInput struct {
	APIKey               string
	ModelName            string
	ThinkingLevel        string
	ExistingStoryContent string
}

// getWrittenChapterCountFromGemini sends the existing story content to Gemini to identify the last fully written chapter.
func getWrittenChapterCountFromGemini(input GetWrittenChapterCountInput) aiEndpoint.ChapterCountResult { // Updated signature
	var result aiEndpoint.ChapterCountResult

	prompt := fmt.Sprintf(`Given the following story text, identify the number of the last *fully written* chapter.
Look for chapter headers like '## Chapter X' (where X is the chapter number).
Return ONLY the number.
If no fully written chapters are found, or if the last detected chapter appears incomplete (e.g., ends abruptly or contains error messages), return 0.
Do not include any other text, explanation, or formatting. Just the pure integer number.

--- Existing Story Content ---
%s
--- End Existing Story Content ---
`, input.ExistingStoryContent)

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
		result.Err = fmt.Errorf("error calling Gemini to get written chapter count: %w", apiResponse.Err)
		return result
	}

	countStr := strings.TrimSpace(apiResponse.GeneratedText)
	countStr = strings.Split(countStr, "\n")[0]

	count, err := strconv.Atoi(countStr)
	if err != nil {
		result.InputTokens = apiResponse.InputTokens
		result.OutputTokens = apiResponse.OutputTokens
		result.Cost = apiResponse.Cost
		result.Err = fmt.Errorf("could not parse written chapter count '%s' from Gemini response: %w", countStr, err)
		return result
	}

	result.Count = count
	result.InputTokens = apiResponse.InputTokens
	result.OutputTokens = apiResponse.OutputTokens
	result.Cost = apiResponse.Cost
	return result
}

// FullStoryConfig holds all configuration needed for story generation.
type FullStoryConfig struct {
	ConfigPath       string
	AbstractFilePath string
	WordsPerChapter  int
	OutputPath       string
	APIKey           string
	ModelName        string
	ThinkingLevel    string
	AbstractContent  string
}

// StoryProgressState holds the current state of the story generation,
// including accumulated tokens and the generated content for context.
type StoryProgressState struct {
	AccumulatedInputTokens  int
	AccumulatedOutputTokens int
	AccumulatedCost         float64
	PreviousChapters        string                  // Content of all chapters written so far, for context
	CurrentHistory          *aiEndpoint.HistoryTurn // Last AI interaction for thought signature chain
	ChaptersAlreadyWritten  int
	FirstNewChapter         int
	FileMode                int // os.O_CREATE|os.O_WRONLY or os.O_APPEND|os.O_WRONLY
}

// parseAndValidateFlags parses command-line flags and performs initial validation.
func parseAndValidateFlags(args []string) (FullStoryConfig, error) {
	var cfg FullStoryConfig
	cmd := flag.NewFlagSet("story", flag.ContinueOnError)
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "Usage of %s story:\n", os.Args[0])
		cmd.PrintDefaults()
	}

	cmd.StringVar(&cfg.ConfigPath, "config", "", "Path to Gemini configuration JSON file (optional). If not provided, API key is taken from GEMINI_API_KEY env var and model defaults to 'gemini-pro'.")
	cmd.StringVar(&cfg.AbstractFilePath, "abstract", "", "Path to the abstract file (text, json, or yaml) generated by the 'abstract' command.")
	cmd.IntVar(&cfg.WordsPerChapter, "words-per-chapter", 5000, "Desired average number of words per chapter (actual count may vary by +/- 20%).")
	cmd.StringVar(&cfg.OutputPath, "output", "", "Path to save the generated full story file (default: fulltext-yyyy-mm-dd-hh-mm-ss.txt based on abstract filename).")

	if err := cmd.Parse(args); err != nil {
		return cfg, fmt.Errorf("failed to parse story subcommand flags: %w", err)
	}

	if cfg.AbstractFilePath == "" {
		return cfg, fmt.Errorf("--abstract is required for story generation")
	}
	if cfg.WordsPerChapter <= 0 {
		return cfg, fmt.Errorf("--words-per-chapter must be a positive number")
	}
	return cfg, nil
}

// setupLogging configures file-based logging. It returns the opened log file, which the caller must close.
func setupLogging(abstractFilePath string) (*os.File, error) {
	originalLogOutput := log.Writer()
	originalLogFlags := log.Flags()

	abstractFileName := filepath.Base(abstractFilePath)
	logFileName := ""
	if strings.HasPrefix(strings.ToLower(abstractFileName), "abstract-") && (strings.HasSuffix(strings.ToLower(abstractFileName), ".txt") || strings.HasSuffix(strings.ToLower(abstractFileName), ".json") || strings.HasSuffix(strings.ToLower(abstractFileName), ".yaml") || strings.HasSuffix(strings.ToLower(abstractFileName), ".yml")) {
		logFileName = strings.Replace(abstractFileName, "abstract-", "log-", 1)
		logFileName = strings.Replace(logFileName, ".txt", ".log", 1)
		logFileName = strings.Replace(logFileName, ".json", ".log", 1)
		logFileName = strings.Replace(logFileName, ".yaml", ".log", 1)
		logFileName = strings.Replace(logFileName, ".yml", ".log", 1)
	} else {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		logFileName = fmt.Sprintf("story-log-%s.log", timestamp)
	}

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: Failed to open log file '%s': %v. Logging will continue to stderr.", logFileName, err)
		log.SetOutput(originalLogOutput) // Ensure logging goes to original output if file fails
		log.SetFlags(originalLogFlags)
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	mw := io.MultiWriter(os.Stderr, logFile)
	log.SetOutput(mw)
	log.Printf("Logging to file: %s", logFileName)
	return logFile, nil
}

// loadGeminiAPIConfig loads Gemini API key, model name, and thinking level.
func loadGeminiAPIConfig(configPath string) (string, string, string, error) {
	geminiConfigDetails := aiEndpoint.LoadGeminiConfigWithFallback(configPath)
	if geminiConfigDetails.Err != nil {
		return "", "", "", geminiConfigDetails.Err
	}
	return geminiConfigDetails.APIKey, geminiConfigDetails.ModelName, geminiConfigDetails.ThinkingLevel, nil
}

// readAbstractAndDetermineTotalChapters reads the abstract file and gets the total planned chapters from Gemini.
func readAbstractAndDetermineTotalChapters(cfg FullStoryConfig) (string, int, int, int, float64, error) {
	abstractContent, _, err := file.ReadAbstractFile(cfg.AbstractFilePath) // thoughtSignature is currently unused
	if err != nil {
		return "", 0, 0, 0, 0, fmt.Errorf("failed to read and parse abstract file '%s': %w", cfg.AbstractFilePath, err)
	}

	log.Printf("Sending abstract to Gemini to get the total number of chapters planned...")
	getChapterCountForStoryInput := GetChapterCountForStoryInput{
		APIKey:        cfg.APIKey,
		ModelName:     cfg.ModelName,
		ThinkingLevel: cfg.ThinkingLevel,
		Abstract:      abstractContent,
	}
	chapterCountPlanResult := getChapterCountFromGeminiForStory(getChapterCountForStoryInput)
	if chapterCountPlanResult.Err != nil {
		return "", 0, 0, 0, 0, fmt.Errorf("failed to get total chapter count from Gemini for story generation: %w", chapterCountPlanResult.Err)
	}

	totalChapters := chapterCountPlanResult.Count
	if totalChapters == 0 {
		return "", 0, 0, 0, 0, fmt.Errorf("Gemini returned 0 planned chapters for the abstract. Cannot proceed with story generation.")
	}
	log.Printf("Chapter plan determination complete. Input tokens: %d, Output tokens: %d, Cost: $%.6f", chapterCountPlanResult.InputTokens, chapterCountPlanResult.OutputTokens, chapterCountPlanResult.Cost)
	fmt.Printf("Total chapters identified by Gemini for story generation: %d\n", totalChapters)
	log.Printf("Total chapters identified by Gemini for story generation: %d", totalChapters)

	return abstractContent, totalChapters, chapterCountPlanResult.InputTokens, chapterCountPlanResult.OutputTokens, chapterCountPlanResult.Cost, nil
}

// determineOutputFilePath calculates the final output file path.
func determineOutputFilePath(abstractFilePath, outputPathFlag string) string {
	if outputPathFlag != "" {
		return outputPathFlag
	}

	re := strings.NewReplacer("abstract-", "fulltext-")
	baseName := filepath.Base(abstractFilePath)
	if strings.HasPrefix(strings.ToLower(baseName), "abstract-") {
		finalOutputPath := re.Replace(baseName)
		finalOutputPath = strings.Replace(finalOutputPath, ".json", ".txt", 1)
		finalOutputPath = strings.Replace(finalOutputPath, ".yaml", ".txt", 1)
		finalOutputPath = strings.Replace(finalOutputPath, ".yml", ".txt", 1)
		return finalOutputPath
	}

	timestamp := time.Now().Format("2006-01-02-15-04-05")
	return fmt.Sprintf("fulltext-%s.txt", timestamp)
}

// initializeStoryStateForResume checks for existing story content and determines the resume point.
// It updates the StoryProgressState with details for resuming.
func initializeStoryStateForResume(cfg FullStoryConfig, finalOutputPath string) (StoryProgressState, error) {
	state := StoryProgressState{
		ChaptersAlreadyWritten: 0,
		FirstNewChapter:        1,
		FileMode:               os.O_CREATE | os.O_WRONLY,
		PreviousChapters:       "",
	}

	if _, err := os.Stat(finalOutputPath); err == nil {
		log.Printf("Output file '%s' already exists. Checking for existing chapters to resume...", finalOutputPath)
		contentBytes, readErr := os.ReadFile(finalOutputPath)
		if readErr != nil {
			log.Printf("Warning: Failed to read existing file '%s' to determine written chapters: %v. Starting from Chapter 1 (will overwrite if abstract/header is incomplete).", finalOutputPath, readErr)
			// state.FileMode remains os.O_CREATE | os.O_WRONLY
		} else {
			existingStoryContent := string(contentBytes)
			log.Printf("Sending existing content to Gemini to identify written chapters for resume...")
			getWrittenChapterCountInput := GetWrittenChapterCountInput{
				APIKey:               cfg.APIKey,
				ModelName:            cfg.ModelName,
				ThinkingLevel:        cfg.ThinkingLevel,
				ExistingStoryContent: existingStoryContent,
			}
			writtenChapterCountResult := getWrittenChapterCountFromGemini(getWrittenChapterCountInput)
			state.AccumulatedInputTokens += writtenChapterCountResult.InputTokens
			state.AccumulatedOutputTokens += writtenChapterCountResult.OutputTokens
			state.AccumulatedCost += writtenChapterCountResult.Cost

			if writtenChapterCountResult.Err != nil {
				log.Printf("Warning: Failed to get written chapter count from Gemini for existing file: %v. Assuming 0 chapters written and starting from Chapter 1 (will overwrite if abstract/header is incomplete).", writtenChapterCountResult.Err)
				// state.FileMode remains os.O_CREATE | os.O_WRONLY
			} else {
				state.ChaptersAlreadyWritten = writtenChapterCountResult.Count
				if state.ChaptersAlreadyWritten > 0 {
					state.FirstNewChapter = state.ChaptersAlreadyWritten + 1
					state.FileMode = os.O_APPEND | os.O_WRONLY    // Open in append mode
					state.PreviousChapters = existingStoryContent // Start context with existing content
					log.Printf("Detected %d chapters already written in '%s'. Resuming generation from Chapter %d.", state.ChaptersAlreadyWritten, finalOutputPath, state.FirstNewChapter)
				} else {
					log.Printf("No complete chapters detected in existing file '%s'. Starting from Chapter 1 (will overwrite if abstract/header is incomplete).", finalOutputPath)
					// state.FileMode remains os.O_CREATE | os.O_WRONLY
				}
			}
		}
	} else {
		log.Printf("Output file '%s' does not exist. Starting new story generation from Chapter 1.", finalOutputPath)
		// state.FileMode remains os.O_CREATE | os.O_WRONLY
	}
	return state, nil
}

// writeInitialStoryHeader writes the header and abstract to the file if starting a new story (chaptersAlreadyWritten == 0).
func writeInitialStoryHeader(f *os.File, abstractContent string, chaptersAlreadyWritten int) error {
	if chaptersAlreadyWritten == 0 {
		if _, err := fmt.Fprintf(f, "--- Full Story: %s ---\n\n", time.Now().Format("2006-01-02 15:04:05")); err != nil {
			return fmt.Errorf("failed to write story header: %w", err)
		}
		if _, err := fmt.Fprintf(f, "Story Plan Abstract:\n%s\n\n", abstractContent); err != nil {
			return fmt.Errorf("failed to write abstract to file: %w", err)
		}
		if _, err := fmt.Fprintf(f, "----------------------------------------\n\n"); err != nil {
			return fmt.Errorf("failed to write separator to file: %w", err)
		}
	}
	return nil
}

// generateStoryChapters loops through and generates each chapter, writing it to the file.
// It updates the StoryProgressState with token counts, cost, and accumulated content.
func generateStoryChapters(
	f *os.File,
	cfg FullStoryConfig,
	totalChapters int,
	state *StoryProgressState,
) error {
	log.Printf("Starting full story generation from Chapter %d to Chapter %d, aiming for %d words per chapter...",
		state.FirstNewChapter, totalChapters, cfg.WordsPerChapter)

	const maxChapterRetries = 3 // Number of retries for chapter generation
	previousChapterThoughtSignature := []byte{}

	for i := state.FirstNewChapter - 1; i < totalChapters; i++ {
		chapterNum := i + 1

		log.Printf("Generating Chapter %d (out of %d)", chapterNum, totalChapters)

		prompt := fmt.Sprintf(`Given the following complete story abstract (plan) and the chapters already written, please write Chapter %d of the story.
Generate a short title for the charpter.
The chapter should be approximately %d words. Focus on progressing the narrative as outlined in the abstract for this specific chapter.

--- Full Story Abstract (Plan) ---
%s
--- End Full Story Abstract (Plan) ---

--- Previously Written Chapters (including abstract and previous chapters) ---
%s
--- End Previously Written Chapters ---

Write Chapter %d now, ensuring it flows logically from previous chapters and adheres to the overall story plan.
`,
			chapterNum,
			cfg.WordsPerChapter,
			cfg.AbstractContent,
			state.PreviousChapters,
			chapterNum,
		)

		var chapterText string
		var chapterSignature []byte
		var chapterInputTokens, chapterOutputTokens int
		var chapterCost float64
		var chapterGenerationErr error

		// Retry logic for CallGeminiAPI for chapter generation
		for attempt := 0; attempt <= maxChapterRetries; attempt++ {
			if attempt > 0 {
				log.Printf("Retrying Chapter %d (attempt %d/%d) after previous failure: %v", chapterNum, attempt, maxChapterRetries, chapterGenerationErr)
				time.Sleep(2 * time.Second) // Small delay before retrying
			}

			apiInput := aiEndpoint.CallGeminiAPIInput{
				Ctx:           context.Background(),
				APIKey:        cfg.APIKey,
				ModelName:     cfg.ModelName,
				Prompt:        prompt,
				ThinkingLevel: cfg.ThinkingLevel,
				//PreviousTurn:     state.CurrentHistory, // remove redundent content
				ThoughtSignature: previousChapterThoughtSignature,
			}
			apiResponse := aiEndpoint.CallGeminiAPI(apiInput)

			chapterText = apiResponse.GeneratedText
			chapterSignature = apiResponse.ThoughtSignature
			chapterInputTokens = apiResponse.InputTokens
			chapterOutputTokens = apiResponse.OutputTokens
			chapterCost = apiResponse.Cost
			chapterGenerationErr = apiResponse.Err

			if chapterGenerationErr == nil {
				// Success, break out of retry loop
				break
			}
		}

		if chapterGenerationErr != nil {
			log.Fatalf("Critical Error: Failed to generate Chapter %d after %d attempts: %v. Marking chapter with error message and proceeding.", chapterNum, maxChapterRetries+1, chapterGenerationErr)
			// If all retries fail, mark the chapter with an error message in the output.
			chapterText = fmt.Sprintf("Error generating Chapter %d: %v\n\n[Generation Failed - Please review logs]", chapterNum, chapterGenerationErr)
			chapterSignature = nil // Clear signature if generation failed
			chapterInputTokens = 0
			chapterOutputTokens = 0
			chapterCost = 0
		}

		// Accumulate token counts and cost
		state.AccumulatedInputTokens += chapterInputTokens
		state.AccumulatedOutputTokens += chapterOutputTokens
		state.AccumulatedCost += chapterCost
		log.Printf("Chapter %d tokens: Input %d, Output %d, Cost: $%.6f. Accumulated tokens: Input %d, Output %d, Accumulated Cost: $%.6f",
			chapterNum, chapterInputTokens, chapterOutputTokens, chapterCost, state.AccumulatedInputTokens, state.AccumulatedOutputTokens, state.AccumulatedCost)

		// Write the generated chapter directly to the file
		chapterHeader := fmt.Sprintf("## Chapter %d\n\n", chapterNum)
		chapterContentToWrite := strings.TrimSpace(chapterText) + "\n\n"

		if _, err := f.WriteString(chapterHeader); err != nil {
			log.Printf("Error writing chapter header for Chapter %d to file: %v", chapterNum, err)
			return fmt.Errorf("failed to write chapter header: %w", err) // Critical error
		}
		if _, err := f.WriteString(chapterContentToWrite); err != nil {
			log.Printf("Error writing Chapter %d content to file: %v", chapterNum, err)
			return fmt.Errorf("failed to write chapter content: %w", err) // Critical error
		}
		log.Printf("Chapter %d generated and written to file.", chapterNum)

		// Append the newly generated chapter to previousChapters for the *next* iteration's context
		state.PreviousChapters += chapterHeader + chapterContentToWrite

		// Update the history for the next iteration to maintain the thought signature chain
		state.CurrentHistory = &aiEndpoint.HistoryTurn{
			UserPrompt:       prompt,
			ModelResponse:    chapterText,
			ThoughtSignature: chapterSignature,
		}
		previousChapterThoughtSignature = chapterSignature

		// Add a small delay to avoid hitting rate limits if generating many chapters quickly
		time.Sleep(1 * time.Second)
	}
	return nil
}

// Execute is the main entry point for the 'story' subcommand.
func Execute(args []string) error {
	// 1. Parse and validate flags
	cfg, err := parseAndValidateFlags(args)
	if err != nil {
		return err
	}

	// Preserve original log output and flags to restore later
	originalLogOutput := log.Writer()
	originalLogFlags := log.Flags()
	defer func() {
		log.SetOutput(originalLogOutput) // Restore original log output when Execute exits
		log.SetFlags(originalLogFlags)   // Restore original log flags
	}()

	// 2. Configure logging
	logFile, err := setupLogging(cfg.AbstractFilePath)
	if err != nil {
		// setupLogging already logs a warning and ensures logging goes to stderr.
		// No need to return an error here, just proceed with stderr logging.
	}
	if logFile != nil {
		defer logFile.Close() // Ensure the log file is closed
	}

	// 3. Load Gemini configuration
	cfg.APIKey, cfg.ModelName, cfg.ThinkingLevel, err = loadGeminiAPIConfig(cfg.ConfigPath)
	if err != nil {
		return err
	}

	// 4. Read abstract and determine total chapters
	var totalChapters int
	var initialInputTokens, initialOutputTokens int
	var initialCost float64
	cfg.AbstractContent, totalChapters, initialInputTokens, initialOutputTokens, initialCost, err = readAbstractAndDetermineTotalChapters(cfg)
	if err != nil {
		return err
	}

	// 5. Determine output path
	finalOutputPath := determineOutputFilePath(cfg.AbstractFilePath, cfg.OutputPath)

	// 6. Initialize story state (resume logic)
	state, err := initializeStoryStateForResume(cfg, finalOutputPath)
	if err != nil {
		return err
	}
	// Add initial token counts from chapter planning and resume checks
	state.AccumulatedInputTokens += initialInputTokens
	state.AccumulatedOutputTokens += initialOutputTokens
	state.AccumulatedCost += initialCost

	// 7. Open the output file for writing based on determined mode
	f, err := os.OpenFile(finalOutputPath, state.FileMode, 0644)
	if err != nil {
		return fmt.Errorf("error opening/creating output file '%s': %w", finalOutputPath, err)
	}
	defer f.Close()

	// 8. Write initial header if starting fresh
	if err := writeInitialStoryHeader(f, cfg.AbstractContent, state.ChaptersAlreadyWritten); err != nil {
		return err
	}

	// 9. Generate story chapter by chapter
	if err := generateStoryChapters(f, cfg, totalChapters, &state); err != nil {
		return err
	}

	fmt.Printf("Full story successfully generated and saved to: %s\n", finalOutputPath)
	log.Printf("Full story saved to: %s. Total accumulated tokens: Input %d, Output %d. Total accumulated cost: $%.6f", finalOutputPath, state.AccumulatedInputTokens, state.AccumulatedOutputTokens, state.AccumulatedCost)
	fmt.Printf("Total accumulated cost for full story generation process: $%.6f\n", state.AccumulatedCost)

	return nil
}
