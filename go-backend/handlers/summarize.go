package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
)

// Pre-compiled regex patterns for removeReferences
var (
	referencePattern          = regexp.MustCompile(`\[[^\]]+\] - [^\n]*\n?`)
	doubleNewlinePattern      = regexp.MustCompile(`\n\n+`)
	trailingWhitespacePattern = regexp.MustCompile(`\s+$`)
)

// getUserIDFromContext safely extracts userID from request context
// Returns Unauthorized error if the user is not authenticated
func getUserIDFromContext(w http.ResponseWriter, r *http.Request) (int, bool) {
	userID, ok := r.Context().Value("current_user").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

// removeReferences removes card reference lines from text before summarization
// References are lines that start with [tag] - title and end with newline
func removeReferences(text string) string {
	// Pattern matches: [anything] - anything followed by newline
	// Also handles the case where reference is at the end without trailing newline
	result := referencePattern.ReplaceAllString(text, "")

	// Clean up any resulting double newlines to avoid empty lines
	result = doubleNewlinePattern.ReplaceAllString(result, "\n\n")

	// Trim trailing whitespace
	result = trailingWhitespacePattern.ReplaceAllString(result, "")

	return result
}

// prepareTextForAnalysis prepares text for LLM analysis by adding title and removing references
func prepareTextForAnalysis(title, body string) string {
	var text string
	if title != "" {
		if body != "" {
			text = fmt.Sprintf("# %s\n\n%s", title, body)
		} else {
			text = fmt.Sprintf("# %s", title)
		}
	} else {
		text = body
	}
	return removeReferences(text)
}

// SummarizeRequest defines the payload for creating a summarization job
type SummarizeRequest struct {
	Text  string `json:"text"`
	Title string `json:"title,omitempty"`
	Model string `json:"model,omitempty"`
}

// GetSummariesByCardRoute returns all summarizations for a given card_pk
func (h *Handler) GetSummariesByCardRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}
	cardIDStr := mux.Vars(r)["card_pk"]
	cardID, err := strconv.Atoi(cardIDStr)
	if err != nil {
		http.Error(w, "Invalid card_pk", http.StatusBadRequest)
		return
	}

	summaries, err := h.querySummarizations(userID, &cardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// ListSummarizationsRoute returns all summarization jobs (lightweight view) for the current user
func (h *Handler) ListSummarizationsRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}

	summaries, err := h.querySummarizations(userID, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// querySummarizations is a shared helper that queries summarizations for a user,
// optionally filtered by card_pk.
//
// When cardPK is provided, only summarizations explicitly linked to that card
// are returned (card_pk IS NOT NULL). Manual/standalone summarizations (created
// without a card, i.e., card_pk IS NULL) are excluded from card-specific queries.
// These manual summaries only appear in the list view (ListSummarizationsRoute).
func (h *Handler) querySummarizations(userID int, cardPK *int) ([]SummarizeJobResponse, error) {
	var rows *sql.Rows
	var err error

	if cardPK != nil {
		rows, err = h.DB.Query(`
			SELECT id, status, result
			FROM summarizations
			WHERE user_id = $1 AND card_pk = $2
			ORDER BY created_at DESC
		`, userID, *cardPK)
	} else {
		rows, err = h.DB.Query(`
			SELECT id, status, result
			FROM summarizations
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query summarizations: %w", err)
	}
	defer rows.Close()

	var summaries []SummarizeJobResponse
	for rows.Next() {
		var job SummarizeJobResponse
		var result sql.NullString
		if err := rows.Scan(&job.ID, &job.Status, &result); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		// Convert sql.NullString to string
		if result.Valid {
			job.Result = result.String
		} else {
			job.Result = ""
		}
		summaries = append(summaries, job)
	}

	return summaries, nil
}

// CreateSummarizationRoute creates a summarization job and enqueues it to the
// LLM job queue. No LLM work happens here: the row is inserted as 'pending'
// and the whole map-reduce runs behind the job queue. The route returns
// immediately with the (still-pending) status.
func (h *Handler) CreateSummarizationRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}

	// Check rate limit
	if !h.checkSummarizationRateLimit(userID) {
		log.Printf("[RATE_LIMIT] User %d exceeded summarization rate limit", userID)
		http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}

	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Store the prepared text so the job can summarize it without re-running
	// the (title + reference-stripping) preparation.
	processedText := prepareTextForAnalysis(req.Title, req.Text)

	var jobID int
	var status string
	err := h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, input_text, status, created_at, updated_at)
			VALUES ($1, $2, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			RETURNING id, status
		`, userID, processedText).Scan(&jobID, &status)
	if err != nil {
		log.Printf("error starting summarization %v", err)
		http.Error(w, "Failed to create summarization", http.StatusInternalServerError)
		return
	}

	// Enqueue the summarization job (non-blocking)
	if _, err := h.runSummarizationJobViaQueue(userID, nil, jobID); err != nil {
		log.Printf("err %v", err)
		http.Error(w, "Failed to create summarization job", http.StatusInternalServerError)
		return
	}

	// Return actual status from database (not hardcoded "pending")
	resp := SummarizeJobResponse{ID: jobID, Status: status}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type SummarizeJobResponse struct {
	ID               int     `json:"id"`
	Status           string  `json:"status"`
	Result           string  `json:"result,omitempty"`
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
	Cost             float64 `json:"cost,omitempty"`
	Model            string  `json:"model,omitempty"`
}

// ProcessEntitiesAndFacts enqueues a card's summarization job and kicks off
// entity extraction. Only the summary goes through the job queue; entity
// extraction (ExtractSaveCardEntities + LinkCardToEntityIfPossible) is
// independent of the summary and stays as-is in a recovered goroutine.
//
// (The name is historical: it no longer processes "facts" — those were removed
// in qsg — but the public field on EditCardParams and its callers still use it.)
func (h *Handler) ProcessEntitiesAndFacts(userID int, card models.Card) {
	// Skip during testing to avoid external LLM calls
	if h.Server.Testing {
		return
	}

	// Store the prepared card text so the job can summarize it.
	processedText := prepareTextForAnalysis(card.Title, card.Body)

	var jobID int
	err := h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			RETURNING id
		`, userID, card.ID, processedText).Scan(&jobID)
	if err != nil {
		log.Printf("error starting summarization %v", err)
		return
	}

	// Enqueue the summarization job (non-blocking; whole map-reduce runs here)
	if _, err := h.runSummarizationJobViaQueue(userID, &card.ID, jobID); err != nil {
		log.Printf("Failed to run summarization job: %v", err)
	}

	// Entity extraction is independent of the summary; keep it async.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] Recovered in ProcessEntitiesAndFacts goroutine: %v", r)
			}
		}()
		// Ensure LinkCardToEntityIfPossible is always called exactly once,
		// regardless of success or failure of entity extraction.
		defer h.LinkCardToEntityIfPossible(userID, card)

		if err := h.ExtractSaveCardEntities(userID, card); err != nil {
			log.Printf("Failed to extract/save card entities: %v", err)
		}
	}()
}

// ExtractSaveCardEntities extracts entities directly from a card's title and body,
// saves them to the database, and links them to the card.
// This is independent of fact extraction.
func (h *Handler) ExtractSaveCardEntities(userID int, card models.Card) error {
	// Skip during testing to avoid external LLM calls
	if h.Server.Testing {
		return nil
	}

	isTesting := h.Server != nil && h.Server.Testing
	client := services.NewDefaultClient(h.DB, userID, isTesting)
	client.RequestType = "analysis"

	// Extract entities from card title and body
	entities, err := services.FindEntities(client, card.Title, card.Body)
	if err != nil {
		return fmt.Errorf("failed to extract entities: %w", err)
	}

	log.Printf("[EntityExtraction] Extracted %d entities from card %d", len(entities), card.ID)

	// Save each entity and link to card
	for _, entity := range entities {
		// Validate entity name and description
		if err := validateEntityName(entity.Name); err != nil {
			log.Printf("[EntityExtraction] Invalid entity name '%s': %v", entity.Name, err)
			continue
		}
		if err := validateEntityDescription(entity.Description); err != nil {
			log.Printf("[EntityExtraction] Invalid entity description for '%s': %v", entity.Name, err)
			continue
		}

		var entityID int

		// Check if entity already exists
		err := h.DB.QueryRow(`
			SELECT id FROM entities WHERE user_id = $1 AND name = $2
		`, userID, entity.Name).Scan(&entityID)

		if err == sql.ErrNoRows {
			// Entity doesn't exist, create new one
			err = h.DB.QueryRow(`
				INSERT INTO entities (user_id, name, description, type, card_pk)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, userID, entity.Name, entity.Description, entity.Type, card.ID).Scan(&entityID)
			if err != nil {
				log.Printf("[EntityExtraction] Error inserting entity '%s': %v", entity.Name, err)
				continue
			}
			log.Printf("[EntityExtraction] Created new entity: %s (ID: %d)", entity.Name, entityID)
		} else if err != nil {
			log.Printf("[EntityExtraction] Error checking entity existence: %v", err)
			continue
		} else {
			// Entity exists, update it
			_, err = h.DB.Exec(`
				UPDATE entities SET description=$1, type=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3
			`, entity.Description, entity.Type, entityID)
			if err != nil {
				log.Printf("[EntityExtraction] Error updating entity '%s': %v", entity.Name, err)
				continue
			}
		}

		// Link entity to card
		_, err = h.DB.Exec(`
			INSERT INTO entity_card_junction (user_id, entity_id, card_pk, chunk_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (entity_id, card_pk) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
		`, userID, entityID, card.ID, 0)
		if err != nil {
			log.Printf("[EntityExtraction] Error linking entity '%s' to card: %v", entity.Name, err)
			continue
		}
	}

	return nil
}

