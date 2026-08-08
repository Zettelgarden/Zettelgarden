package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
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
					 VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
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
					"UPDATE entities SET description=$1, type=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3",
					entity.Description, entity.Type, entityID)
				if err != nil {
					p.logger.Printf("[Processor] Failed to update entity '%s': %v", entity.Name, err)
					continue
				}
			}

			// Link entity to fact
			_, err = p.db.ExecContext(ctx,
				`INSERT INTO entity_fact_junction (user_id, entity_id, fact_id, created_at, updated_at)
				 VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				 ON CONFLICT (entity_id, fact_id) DO UPDATE SET updated_at = CURRENT_TIMESTAMP`,
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
				 ON CONFLICT (entity_id, card_pk) DO UPDATE SET updated_at = CURRENT_TIMESTAMP`,
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

// processSummarizationJob runs the map-reduce summarizer for a summarization
// row. It loads the prepared input_text from the row, summarizes each chunk
// (map), then reduces the chunk-summaries into the final markdown, and writes
// the result back to the summarizations row.
//
// All LLM work happens here, behind the job queue, so it is retried,
// cancelable, and never blocks the HTTP request.
func (p *LLMJobProcessor) processSummarizationJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
	// Extract summarization_id from payload
	summarizationID, ok := job.Payload["summarization_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid summarization_id in payload")
	}

	// Load the prepared input_text from the summarizations row
	var inputText string
	err := p.db.QueryRowContext(ctx,
		"SELECT input_text FROM summarizations WHERE id = $1 AND user_id = $2",
		int(summarizationID), job.UserID).Scan(&inputText)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("summarization not found")
		}
		return nil, fmt.Errorf("failed to get summarization: %w", err)
	}

	// Create LLM client
	client := NewDefaultClient(p.db, job.UserID, false)
	client.RequestType = "summarization"

	// MAP: summarize each chunk independently
	chunkSummaries, usage, err := SummarizeChunks(client, inputText)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize chunks: %w", err)
	}

	// REDUCE: combine chunk-summaries into the final markdown
	result, usage, err := SummarizeReduce(client, chunkSummaries, usage)
	if err != nil {
		return nil, fmt.Errorf("failed to reduce summary: %w", err)
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

// updateSummarizationResult updates the summarization record with the result
func (p *LLMJobProcessor) updateSummarizationResult(ctx context.Context, summarizationID int, result string, usage models.Usage, model string) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE summarizations
		 SET status = 'complete', result = $1, prompt_tokens = $2, completion_tokens = $3,
		     total_tokens = $4, cost = $5, model = $6, updated_at = CURRENT_TIMESTAMP
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
	// 2. Calling the LLM with tools
	// 3. Storing the response

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
		 VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT (user_id, name) DO UPDATE
		 SET description = COALESCE(EXCLUDED.description, entities.description),
		     type = COALESCE(EXCLUDED.type, entities.type),
		     updated_at = CURRENT_TIMESTAMP
		 RETURNING id`,
		userID, entity.Name, entity.Description, entity.Type, cardPK).Scan(&entityID)
	if err != nil {
		return 0, err
	}

	// Link entity to card
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO entity_card_junction (entity_id, card_pk, created_at)
		 VALUES ($1, $2, CURRENT_TIMESTAMP)
		 ON CONFLICT DO NOTHING`,
		entityID, cardPK)
	if err != nil {
		p.logger.Printf("[Processor] Failed to link entity %d to card %d: %v", entityID, cardPK, err)
	}

	return entityID, nil
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
