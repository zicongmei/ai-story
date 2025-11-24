package file

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// AbstractOutput structure for YAML/JSON output
type AbstractOutput struct {
	Abstract         string `json:"abstract" yaml:"abstract"`
	ThoughtSignature []byte `json:"thought_signature,omitempty" yaml:"thought_signature,omitempty"`
}

// ReadAbstractFile reads an abstract from the specified file path.
// It attempts to parse it as YAML or JSON first, falling back to plain text if parsing fails.
// It returns the abstract content, a boolean indicating if it was successfully parsed (YAML/JSON), and an error.
func ReadAbstractFile(abstractFilePath string) (abstractContent string, parsed bool, err error) {
	abstractContentBytes, err := os.ReadFile(abstractFilePath)
	if err != nil {
		return "", false, fmt.Errorf("failed to read abstract file '%s': %w", abstractFilePath, err)
	}

	abstractContent = string(abstractContentBytes) // Default to raw content
	parsed = false

	if strings.HasSuffix(strings.ToLower(abstractFilePath), ".yaml") || strings.HasSuffix(strings.ToLower(abstractFilePath), ".yml") {
		var abstractData AbstractOutput
		if err := yaml.Unmarshal(abstractContentBytes, &abstractData); err != nil {
			log.Printf("Warning: Failed to parse abstract file '%s' as YAML: %v. Attempting to treat as plain text.", abstractFilePath, err)
			// Continue, abstractContent remains raw content
		} else {
			abstractContent = abstractData.Abstract
			log.Printf("Successfully parsed abstract content from YAML file.")
			parsed = true
		}
	} else if strings.HasSuffix(strings.ToLower(abstractFilePath), ".json") {
		var abstractData AbstractOutput
		if err := json.Unmarshal(abstractContentBytes, &abstractData); err != nil {
			log.Printf("Warning: Failed to parse abstract file '%s' as JSON: %v. Attempting to treat as plain text.", abstractFilePath, err)
			// Continue, abstractContent remains raw content
		} else {
			abstractContent = abstractData.Abstract
			log.Printf("Successfully parsed abstract content from JSON file.")
			parsed = true
		}
	}

	if !parsed {
		log.Printf("Abstract file '%s' is not a recognized YAML or JSON format, or parsing failed. Treating content as plain text abstract.", abstractFilePath)
	}

	return abstractContent, parsed, nil
}

// WriteAbstractFile writes the abstract content and thought signature to the specified file path in YAML format.
func WriteAbstractFile(outputPath string, abstract string, thoughtSignature []byte) error {
	outputData := AbstractOutput{
		Abstract:         abstract,
		ThoughtSignature: thoughtSignature,
	}
	yamlBytes, err := yaml.Marshal(outputData)
	if err != nil {
		return fmt.Errorf("error marshaling abstract output to YAML: %w", err)
	}

	err = os.WriteFile(outputPath, yamlBytes, 0644)
	if err != nil {
		return fmt.Errorf("error saving abstract to file '%s': %w", outputPath, err)
	}
	return nil
}