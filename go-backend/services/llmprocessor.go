package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"log"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// LLMJobProcessor implements JobProcessor interface for LLM operations
type LLMJobProcessor struct {
	db     *sql.DB
	logger *log.Logger
}

// NewLLMJobProcessor creates a new LLM job processor
func NewLLMJobProcessor(db *sql.DB) *LLMJobProcessor {
	return &LLMJobProcessor{
		db:     db,
		logger: log.Default(),
	}
}

// ProcessJob processes a job based on its type and returns the result
func (p *LLMJobProcessor) ProcessJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	p.logger.Printf("[Processor] Processing job %d (type: %s, user: %d)", job.ID, job.JobType, job.UserID)

	switch job.JobType {
	case models.JobTypeEmbedding:
		return p.processEmbeddingJob(ctx, job)
	case models.JobTypeEntityExtraction:
		return p.processEntityExtractionJob(ctx, job)
	case models.JobTypeMemory:
		return p.processMemoryJob(ctx, job)
	case models.JobTypeSummarization:
		return p.processSummarizationJob(ctx, job)
	case models.JobTypeChat:
		return p.processChatJob(ctx, job)
	case models.JobTypeEmail:
		return p.processEmailJob(ctx, job)
	default:
		return nil, fmt.Errorf("unknown job type: %s", job.JobType)
	}
}

// processEmbeddingJob generates embeddings for a card
func (p *LLMJobProcessor) processEmbeddingJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract card_pk from payload
	cardPK, ok := job.Payload["card_pk"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid card_pk in payload")
	}

	// Get card content
	var title, content string
	err := p.db.QueryRowContext(ctx,
		"SELECT title, content FROM cards WHERE id = $1 AND user_id = $2",
		int(cardPK), job.UserID).Scan(&title, &content)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("card not found")
		}
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	// Create LLM client
	client := NewDefaultClient(p.db, job.UserID)
	client.RequestType = "embedding"

	// Generate embedding using OpenAI
	// Note: This uses the embedding endpoint which returns vector data
	textForEmbedding := title + "\n\n" + content

	// Create embedding request
 embeddingRequest := openai.EmbeddingRequest{
		Input: []string{textForEmbedding},
		Model: openai.AdaEmbeddingV2,
	}

	// Execute embedding request
	resp, err := client.Client.CreateEmbeddings(ctx, embeddingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	// Update card with embedding
	embedding := resp.Data[0].Embedding
	_, err = p.db.ExecContext(ctx,
		"UPDATE cards SET embedding = $1, updated_at = NOW() WHERE id = $2",
		embeddingToJSONB(embedding), int(cardPK))
	if err != nil {
		return nil, fmt.Errorf("failed to update card with embedding: %w", err)
	}

	return map[string]interface{}{
		"card_pk":   int(cardPK),
		"embedding_dim": len(embedding),
		"status":    "completed",
	}, nil
}

// processEntityExtractionJob extracts entities from a card
func (p *LLMJobProcessor) processEntityExtractionJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract card_pk from payload
	cardPK, ok := job.Payload["card_pk"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid card_pk in payload")
	}

	// Get card content
	var title, content string
	err := p.db.QueryRowContext(ctx,
		"SELECT title, content FROM cards WHERE id = $1 AND user_id = $2",
		int(cardPK), job.UserID).Scan(&title, &content)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("card not found")
		}
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	// Create LLM client
	client := NewDefaultClient(p.db, job.UserID)
	client.RequestType = "entity_extraction"

	// Extract entities using the service function
	entities, err := FindEntities(client, title, content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities: %w", err)
	}

	// Save entities to database
	var savedEntityIDs []int
	for _, entity := range entities {
		entityID, err := p.saveEntity(ctx, job.UserID, entity, int(cardPK))
		if err != nil {
			p.logger.Printf("[Processor] Failed to save entity %s: %v", entity.Name, err)
			continue
		}
		savedEntityIDs = append(savedEntityIDs, entityID)
	}

	return map[string]interface{}{
		"card_pk":        int(cardPK),
		"entities_found": len(entities),
		"entities_saved": len(savedEntityIDs),
		"entity_ids":     savedEntityIDs,
		"status":         "completed",
	}, nil
}

