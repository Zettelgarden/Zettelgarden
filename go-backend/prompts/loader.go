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

// GetResearchAssistantPrompt loads the research assistant system prompt
func GetResearchAssistantPrompt() (string, error) {
	return LoadPrompt("research_assistant.md")
}

// GetSubagentResearcherPrompt loads the subagent researcher system prompt
func GetSubagentResearcherPrompt() (string, error) {
	return LoadPrompt("subagent_researcher.md")
}

// GetTitleGeneratorPrompt loads the title generator system prompt
func GetTitleGeneratorPrompt() (string, error) {
	return LoadPrompt("title_generator.md")
}