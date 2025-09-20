package handlers

import (
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

	"github.com/sashabaranov/go-openai"
)

func (s *Handler) GenerateMemory(userID uint, cardContent string) {

	if s.Server.Testing {
		return
	}

	go func() {
		client := services.NewDefaultClient(s.DB, int(userID))
		client.RequestType = "memory"
		_, err := GenerateUserMemory(s.DB, client, userID, cardContent)
		if err != nil {
			log.Printf("error generating user memory: %v", err)
			return
		}
		_, err = s.DB.Exec("UPDATE users SET memory_has_changed = true WHERE id = $1", userID)
		if err != nil {
			log.Printf("failed to update memory_has_changed flag for user %d: %v", userID, err)
			return
		}
	}()
}

func (s *Handler) GenerateChatMemory(userID uint, userMessage, assistantMessage string) {

	if s.Server.Testing {
		return
	}

	go func() {
		client := services.NewDefaultClient(s.DB, int(userID))
		client.RequestType = "chat_memory"
		_, err := GenerateUserChatMemory(s.DB, client, userID, userMessage, assistantMessage)
		if err != nil {
			log.Printf("error generating user chat memory: %v", err)
			return
		}
		_, err = s.DB.Exec("UPDATE users SET memory_has_changed = true WHERE id = $1", userID)
		if err != nil {
			log.Printf("failed to update memory_has_changed flag for user %d: %v", userID, err)
			return
		}
	}()
}

func GetUserMemory(db *sql.DB, userID int) (string, error) {
	var memory string
	err := db.QueryRow("SELECT memory FROM user_memories WHERE user_id = $1", userID).Scan(&memory)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return memory, nil
}

func UpdateUserMemory(db *sql.DB, userID uint, memory string) error {
	_, err := db.Exec(`
		INSERT INTO user_memories (user_id, memory, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET memory = $2, updated_at = NOW()
	`, userID, memory)
	return err
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

func GenerateUserMemory(db *sql.DB, client *models.LLMClient, userID uint, cardContent string) (string, error) {
	userMemory, err := GetUserMemory(db, int(userID))

	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(

		`

		You are an AI Memory Scribe for a zettelkasten application. Your task is to analyze a new piece of user text and add granular, raw observations to the "Recent Observations" section of the user's memory file.
		 
		In particular, you should be interested only in observations about the *user*, not about the text itself. Think about what the text says about the user, and what it means that the user has added this text to their zettelkasten. You should only be interested with meta observations about the user, not the actual details of what has been recorded (that is what the zettelkasten itself is for!).

You must follow this process precisely:
1.  **Preserve the Long-Term Memory:** The entire section under the '## Long-Term Memory' heading must be copied into the output exactly as it is, without any changes. If it does not exist, create it
2.  **Analyze the New Text:** Read the "New User Text" provided below and extract atomic, specific observations about the user's interests, activities, or personality.
3.  **Append New Observations:** Add your new findings as bullet points to the end of the existing content under the '## Recent Observations' heading.
4.  **Output the Full Document:** Your final output must be the complete, updated memory block, including both the untouched Long-Term Memory and the newly appended-to Recent Observations.

**CRITICAL RULES:**
*   **DO NOT MODIFY THE LONG-TERM MEMORY SECTION.**
*   Do not synthesize or abstract. Capture raw data points.
*   Your output must be the ENTIRE updated text block in valid Markdown.
*   Keep in mind that the texts are from zettelkasten cards, there is a chance they are quotes and are not actual facts about the user.

**Existing Memory Block:**
%s

**New Text:**
%s

**Updated User Memory (present the updated memory in a similar structured format, e.g., using bullet points or sections for "Core Interests," "Personality Insights," etc.):**`,
		userMemory,
		cardContent,
	)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	response, err := services.ExecuteLLMRequest(client, messages)
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

	response, err := services.ExecuteLLMRequest(client, messages)
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

func GenerateUserChatMemory(db *sql.DB, client *models.LLMClient, userID uint, userMessage, assistantMessage string) (string, error) {
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

	response, err := services.ExecuteLLMRequest(client, messages)
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