// processMemoryJob generates or updates user memory
func (p *LLMJobProcessor) processMemoryJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract memory_type from payload
	memoryType, ok := job.Payload["memory_type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing memory_type in payload")
	}

	client := NewDefaultClient(p.db, job.UserID)

	switch memoryType {
	case "card":
		// Extract card_content from payload
		cardContent, ok := job.Payload["card_content"].(string)
		if !ok {
			return nil, fmt.Errorf("missing card_content in payload")
		}

		_, err := GenerateUserMemory(ctx, p.db, client, uint(job.UserID), cardContent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate memory: %w", err)
		}

		// Update memory_has_changed flag
		_, err = p.db.ExecContext(ctx,
			"UPDATE users SET memory_has_changed = true WHERE id = $1",
			job.UserID)
		if err != nil {
			p.logger.Printf("[Processor] Failed to update memory_has_changed flag: %v", err)
		}

	case "chat":
		// Extract user_message and assistant_message from payload
		userMessage, ok1 := job.Payload["user_message"].(string)
		assistantMessage, ok2 := job.Payload["assistant_message"].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("missing user_message or assistant_message in payload")
		}

		_, err := GenerateUserChatMemory(ctx, p.db, client, uint(job.UserID), userMessage, assistantMessage)
		if err != nil {
			return nil, fmt.Errorf("failed to generate chat memory: %w", err)
		}

		// Update memory_has_changed flag
		_, err = p.db.ExecContext(ctx,
			"UPDATE users SET memory_has_changed = true WHERE id = $1",
			job.UserID)
		if err != nil {
			p.logger.Printf("[Processor] Failed to update memory_has_changed flag: %v", err)
		}

	default:
		return nil, fmt.Errorf("unknown memory type: %s", memoryType)
	}

	return map[string]interface{}{
		"user_id":    job.UserID,
		"memory_type": memoryType,
		"status":     "completed",
	}, nil
}

// processSummarizationJob processes a summarization job
func (p *LLMJobProcessor) processSummarizationJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract summarization_id from payload
	summarizationID, ok := job.Payload["summarization_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid summarization_id in payload")
	}

	// Get summarization details
	var inputText string
	var cardPK sql.NullInt64
	err := p.db.QueryRowContext(ctx,
		"SELECT input_text, card_pk FROM summarizations WHERE id = $1 AND user_id = $2",
		int(summarizationID), job.UserID).Scan(&inputText, &cardPK)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("summarization not found")
		}
		return nil, fmt.Errorf("failed to get summarization: %w", err)
	}

	// Create LLM client
	client := NewDefaultClient(p.db, job.UserID)
	client.RequestType = "summarization"

	// Extract theses and arguments
	analyses, facts, usage, err := ExtractThesesAndArguments(client, inputText)
	if err != nil {
		return nil, fmt.Errorf("failed to extract theses: %w", err)
	}

	// Generate summary (facts are passed but may be empty)
	result, _, usage, err := AnalyzeAndSummarizeText(client, analyses, facts, usage)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize: %w", err)
	}

	// Update summarization record
	_, err = p.db.ExecContext(ctx,
		`UPDATE summarizations
		 SET status = 'complete', result = $1, prompt_tokens = $2, completion_tokens = $3,
		     total_tokens = $4, cost = $5, model = $6, updated_at = NOW()
		 WHERE id = $7`,
		result, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
		usage.TotalCost, "deprecated", int(summarizationID))
	if err != nil {
		return nil, fmt.Errorf("failed to update summarization: %w", err)
	}

	return map[string]interface{}{
		"summarization_id": int(summarizationID),
		"result":           result,
		"prompt_tokens":    usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"total_tokens":     usage.TotalTokens,
		"cost":             usage.TotalCost,
		"status":           "completed",
	}, nil
}

