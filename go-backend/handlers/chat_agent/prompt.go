package chat_agent

import (
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"log"
	"time"
)

// buildSystemPrompt constructs the complete system prompt including memory, instructions, and context
func (s *ChatService) buildSystemPrompt(userID int, conversation *models.ChatConversation, getCardFn func(int, string) (interface{}, error), getInstructionsFn func(int) (string, error)) (string, error) {
	systemPrompt, err := prompts.GetZettelgardenAssistantPrompt()
	if err != nil {
		log.Printf("Error loading system prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		systemPrompt = "You are the Zettelgarden Assistant, a daily productivity companion for managing a Zettelkasten knowledge base."
	}

	// Add primary card context if this conversation is about a specific card
	if conversation != nil && conversation.PrimaryCardID != nil {
		cardID := fmt.Sprintf("%d", *conversation.PrimaryCardID)
		card, cardErr := getCardFn(userID, cardID)
		if cardErr == nil {
			// Format card information - this will be simplified once we have proper card types
			systemPrompt += "## Primary Focus Card\n\n"
			systemPrompt += fmt.Sprintf("This conversation is primarily about a card (ID: %s).\n", cardID)
			systemPrompt += "Use the get_card_by_id tool to retrieve the full content when needed.\n"
			systemPrompt += "Reference this card's content to help the user explore and develop related ideas.\n"
			_ = card // Use card to avoid unused variable warning
		}
	}

	// Note: User memory is now available via the get_user_memory tool
	// This allows just-in-time retrieval instead of pre-loading into context

	// Add user's chat instructions if they exist
	if getInstructionsFn != nil {
		instructions, instrErr := getInstructionsFn(userID)
		if instrErr == nil && instructions != "" {
			systemPrompt += "\n\n## User Instructions\n\n"
			systemPrompt += instructions
		}
	}

	// Add current date and time
	currentTime := time.Now()
	systemPrompt += "\n\n## Current Date and Time\n\n"
	systemPrompt += fmt.Sprintf("Today's date is %s (UTC: %s)",
		currentTime.Format("Monday, January 2, 2006"),
		currentTime.UTC().Format("2006-01-02 15:04:05 UTC"))

	return systemPrompt, nil
}
