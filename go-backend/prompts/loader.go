package prompts

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// LoadPrompt loads a prompt from a markdown file in the prompts directory
func LoadPrompt(filename string) (string, error) {
	// Ensure filename has .md extension
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	// Get the path to the prompts directory
	promptsDir := filepath.Join(".", "prompts")
	filePath := filepath.Join(promptsDir, filename)

	// Read the file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
