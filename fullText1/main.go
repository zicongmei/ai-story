package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zicongmei/ai-story/fullText1/pkg/abstract"
)

func main() {
	// Configure logging to include file and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "abstract":
		if err := abstract.Execute(os.Args[2:]); err != nil {
			log.Fatalf("Abstract subcommand failed: %v", err)
		}
	case "story":
		// Placeholder for future story generation logic
		fmt.Println("Story generation subcommand (future feature).")
		fmt.Println("Usage: ai-story story [args...]")
		// Example: go run main.go story --abstract-file path/to/abstract.txt --output story.txt
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown subcommand: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: ai-story <command> [arguments]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  abstract  Generate a story abstract/plan using Gemini API.")
	fmt.Println("  story     Generate a full story from an abstract (future feature).")
	fmt.Println("\nRun 'ai-story abstract --help' for abstract subcommand options.")
}