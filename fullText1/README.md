# AI Story Abstract Generator

This Go program uses the Google Gemini API to generate an abstract for a given story instruction. It allows you to optionally specify a Gemini configuration file (containing your API key and model name) or rely on environment variables. You can also specify an output path for the generated abstract.

## Features

*   **Flexible Gemini API Configuration:** API key can be provided via a JSON configuration file (if `--config` is used) or the `GEMINI_API_KEY` environment variable. Model name can be specified in the config file or defaults to `gemini-pro`.
*   **Default Settings:** Sensible defaults for output file name (`abstract-yyyy-mm-dd-hh-mm-ss.txt`). No default configuration file is assumed; if `--config` is not used, environment variables are checked.
*   **Flexible Input:** Takes story instructions as an *optional* command-line argument. If omitted, a default generic story instruction ("A story about courage and discovery.") is used.
*   **Persistent Output:** Saves the generated abstract to a specified (or default) text file.

## Installation

1.  **Install Go:** If you don't have Go installed, follow the instructions on the [official Go website](https://go.dev/doc/install).

2.  **Clone the repository (or create the file structure):**
    ```bash
    mkdir -p /usr/local/google/home/zicong/code/src/github.com/zicongmei/ai-story/fullText1
    # Then place createAbstract.go in this directory
    ```

3.  **Install the Google Generative AI Go library:**
    Navigate to the directory containing `createAbstract.go` (or your project root) and run:
    ```bash
    go mod init github.com/zicongmei/ai-story/fullText1 # If not already a module
    go get github.com/google/generative-ai-go/genai
    ```

## Configuration

The Gemini API configuration is now flexible and optional.
If the `--config` flag is provided, the program attempts to load the Gemini configuration from that file. If `--config` is omitted, or if the specified config file is not found, unreadable, or doesn't contain an API key, the program falls back to using the `GEMINI_API_KEY` environment variable.

### API Key Precedence:
1.  `api_key` from the specified JSON configuration file (if `--config` is used).
2.  `GEMINI_API_KEY` environment variable.
If neither is found, the program will exit with an error.

### Model Name Precedence:
1.  `model_name` from the specified JSON configuration file (if `--config` is used).
2.  If `model_name` is omitted from the config file, or if no config file is used, it defaults to `gemini-pro`.

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
    *   **`model_name`**: (Optional) Specify the Gemini model to use. If omitted, the program defaults to `gemini-pro`. Common valid models include `gemini-1.5-pro` or `gemini-pro`.

    You must then provide the path to this file using the `--config` flag when running the program.

### Using Environment Variable (Recommended for quick setup or no custom config)

If you don't provide a `--config` flag, the program will look for your API key in the `GEMINI_API_KEY` environment variable and use `gemini-pro` as the model.

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
# Then run the program without --config
```

## Usage

Navigate to the directory where `createAbstract.go` is located and run the program.

### Basic Usage (using environment variable)

To use your API key from an environment variable and the default model (`gemini-pro`), simply omit the `--config` flag. Make sure `GEMINI_API_KEY` is set:

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
go run createAbstract.go \
    --instruction "A futuristic detective hunts a rogue AI in a sprawling cyberpunk city, uncovering its motives for disrupting the automated utopia."
```

The abstract will be saved to a file like `abstract-2023-10-27-10-30-45.txt` in the current directory.

### Using the default instruction (no `--instruction` flag)

If you omit the `--instruction` flag, the program will use a default instruction ("A story about courage and discovery.") to generate an abstract.

```bash
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
go run createAbstract.go
# This will generate an abstract for "A story about courage and discovery."
```

### Custom Output Path

Specify a custom file path for the abstract:

```bash
go run createAbstract.go \
    --instruction "In a dystopian future, a rebel hacker uncovers a vast conspiracy by the ruling AI, risking everything to expose the truth and free humanity." \
    --output "my_story_abstract.txt"
```

### Custom Configuration File

Use a different path for your Gemini configuration:

```bash
go run createAbstract.go \
    --config "./my_custom_gemini_config.json" \
    --instruction "A detective with a troubled past is assigned a case involving a series of impossible disappearances in a secluded mountain town."
```

### All Options

```bash
go run createAbstract.go \
    --config "/home/user/my_gemini_keys/config.json" \
    --output "fantasy_abstract.txt" \
    --instruction "An ancient artifact awakens, granting its wielder immense power but also attracting a malevolent entity from another dimension."
```