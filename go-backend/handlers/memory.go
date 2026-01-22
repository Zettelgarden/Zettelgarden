package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// GenerateMemory generates user memory based on card content (fire-and-forget with timeout)
func (s *Handler) GenerateMemory(userID uint, cardContent string) {
	if s.Server.Testing {
		return
	}

	go func() {
		// Create a context with timeout to prevent indefinite goroutine execution
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		client := services.NewDefaultClient(s.DB, int(userID))
		client.RequestType = "memory"
		_, err := GenerateUserMemory(ctx, s.DB, client, userID, cardContent)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("user memory generation timed out for user %d", userID)
			} else {
				log.Printf("error generating user memory: %v", err)
			}
			return
		}
		_, err = s.DB.Exec("UPDATE users SET memory_has_changed = true WHERE id = $1", userID)
		if err != nil {
			log.Printf("failed to update memory_has_changed flag for user %d: %v", userID, err)
			return
		}
	}()
}

// GenerateChatMemory generates user memory based on chat (fire-and-forget with timeout)
func (s *Handler) GenerateChatMemory(userID uint, userMessage, assistantMessage string) {
	if s.Server.Testing {
		return
	}

	go func() {
		// Create a context with timeout to prevent indefinite goroutine execution
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		client := services.NewDefaultClient(s.DB, int(userID))
		client.RequestType = "chat_memory"
		_, err := GenerateUserChatMemory(ctx, s.DB, client, userID, userMessage, assistantMessage)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("chat memory generation timed out for user %d", userID)
			} else {
				log.Printf("error generating user chat memory: %v", err)
			}
			return
		}
		_, err = s.DB.Exec("UPDATE users SET memory_has_changed = true WHERE id = $1", userID)
		if err != nil {
			log.Printf("failed to update memory_has_changed flag for user %d: %v", userID, err)
			return
		}
	}()
}

// GetUserMemory is a wrapper for services.GetUserMemory for backward compatibility
func GetUserMemory(db *sql.DB, userID int) (string, error) {
	return services.GetUserMemory(db, userID)
}

// UpdateUserMemory is a wrapper for services.UpdateUserMemory for backward compatibility
func UpdateUserMemory(db *sql.DB, userID uint, memory string) error {
	return services.UpdateUserMemory(db, userID, memory)
}

func (s *Handler) GetUserMemoryRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	memory, err := GetUserMemory(s.DB, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"memory": memory})
}

func (s *Handler) UpdateUserMemoryRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON
	var requestData struct {
		Memory string `json:"memory"`
	}

	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Update memory in database
	err = UpdateUserMemory(s.DB, uint(userID), requestData.Memory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Memory updated successfully"})
}

// GenerateUserMemory generates user memory based on card content
func GenerateUserMemory(ctx context.Context, db *sql.DB, client *models.LLMClient, userID uint, cardContent string) (string, error) {
	userMemory, err := GetUserMemory(db, int(userID))
	if err != nil {
		return "", err
	}

	// Load the card memory prompt
	promptTemplate, err := prompts.GetCardMemoryAssistantPrompt()
	if err != nil {
		log.Printf("Error loading card memory prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		promptTemplate = `You are analyzing a card to update user memory.

**Existing Memory:**
%s

**New Card Content:**
%s

**Please update the memory with observations about the user based on this card:**`
	}

	prompt := fmt.Sprintf(promptTemplate, userMemory, cardContent)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	response, err := services.ExecuteLLMRequest(ctx, client, messages)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	content := response.Choices[0].Message.Content
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")

	err = UpdateUserMemory(db, uint(userID), content)
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

func CompressUserMemory(db *sql.DB, client *models.LLMClient, userID uint) (string, error) {
	userMemory, err := GetUserMemory(db, int(userID))
	if err != nil {
		log.Printf("error getting memory: %v", err)
		return "", err
	}
	// skip if the user hasn't changed a card
	if userMemory == "" {
		return "", err
	}

	prompt := fmt.Sprintf(

		`
		You are an expert AI knowledge architect and editor. Your purpose is to perform a daily consolidation of the user's memory file. Your primary goal is to maintain a **concise, high-signal Long-Term Memory** that fits efficiently within a limited context window.

Your task is to produce a new, superior, and more compact version of the entire memory block.

**Your Process:**

1.  **Analyze Both Sections:** Read and understand the existing '## Long-Term Memory' and all the raw data in '## Recent Observations'.
2.  **Synthesize and Integrate:** Integrate the significant insights and themes from the "Recent Observations" into the "Long-Term Memory."
    *   Strengthen existing points with new evidence *or* replace them with a more insightful abstraction.
    *   Add new domains or traits **only if a strong, new, and consolidated pattern has emerged.**
    *   Restructure and refine the LTM for maximum clarity and semantic density.

3.  **Prune and Compress the LTM:** This is a critical step. After integrating new insights, you must actively shorten the Long-Term Memory section itself.
    *   **Merge Redundancies:** Find two or more points that describe the same core trait and combine them into a single, more powerful statement.
    *   **Remove Obsolete Details:** If an abstraction (e.g., "Deep interest in operational reliability") now covers several older, specific points (e.g., "mentioned BGP," "talked about on-call"), remove the older, specific points.
    *   **Rephrase for Brevity:** Edit existing statements to be more direct and concise without losing their meaning.

4.  **Empty the Recent Observations:** After synthesis and compression, clear the "Recent Observations" section. It has served its purpose and must be empty for the next cycle.

5.  **Output the Full Document:** Your final output is the complete, updated memory block, containing the newly refactored Long-Term Memory and the empty Recent Observations section.

**Full Memory Block to be Refactored:**
%s
`,
		userMemory,
	)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	response, err := services.ExecuteLLMRequest(context.Background(), client, messages)
	if err != nil {
		log.Printf("error getting LLM response: %v", err)
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	err = UpdateUserMemory(db, uint(userID), response.Choices[0].Message.Content)
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

// GenerateUserChatMemory generates user memory based on chat conversation
func GenerateUserChatMemory(ctx context.Context, db *sql.DB, client *models.LLMClient, userID uint, userMessage, assistantMessage string) (string, error) {
	userMemory, err := GetUserMemory(db, int(userID))
	if err != nil {
		return "", err
	}

	// Load the chat memory prompt
	promptTemplate, err := prompts.GetChatMemoryAssistantPrompt()
	if err != nil {
		log.Printf("Error loading chat memory prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		promptTemplate = `You are analyzing a chat conversation to update user memory.

**Existing Memory:**
%s

**Chat Exchange:**
User: %s
Assistant: %s

**Please update the memory with observations about the user based on this conversation:**`
	}

	prompt := fmt.Sprintf(promptTemplate, userMemory, userMessage, assistantMessage)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	response, err := services.ExecuteLLMRequest(ctx, client, messages)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	content := response.Choices[0].Message.Content
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")

	err = UpdateUserMemory(db, uint(userID), content)
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}
