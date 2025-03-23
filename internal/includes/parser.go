package includes

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadIncludesFile loads the includes file from the given path
func LoadIncludesFile(path string) (IncludesFile, error) {
	var includes IncludesFile

	data, err := os.ReadFile(path)
	if err != nil {
		return includes, err
	}

	// Parse the YAML content
	err = yaml.Unmarshal(data, &includes)
	if err != nil {
		return includes, fmt.Errorf("failed to parse includes file: %w", err)
	}

	return includes, nil
}

// IsModelIncluded checks if a model is included
func IsModelIncluded(includes IncludesFile, modelName string) bool {
	for _, model := range includes.Models {
		if model == modelName {
			return true
		}
	}
	return false
}

// IsQueryIncluded checks if a query is included
func IsQueryIncluded(includes IncludesFile, queryName string) bool {
	for _, query := range includes.Queries {
		if query == queryName {
			return true
		}
	}
	return false
}

// WriteIncludesFile writes the includes file to the given path
// If commentOut is true, all entries will be commented out
func WriteIncludesFile(path string, models []string, queries []string, commentOut bool) error {
	// Create the content
	var content strings.Builder

	content.WriteString("models:\n")
	for _, model := range models {
		if commentOut {
			content.WriteString("# - " + model + "\n")
		} else {
			content.WriteString("- " + model + "\n")
		}
	}

	content.WriteString("\nqueries:\n")
	for _, query := range queries {
		if commentOut {
			content.WriteString("# - " + query + "\n")
		} else {
			content.WriteString("- " + query + "\n")
		}
	}

	// Write the content to the file
	return os.WriteFile(path, []byte(content.String()), 0o644)
}
