# AI Story Abstract Generator

This Go program provides a command-line interface to interact with the Google Gemini API, primarily for generating story abstracts and full stories.

## Features

*   **Subcommand-based CLI:** Uses `abstract` subcommand to generate story plans and a `story` subcommand for full story generation.
*   **Flexible Gemini API Configuration:** API key can be provided via a JSON configuration file (if `--config` is used) or the `GEMINI_API_KEY` environment variable. Model name can be specified in the config file or defaults to `gemini-pro`.
*   **Output Language Control:** Specify the desired language for the generated abstract using the `--language` flag.
*   **Chapter Count Control:** Specify the desired number of chapters using the `--chapters` flag for the abstract.
*   **Detailed Token and Cost Logging:** Logs input and output token counts and estimated cost for every Gemini API call. For story generation, it also logs accumulated input and output token counts and total estimated cost across all chapter generations.
*   **Story Generation from Abstract:** The `story` subcommand takes an abstract and generates the full text, chapter by chapter, adhering to a specified word count per chapter, **sending the entire abstract to the AI as context for each chapter generation**. Each generated chapter is immediately appended to the output file.
*   **Robust Chapter Generation:** When generating individual story chapters, if `utils.CallGeminiAPI` encounters an error, the program will automatically **retry up to 3 times** to regenerate that chapter before marking it with an error message and continuing. This improves resilience against transient API issues.
*   **Resume Generation:** If the `--output` file already exists, the program will send its content to Gemini to identify the number of previously written chapters. Generation will then resume from the next missing chapter. The full content of the existing file (including abstract and previously written chapters) is sent as context for the first new chapter, and subsequent newly generated chapters are appended to this context for continuous flow.
*   **Default Settings:** Sensible defaults for output file name (`abstract-yyyy-mm-dd-hh-mm-ss.txt` or `fulltext-yyyy-mm-dd-hh-mm-ss.txt`). No default configuration file is assumed; if `--config` is not used, environment variables are checked.
*   **Flexible Input:** Takes story instructions as an *optional* command-line argument for the `abstract` subcommand.
*   **Persistent Output:** Saves the generated abstract or full story to a specified (or default) text file.
*   **Dynamic Thinking Budget:** The Gemini API calls are configured with `ThinkingBudget: -1`, enabling dynamic thinking by the model.

## Installation

1.  **Install Go:** If you don't have Go installed, follow the instructions on the [official Go website](https://go.dev/doc/install).

2.  **Clone the repository (or create the file structure):**
    ```bash
    mkdir -p /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/pkg/abstract
    mkdir -p /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/pkg/story
    mkdir -p /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/pkg/utils
    # Place main.go in /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/
    # Place createAbstract.go in /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/pkg/abstract/
    # Place story.go in /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/pkg/story/
    # Place gemini.go in /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1/pkg/utils/
    ```

3.  **Install the Google Generative AI Go library:**
    Navigate to the project root (`/usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1`) and run:
    ```bash
    go mod init github.com/zicongmei/ai-story/fullText1 # If not already a module
    go get github.com/google/generative-ai-go/genai
    ```

## Configuration

The Gemini API configuration is now flexible and optional.
If the `--config` flag is provided with either the `abstract` or `story` subcommand, the program attempts to load the Gemini configuration from that file. If `--config` is omitted, or if the specified config file is not found, unreadable, or doesn't contain an API key, the program falls back to using the `GEMINI_API_KEY` environment variable.

### API Key Precedence (for all subcommands):
1.  `api_key` from the specified JSON configuration file (if `--config` is used).
2.  `GEMINI_API_KEY` environment variable.
If neither is found, the program will exit with an error.

