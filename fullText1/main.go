package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zicongmei/ai-story/fullText1/pkg/abstract"
	"github.com/zicongmei/ai-story/fullText1/pkg/story" // Import the new story package
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
		if err := story.Execute(os.Args[2:]); err != nil {
			log.Fatalf("Story subcommand failed: %v", err)
		}
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
	fmt.Println("  story     Generate a full story from an abstract.")
	fmt.Println("\nRun 'ai-story abstract --help' for abstract subcommand options.")
	fmt.Println("Run 'ai-story story --help' for story subcommand options.")
}