// runSummarizationJobViaQueue enqueues a summarization job to the LLM job
// queue. The job carries only the summarization id (and optional card_pk);
// the map-reduce job loads the prepared input_text straight from the
// summarizations row. This replaces both the old goroutine-based approach and
// the analyses->payload->parse round-trip.
func (h *Handler) runSummarizationJobViaQueue(userID int, cardPK *int, summarizationID int) (int, error) {
	payload := map[string]interface{}{
		"summarization_id": summarizationID,
		"card_pk":          cardPK,
	}

	jobQueue := h.JobRunner
	job, err := jobQueue.Run(context.Background(), models.CreateJobParams{
		UserID:      userID,
		JobType:     models.JobTypeSummarization,
		Payload:     payload,
		MaxRetries:  3,
		TimeoutSecs: 300,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to enqueue summarization job: %w", err)
	}

	// Link summarization to LLM job
	_, err = h.DB.Exec(`UPDATE summarizations SET llm_job_id = $1 WHERE id = $2`, job.ID, summarizationID)
	if err != nil {
		log.Printf("Failed to link summarization %d to job %d: %v", summarizationID, job.ID, err)
	}

	return summarizationID, nil
}

// GetSummarizationRoute fetches a summarization job by id
func (h *Handler) GetSummarizationRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}
	idStr := mux.Vars(r)["id"]
	jobID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var job models.Summarization
	var result sql.NullString
	err = h.DB.QueryRow(`
		SELECT id, user_id, input_text, status, result,
		       prompt_tokens, completion_tokens, total_tokens, cost, model,
		       created_at, updated_at
		FROM summarizations
		WHERE id=$1 AND user_id=$2
	`, jobID, userID).Scan(
		&job.ID,
		&job.UserID,
		&job.InputText,
		&job.Status,
		&result,
		&job.PromptTokens,
		&job.CompletionTokens,
		&job.TotalTokens,
		&job.Cost,
		&job.Model,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		log.Printf("summarization job error %v", err)
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Convert sql.NullString to string, preserving the distinction between NULL and empty
	if result.Valid {
		job.Result = result.String
	} else {
		// Result is NULL - only set to empty string for non-complete statuses
		// For complete/failed statuses with NULL result, set to empty string
		job.Result = ""
	}

	resp := SummarizeJobResponse{
		ID:               job.ID,
		Status:           job.Status,
		Result:           job.Result,
		PromptTokens:     job.PromptTokens,
		CompletionTokens: job.CompletionTokens,
		TotalTokens:      job.TotalTokens,
		Cost:             job.Cost,
		Model:            job.Model,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CancelSummarizationRoute cancels a pending summarization job
func (h *Handler) CancelSummarizationRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}
	idStr := mux.Vars(r)["id"]
	summarizationID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	// Get llm_job_id for this summarization
	var llmJobID sql.NullInt64
	err = h.DB.QueryRow(`
		SELECT llm_job_id FROM summarizations WHERE id = $1 AND user_id = $2
	`, summarizationID, userID).Scan(&llmJobID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Summarization not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch summarization", http.StatusInternalServerError)
		}
		return
	}

	// Note: with inline processing (services.JobRunner) we cannot forcibly
	// cancel an in-flight goroutine. The user-visible effect is achieved by
	// marking the summarization itself as failed below; the linked audit row
	// will reflect whatever state the work reached.

	// Update summarization status
	_, err = h.DB.Exec(`UPDATE summarizations SET status='failed', updated_at=CURRENT_TIMESTAMP WHERE id=$1`, summarizationID)
	if err != nil {
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