// processChatJob processes a chat message job
func (p *LLMJobProcessor) processChatJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract conversation_id and message from payload
	conversationID, ok1 := job.Payload["conversation_id"].(float64)
	message, ok2 := job.Payload["message"].(string)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("missing conversation_id or message in payload")
	}

	// This is a placeholder - actual chat processing would involve:
	// 1. Loading conversation history
	// 2. Getting user memory
	// 3. Calling the LLM with tools
	// 4. Storing the response

	// For now, return a basic response indicating the job type
	// The full chat implementation will be added when migrating chat handlers

	return map[string]interface{}{
		"conversation_id": int(conversationID),
		"message_length":  len(message),
		"status":          "completed",
		"note":            "Chat processing to be implemented in chat migration phase",
	}, nil
}

// processEmailJob processes an email job
func (p *LLMJobProcessor) processEmailJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract email details from payload
	toEmail, ok1 := job.Payload["to"].(string)
	subject, ok2 := job.Payload["subject"].(string)
	body, ok3 := job.Payload["body"].(string)
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("missing to, subject, or body in payload")
	}

	// This would integrate with the mail service
	// For now, we'll log the email details
	p.logger.Printf("[Processor] Email job: to=%s, subject=%s", toEmail, subject)

	// Store sent email record
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO sent_emails (user_id, to_email, subject, body, sent_at, created_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		job.UserID, toEmail, subject, body)
	if err != nil {
		return nil, fmt.Errorf("failed to record sent email: %w", err)
	}

	return map[string]interface{}{
		"user_id": job.UserID,
		"to":      toEmail,
		"subject": subject,
		"status":  "sent",
	}, nil
}

// Helper: saveEntity saves an entity to the database
func (p *LLMJobProcessor) saveEntity(ctx context.Context, userID int, entity models.Entity, cardPK int) (int, error) {
	var entityID int
	err := p.db.QueryRowContext(ctx,
		`INSERT INTO entities (user_id, name, description, type, card_pk, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		 ON CONFLICT (user_id, name) DO UPDATE
		 SET description = COALESCE(EXCLUDED.description, entities.description),
		     type = COALESCE(EXCLUDED.type, entities.type),
		     updated_at = NOW()
		 RETURNING id`,
		userID, entity.Name, entity.Description, entity.Type, cardPK).Scan(&entityID)
	if err != nil {
		return 0, err
	}

	// Link entity to card
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO entity_card_junction (entity_id, card_pk, created_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT DO NOTHING`,
		entityID, cardPK)
	if err != nil {
		p.logger.Printf("[Processor] Failed to link entity %d to card %d: %v", entityID, cardPK, err)
	}

	return entityID, nil
}

// Helper: embeddingToJSONB converts embedding slice to JSONB format
func embeddingToJSONB(embedding []float32) []byte {
	// Convert to JSON array string
	jsonStr := "["
	for i, v := range embedding {
		if i > 0 {
			jsonStr += ","
		}
		jsonStr += fmt.Sprintf("%f", v)
	}
	jsonStr += "]"
	return []byte(jsonStr)
}

// Helper: GenerateUserMemory for card-based memory generation
func GenerateUserMemory(ctx context.Context, db *sql.DB, client *models.LLMClient, userID uint, cardContent string) (string, error) {
	userMemory, err := GetUserMemory(db, int(userID))
	if err != nil {
		return "", err
	}

	promptTemplate, err := prompts.GetCardMemoryAssistantPrompt()
	if err != nil {
		// Fallback prompt
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

	response, err := ExecuteLLMRequest(ctx, client, messages)
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

	return content, nil
}

// Helper: GenerateUserChatMemory for chat-based memory generation
func GenerateUserChatMemory(ctx context.Context, db *sql.DB, client *models.LLMClient, userID uint, userMessage, assistantMessage string) (string, error) {
	userMemory, err := GetUserMemory(db, int(userID))
	if err != nil {
		return "", err
	}

	promptTemplate, err := prompts.GetChatMemoryAssistantPrompt()
	if err != nil {
		// Fallback prompt
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

	response, err := ExecuteLLMRequest(ctx, client, messages)
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

	return content, nil
}
