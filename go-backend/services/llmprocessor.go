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

// LLMJobProcessor executes LLM operations for each job type. It is invoked
// directly by services.JobRunner (no longer via a JobProcessor interface).
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
	case models.JobTypeEntityExtraction:
		return p.processEntityExtractionJob(ctx, job)
	case models.JobTypeFactEntityExtraction:
		return p.processFactEntityExtractionJob(ctx, job)
	case models.JobTypeMemory:
		return p.processMemoryJob(ctx, job)
	case models.JobTypeSummarization:
		return p.processSummarizationJob(ctx, job)
	case models.JobTypeChat:
		return p.processChatJob(ctx, job)
	case models.JobTypeFileTextExtraction:
		return p.processFileTextExtractionJob(ctx, job)
	default:
		return nil, fmt.Errorf("unknown job type: %s", job.JobType)
	}
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
	client := NewDefaultClient(p.db, job.UserID, false)
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

// processFactEntityExtractionJob extracts entities from facts and links them
func (p *LLMJobProcessor) processFactEntityExtractionJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract card_pk from payload
	cardPKFloat, ok := job.Payload["card_pk"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid card_pk in payload")
	}
	cardPK := int(cardPKFloat)

	// Extract facts from payload
	factsData, ok := job.Payload["facts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid facts in payload")
	}

	// Parse facts from payload
	var facts []models.Fact
	for _, f := range factsData {
		factMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		factID, _ := factMap["id"].(float64)
		factText, _ := factMap["fact"].(string)
		facts = append(facts, models.Fact{
			ID:   int(factID),
			Fact: factText,
		})
	}

	// Create LLM client
	client := NewDefaultClient(p.db, job.UserID, false)
	client.RequestType = "analysis"

	// Extract entities using batch processing
	factEntities, err := FindEntitiesBatch(client, facts)
	if err != nil {
		p.logger.Printf("[Processor] FindEntitiesBatch error: %v", err)
		return nil, fmt.Errorf("failed to extract entities from facts: %w", err)
	}

	// Save entities and create links
	var entitiesSaved int
	var entityFactLinks int
	var entityCardLinks int

	for i, entities := range factEntities {
		if i >= len(facts) {
			break
		}
		fact := facts[i]

		for _, entity := range entities {
			// Validate entity before processing
			if entity.Name == "" {
				continue
			}

			// Check if entity exists
			var entityID int
			err := p.db.QueryRowContext(ctx,
				"SELECT id FROM entities WHERE user_id = $1 AND name = $2",
				job.UserID, entity.Name).Scan(&entityID)

			if err == sql.ErrNoRows {
				// Entity doesn't exist, insert it
				err = p.db.QueryRowContext(ctx,
					`INSERT INTO entities (user_id, name, description, type, card_pk, created_at, updated_at)
					 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
					 RETURNING id`,
					job.UserID, entity.Name, entity.Description, entity.Type, entity.CardPK).Scan(&entityID)
				if err != nil {
					p.logger.Printf("[Processor] Failed to insert entity '%s': %v", entity.Name, err)
					continue
				}
				entitiesSaved++
			} else if err != nil {
				p.logger.Printf("[Processor] Failed to query entity '%s': %v", entity.Name, err)
				continue
			} else {
				// Entity exists, update it
				_, err = p.db.ExecContext(ctx,
					"UPDATE entities SET description=$1, type=$2, updated_at=NOW() WHERE id=$3",
					entity.Description, entity.Type, entityID)
				if err != nil {
					p.logger.Printf("[Processor] Failed to update entity '%s': %v", entity.Name, err)
					continue
				}
			}

			// Link entity to fact
			_, err = p.db.ExecContext(ctx,
				`INSERT INTO entity_fact_junction (user_id, entity_id, fact_id, created_at, updated_at)
				 VALUES ($1, $2, $3, NOW(), NOW())
				 ON CONFLICT (entity_id, fact_id) DO UPDATE SET updated_at = NOW()`,
				job.UserID, entityID, fact.ID)
			if err != nil {
				p.logger.Printf("[Processor] Failed to link entity to fact: %v", err)
			} else {
				entityFactLinks++
			}

			// Link entity to card
			_, err = p.db.ExecContext(ctx,
				`INSERT INTO entity_card_junction (user_id, entity_id, card_pk, chunk_id)
				 VALUES ($1, $2, $3, $4)
				 ON CONFLICT (entity_id, card_pk) DO UPDATE SET updated_at = NOW()`,
				job.UserID, entityID, cardPK, 0)
			if err != nil {
				p.logger.Printf("[Processor] Failed to link entity to card: %v", err)
			} else {
				entityCardLinks++
			}
		}
	}

	return map[string]interface{}{
		"card_pk":           cardPK,
		"entities_saved":    entitiesSaved,
		"entity_fact_links": entityFactLinks,
		"entity_card_links": entityCardLinks,
		"status":            "completed",
	}, nil
}

