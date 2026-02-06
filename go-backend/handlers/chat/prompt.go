// Package chat provides system prompt building functionality for the chat service.
package chat

import (
	"context"
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"time"
)

const (
	// PromptVersionCurrent is the current version of the system prompt format.
	PromptVersionCurrent = "2.0"

	// PromptVersionLegacy is the legacy system prompt format.
	PromptVersionLegacy = "1.0"
)

// BuildSystemPrompt constructs the system prompt with all necessary context.
func (s *Service) BuildSystemPrompt(ctx context.Context, userID int, conversation *models.ChatConversation) (string, error) {
	return s.buildSystemPrompt(ctx, userID, conversation)
}

// buildSystemPrompt constructs the system prompt with all necessary context.
func (s *Service) buildSystemPrompt(ctx context.Context, userID int, conversation *models.ChatConversation) (string, error) {
	systemPrompt, err := prompts.GetZettelgardenAssistantPrompt()
	if err != nil {
		s.logError("Error loading system prompt: %v, using fallback", err)
		systemPrompt = "You are the Zettelgarden Assistant, a daily productivity companion for managing a Zettelkasten knowledge base."
	}

	// Add prompt version indicator
	systemPrompt += fmt.Sprintf("\n\n[Prompt Version: %s]", PromptVersionCurrent)

	// Add primary card context if this conversation is about a specific card
	if conversation != nil && conversation.PrimaryCardID != nil {
		// TODO: Fetch card details
		systemPrompt += fmt.Sprintf("\n\n## Primary Focus Card\n\n")
		systemPrompt += fmt.Sprintf("This conversation is about card ID: %d.\n", *conversation.PrimaryCardID)
	}

	// Add user's chat instructions if they exist
	// TODO: Fetch user instructions

	// Add current date and time
	currentTime := time.Now()
	systemPrompt += "\n\n## Current Date and Time\n\n"
	systemPrompt += fmt.Sprintf("Today's date is %s (UTC: %s)",
		currentTime.Format("Monday, January 2, 2006"),
		currentTime.UTC().Format("2006-01-02 15:04:05 UTC"))

	return systemPrompt, nil
}

// GetPromptVersion returns the current prompt version identifier.
func GetPromptVersion() string {
	return PromptVersionCurrent
}

// IsPromptVersionSupported checks if a given prompt version is supported.
func IsPromptVersionSupported(version string) bool {
	switch version {
	case PromptVersionCurrent, PromptVersionLegacy:
		return true
	default:
		return false
	}
}