### Model Name Precedence (for all subcommands):
1.  `model_name` from the specified JSON configuration file (if `--config` is used).
2.  If `model_name` is omitted from the config file, or if no config file is used, it defaults to `gemini-2.5-flash` (as defined in code's `DefaultGeminiModel`). **Note:** The example config `gemini-1.5-pro` maps to `gemini-2.5-pro` for pricing purposes.

### Using a Configuration File (Optional, requires `--config` flag)

You can create a custom JSON file to store your API key and model name.

1.  **Create your `gemini.json` file:**
    Create a file named `my_gemini_config.json` (or any name you prefer) in a location of your choice, with the following content:

    ```json
    {
      "api_key": "YOUR_GEMINI_API_KEY",
      "model_name": "gemini-1.5-pro"
    }
    ```
    *   **`api_key`**: Replace `YOUR_GEMINI_API_KEY` with your actual Google Gemini API key. You can obtain one from the [Google AI Studio](https://makersuite.google.com/keys). If omitted here, the `GEMINI_API_KEY` environment variable will be used as a fallback.
    *   **`model_name`**: (Optional) Specify the Gemini model to use. If omitted, the program defaults to `gemini-2.5-flash`. Common valid models include `gemini-1.5-pro` (mapped to `gemini-2.5-pro` for pricing) or `gemini-2.5-flash`.

    You must then provide the path to this file using the `--config` flag when running either `abstract` or `story` subcommand.

### Using Environment Variable (Recommended for quick setup or no custom config)

If you don't provide a `--config` flag to either subcommand, the program will look for your API key in the `GEMINI_API_KEY` environment variable and use `gemini-2.5-flash` as the model.

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
# Then run the program without --config for either subcommand
```

## Usage

Navigate to the project root (`/usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1`) and run the program using subcommands.

### Main Program Help

```bash
go run main.go help
# Or simply:
go run main.go
```

### Abstract Subcommand

To generate an abstract, use the `abstract` subcommand.

#### Basic Usage (using environment variable)

To use your API key from an environment variable and the default model (`gemini-2.5-flash`), simply omit the `--config` flag. Make sure `GEMINI_API_KEY` is set:

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
go run main.go abstract \
    --instruction "A futuristic detective hunts a rogue AI in a sprawling cyberpunk city, uncovering its motives for disrupting the automated utopia."
```

The abstract will be saved to a file like `abstract-2023-10-27-10-30-45.txt` in the current directory.

#### Using the default instruction (no `--instruction` flag)

If you omit the `--instruction` flag, the program will use a default instruction ("A story about courage and discovery.") to generate an abstract.

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
go run main.go abstract
# This will generate an abstract for "A story about courage and discovery."
```

#### Custom Output Path

Specify a custom file path for the abstract:

```bash
go run main.go abstract \
    --instruction "In a dystopian future, a rebel hacker uncovers a vast conspiracy by the ruling AI, risking everything to expose the truth and free humanity." \
    --output "my_story_abstract.txt"
```

#### Custom Configuration File

Use a different path for your Gemini configuration:

```bash
go run main.go abstract \
    --config "./my_custom_gemini_config.json" \
    --instruction "A detective with a troubled past is assigned a case involving a series of impossible disappearances in a secluded mountain town."
```

#### Specifying Output Language

Generate the abstract in a language other than English (e.g., Spanish):

```bash
go run main.go abstract \
    --instruction "A brave knight embarks on a quest to save a princess from a dragon." \
    --language "spanish" \
    --output "spanish_abstract.txt"
```

#### Specifying Number of Chapters

Provide a specific number of chapters for the story plan:

```bash
go run main.go abstract \
    --language chinese \
    --chapters 30
```

If `--chapters` is not provided, a random number between 20-40 will be used.

*   **Chapter Count Extraction:** After generating and saving the abstract, the program performs an additional API call to Gemini to extract and display *only* the total number of chapters identified within the abstract. This provides a clean, numeric output for the chapter count before proceeding to full story generation.
*   **Token & Cost Logging:** Input and output token counts for each Gemini API call (abstract generation and chapter count extraction), along with their estimated costs, are logged to the console. The total accumulated cost for the abstract generation process is also displayed.

#### All Options for Abstract Subcommand

```bash
go run main.go abstract \
    --config "/home/user/my_gemini_keys/config.json" \
    --output "fantasy_abstract.txt" \
    --instruction "An ancient artifact awakens, granting its wielder immense power but also attracting a malevolent entity from another dimension." \
    --language "french" \
    --chapters 25
```

### Story Subcommand

To generate a full story from an existing abstract, use the `story` subcommand. **The full abstract text is provided to the AI as context for each chapter generation, allowing the model to understand the overall narrative arc.**

#### Basic Usage (using environment variable)

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
go run main.go story \
    --abstract-file "abstract-2023-10-27-10-30-45.txt" \
    --words-per-chapter 500
```
This will generate a full story based on `abstract-2023-10-27-10-30-45.txt`, with each chapter aiming for around 500 words (actual word count may vary by +/- 20%). The output file will be named `fulltext-2023-10-27-10-30-45.txt`. Each chapter will be written to the output file immediately after generation.

*   **Token & Cost Logging:** Input and output token counts and estimated costs for each Gemini API call (chapter count extraction and individual chapter generation) are logged to the console. Accumulated input and output token counts and total estimated cost for the entire story generation process are also logged and displayed after all chapters are generated.

#### Custom Output Path and Configuration

```bash
go run main.go story \
    --config "./my_custom_gemini_config.json" \
    --abstract-file "my_story_abstract.txt" \
    --words-per-chapter 750 \
    --output "full_story_epic.txt"
```

#### All Options for Story Subcommand

```bash
go run main.go story \
    --config "/home/user/my_gemini_keys/config.json" \
    --abstract-file "fantasy_abstract.txt" \
    --words-per-chapter 600 \
    --output "generated_fantasy_story.txt"
```