// processMemoryJob generates or updates user memory
func (p *LLMJobProcessor) processMemoryJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract memory_type from payload
	memoryType, ok := job.Payload["memory_type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing memory_type in payload")
	}

	client := NewDefaultClient(p.db, job.UserID, false)

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

	default:
		return nil, fmt.Errorf("unknown memory type: %s", memoryType)
	}

	return map[string]interface{}{
		"user_id":     job.UserID,
		"memory_type": memoryType,
		"status":      "completed",
	}, nil
}

// processSummarizationJob processes a summarization job
func (p *LLMJobProcessor) processSummarizationJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract summarization_id from payload
	summarizationID, ok := job.Payload["summarization_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid summarization_id in payload")
	}

	// Check if pre-extracted analyses/facts are in payload (new path)
	if analysesData, hasAnalyses := job.Payload["analyses"]; hasAnalyses {
		// New path: data already extracted, just call AnalyzeAndSummarizeText
		analyses, facts, usage, err := p.parsePayloadAnalyses(analysesData, job.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse payload analyses: %w", err)
		}

		client := NewDefaultClient(p.db, job.UserID, false)
		client.RequestType = "summarization"

		result, _, usage, err := AnalyzeAndSummarizeText(client, analyses, facts, usage)
		if err != nil {
			return nil, fmt.Errorf("failed to summarize: %w", err)
		}

		// Update summarization record
		err = p.updateSummarizationResult(ctx, int(summarizationID), result, usage, client.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to update summarization: %w", err)
		}

		return map[string]interface{}{
			"summarization_id": int(summarizationID),
			"result":           result,
			"status":           "completed",
		}, nil
	}

	// Legacy path: fetch input_text and extract (keep for backward compatibility)
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
	client := NewDefaultClient(p.db, job.UserID, false)
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
	err = p.updateSummarizationResult(ctx, int(summarizationID), result, usage, client.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to update summarization: %w", err)
	}

	return map[string]interface{}{
		"summarization_id": int(summarizationID),
		"result":           result,
		"status":           "completed",
	}, nil
}

// parsePayloadAnalyses parses analyses and facts from the job payload
func (p *LLMJobProcessor) parsePayloadAnalyses(analysesData interface{}, payload map[string]interface{}) ([]models.SectionAnalysis, []string, models.Usage, error) {
	// Parse analyses
	analysesSlice, ok := analysesData.([]interface{})
	if !ok {
		return nil, nil, models.Usage{}, fmt.Errorf("invalid analyses format in payload")
	}

	analyses := make([]models.SectionAnalysis, len(analysesSlice))
	for i, a := range analysesSlice {
		sectionMap, ok := a.(map[string]interface{})
		if !ok {
			return nil, nil, models.Usage{}, fmt.Errorf("invalid section format at index %d", i)
		}

		section, _ := sectionMap["section"].(string)
		analyses[i].Section = section

		// Parse theses
		thesesSlice, ok := sectionMap["theses"].([]interface{})
		if !ok {
			continue
		}

		analyses[i].Theses = make([]models.ThesisEntry, len(thesesSlice))
		for j, t := range thesesSlice {
			thesisMap, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			thesis, _ := thesisMap["thesis"].(string)
			analyses[i].Theses[j].Thesis = thesis

			// Parse arguments
			argsSlice, ok := thesisMap["arguments"].([]interface{})
			if ok {
				analyses[i].Theses[j].Arguments = make([]models.Argument, len(argsSlice))
				for k, arg := range argsSlice {
					argMap, ok := arg.(map[string]interface{})
					if !ok {
						continue
					}
					argument, _ := argMap["argument"].(string)
					importance, _ := argMap["importance"].(float64)
					analyses[i].Theses[j].Arguments[k] = models.Argument{
						Argument:   argument,
						Importance: int(importance),
					}
				}
			}
		}
	}

	// Parse facts
	var facts []string
	if factsData, ok := payload["facts"].([]interface{}); ok {
		facts = make([]string, len(factsData))
		for i, f := range factsData {
			facts[i], _ = f.(string)
		}
	}

	// Parse usage
	var usage models.Usage
	if usageData, ok := payload["usage"].(map[string]interface{}); ok {
		if pt, ok := usageData["prompt_tokens"].(float64); ok {
			usage.PromptTokens = int(pt)
		}
		if ct, ok := usageData["completion_tokens"].(float64); ok {
			usage.CompletionTokens = int(ct)
		}
		if tt, ok := usageData["total_tokens"].(float64); ok {
			usage.TotalTokens = int(tt)
		}
		if tc, ok := usageData["total_cost"].(float64); ok {
			usage.TotalCost = tc
		}
	}

	return analyses, facts, usage, nil
}

// updateSummarizationResult updates the summarization record with the result
func (p *LLMJobProcessor) updateSummarizationResult(ctx context.Context, summarizationID int, result string, usage models.Usage, model string) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE summarizations
		 SET status = 'complete', result = $1, prompt_tokens = $2, completion_tokens = $3,
		     total_tokens = $4, cost = $5, model = $6, updated_at = NOW()
		 WHERE id = $7`,
		result, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
		usage.TotalCost, model, summarizationID)
	return err
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

	err = UpdateUserMemory(db, int(userID), content)
	if err != nil {
		return "", err
	}

	return content, nil
}

// processFileTextExtractionJob extracts text from an uploaded file
// TODO: This requires S3 client integration to download files from storage
// For now, this is a placeholder that will be enhanced when integrated into main.go
func (p *LLMJobProcessor) processFileTextExtractionJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract file_id from payload
	fileIDFloat, ok := job.Payload["file_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid file_id in payload")
	}
	fileID := int(fileIDFloat)

	p.logger.Printf("[Processor] Processing file text extraction for file %d (user: %d)", fileID, job.UserID)

	// Get file metadata
	var contentType string
	var s3Key string
	err := p.db.QueryRowContext(ctx,
		"SELECT type, path FROM files WHERE id = $1 AND user_id = $2",
		fileID, job.UserID).Scan(&contentType, &s3Key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found")
		}
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	// TODO: Download file from S3 using s3Key
	// For now, we'll mark this as needing manual implementation
	// When integrated into main.go, we should pass S3 client to the processor

	// Placeholder: Extract text would happen here after S3 download
	// extractedText, err := ExtractText(contentType, fileReader)

	// For now, just mark as processed
	p.logger.Printf("[Processor] File text extraction job for file %d - S3 integration pending", fileID)

	return map[string]interface{}{
		"file_id":      fileID,
		"status":       "pending_s3_integration",
		"content_type": contentType,
		"s3_key":       s3Key,
		"note":         "S3 download integration required in main.go",
	}, nil
}
