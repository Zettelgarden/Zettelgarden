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
	"strings"

	"github.com/gorilla/mux"
)

// Pre-compiled regex patterns for removeReferences
var (
	referencePattern         = regexp.MustCompile(`\[[^\]]+\] - [^\n]*\n?`)
	doubleNewlinePattern     = regexp.MustCompile(`\n\n+`)
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

// CreateSummarizationRoute creates a summarization job and enqueues it to the LLM job queue
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

	var jobID int
	var status string
	err := h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, input_text, status, created_at, updated_at)
			VALUES ($1, $2, 'pending', NOW(), NOW())
			RETURNING id, status
		`, userID, req.Text).Scan(&jobID, &status)

	if err != nil {
		log.Printf("error starting summarization %v", err)
		http.Error(w, "Failed to create summarization", http.StatusInternalServerError)
		return
	}

	// Extract theses and arguments
	isTesting := h.Server != nil && h.Server.Testing
	client := services.NewDefaultClient(h.DB, userID, isTesting)
	client.RequestType = "analysis"
	processedText := prepareTextForAnalysis(req.Title, req.Text)
	analyses, facts, usage, err := services.ExtractThesesAndArguments(client, processedText)
	if err != nil {
		log.Printf("Failed to extract theses: %v", err)
		http.Error(w, "Failed to analyze text", http.StatusInternalServerError)
		return
	}

	// Enqueue the summarization job
	summarizationID, err := h.runSummarizationJobViaQueue(userID, analyses, facts, usage, nil, jobID)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, "Failed to create summarization job", http.StatusInternalServerError)
		return
	}

	// Return actual status from database (not hardcoded "pending")
	resp := SummarizeJobResponse{ID: summarizationID, Status: status}
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

func (h *Handler) ProcessEntitiesAndFacts(userID int, card models.Card) {
	// Skip during testing to avoid external LLM calls
	if h.Server.Testing {
		return
	}

	var jobID int

	err := h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'pending', NOW(), NOW())
			RETURNING id
		`, userID, card.ID, "").Scan(&jobID)

	if err != nil {
		log.Printf("error starting summarization %v", err)
		return
	}

	go func() {
		// Panic recovery to prevent goroutine crashes
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] Recovered in ProcessEntitiesAndFacts goroutine: %v", r)
			}
		}()
		// Ensure LinkCardToEntityIfPossible is always called exactly once, regardless of success or failure
		defer h.LinkCardToEntityIfPossible(userID, card)

		isTesting := h.Server != nil && h.Server.Testing
		client := services.NewDefaultClient(h.DB, userID, isTesting)
		client.RequestType = "analysis"
		processedText := prepareTextForAnalysis(card.Title, card.Body)
		analyses, facts, usage, err := services.ExtractThesesAndArguments(client, processedText)
		if err != nil {
			log.Printf("Fact extraction failed: %v", err)
			return
		}

		// Save the detailed analysis linked to the job ID
		if err := h.SaveAnalysis(userID, card.ID, jobID, analyses); err != nil {
			log.Printf("Failed to save analysis: %v", err)
		}

		// Enqueue the summarization job
		_, err = h.runSummarizationJobViaQueue(userID, analyses, facts, usage, &card.ID, jobID)
		if err != nil {
			log.Printf("Failed to run summarization job: %v", err)
			return
		}

		log.Printf("facts %v", facts)
		if len(facts) > 0 {
			factObjs, err := h.ExtractSaveCardFacts(userID, card.ID, facts)
			if err != nil {
				log.Printf("Failed to save card facts: %v", err)
			} else {
				if err := h.ExtractSaveFactEntities(userID, card, factObjs); err != nil {
					log.Printf("Failed to extract/save fact entities: %v", err)
				}
			}
		}
	}()
}

// extractSectionOrder attempts to extract a section number from the section title.
// Returns the extracted number if found, otherwise falls back to the provided default index.
// Supports formats like "Section 1: Title", "Section 2", "1. Introduction", etc.
func extractSectionOrder(sectionTitle string, defaultIndex int) int {
	// Try to match common section patterns
	patterns := []string{
		"Section (\\d+)",    // "Section 1: Title"
		"Section\\s*(\\d+)", // "Section 1"
		"^(\\d+)\\.",        // "1. Introduction"
		"Part (\\d+)",       // "Part 1"
		"Chapter (\\d+)",    // "Chapter 1"
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(sectionTitle)
		if len(matches) > 1 {
			// Parse the captured number
			var num int
			_, err := fmt.Sscanf(matches[1], "%d", &num)
			if err == nil && num > 0 {
				return num
			}
		}
	}

	// Fall back to array index if no pattern matched
	return defaultIndex
}

// SaveAnalysis persists the structured analysis from the LLM into the database.
func (h *Handler) SaveAnalysis(userID, cardPK, summarizationID int, analyses []models.SectionAnalysis) error {
	// Validate cardPK is a positive integer
	if cardPK <= 0 {
		return fmt.Errorf("invalid card_pk: must be positive, got %d", cardPK)
	}

	tx, err := h.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Only rollback if we're not in testing mode (test framework handles cleanup)
	if h.ShouldCommitTx() {
		defer tx.Rollback() // Rollback on error, if commit fails
	}

	for sectionIndex, analysis := range analyses {
		// Skip sections with empty or whitespace-only titles
		sectionTitle := strings.TrimSpace(analysis.Section)
		if sectionTitle == "" {
			continue
		}

		// Try to extract section number from title for proper ordering
		// Falls back to array index if no number found
		sectionOrder := extractSectionOrder(sectionTitle, sectionIndex)

		// Insert Section - remove ON CONFLICT to allow multiple sections with same title
		// Add section_order to distinguish between sections with identical titles
		var sectionID int
		err := tx.QueryRow(`
			INSERT INTO summary_sections (user_id, card_pk, summarization_id, section_title, section_order)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, userID, cardPK, summarizationID, sectionTitle, sectionOrder).Scan(&sectionID)
		if err != nil {
			return fmt.Errorf("failed to insert section: %w", err)
		}

		for _, thesisEntry := range analysis.Theses {
			// Skip theses with empty or whitespace-only content
			thesis := strings.TrimSpace(thesisEntry.Thesis)
			if thesis == "" {
				continue
			}

			// Insert Thesis
			var thesisID int
			err := tx.QueryRow(`
				INSERT INTO summary_theses (user_id, card_pk, summarization_id, section_id, thesis)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, userID, cardPK, summarizationID, sectionID, thesis).Scan(&thesisID)
			if err != nil {
				return fmt.Errorf("failed to insert thesis: %w", err)
			}

			// Insert Arguments for the thesis
			for _, arg := range thesisEntry.Arguments {
				_, err := tx.Exec(`
					INSERT INTO summary_arguments (user_id, card_pk, summarization_id, thesis_id, argument, importance)
					VALUES ($1, $2, $3, $4, $5, $6)
				`, userID, cardPK, summarizationID, thesisID, arg.Argument, arg.Importance)
				if err != nil {
					return fmt.Errorf("failed to insert argument: %w", err)
				}
			}
		}
	}

	if h.ShouldCommitTx() {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// GetCardAnalysisRoute retrieves the analysis for the most recent summarization of a card.
func (h *Handler) GetCardAnalysisRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}
	cardPKStr := mux.Vars(r)["card_pk"]
	cardPK, err := strconv.Atoi(cardPKStr)
	if err != nil {
		http.Error(w, "Invalid card_pk", http.StatusBadRequest)
		return
	}

	analysis, err := services.GetCardAnalysis(h.DB, userID, cardPK)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load analysis: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// runSummarizationJobViaQueue enqueues a summarization job to the LLM job queue.
// This replaces the old goroutine-based approach with proper job queue integration.
func (h *Handler) runSummarizationJobViaQueue(userID int, analyses []models.SectionAnalysis, facts []string, usage models.Usage, cardPK *int, summarizationID int) (int, error) {
	// Convert analyses to JSON-serializable format for payload
	analysesPayload := make([]map[string]interface{}, len(analyses))
	for i, a := range analyses {
		thesesPayload := make([]map[string]interface{}, len(a.Theses))
		for j, t := range a.Theses {
			argsPayload := make([]map[string]interface{}, len(t.Arguments))
			for k, arg := range t.Arguments {
				argsPayload[k] = map[string]interface{}{
					"argument":   arg.Argument,
					"importance": arg.Importance,
				}
			}
			thesesPayload[j] = map[string]interface{}{
				"thesis":    t.Thesis,
				"arguments": argsPayload,
			}
		}
		analysesPayload[i] = map[string]interface{}{
			"section":  a.Section,
			"theses":   thesesPayload,
		}
	}

	payload := map[string]interface{}{
		"summarization_id": summarizationID,
		"card_pk":          cardPK,
		"analyses":         analysesPayload,
		"facts":            facts,
		"usage": map[string]interface{}{
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
			"total_cost":        usage.TotalCost,
		},
	}

	jobQueue := services.NewJobQueue(h.DB)
	job, err := jobQueue.Enqueue(context.Background(), models.CreateJobParams{
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

	// Cancel via job queue if linked
	if llmJobID.Valid {
		jobQueue := services.NewJobQueue(h.DB)
		err = jobQueue.Cancel(context.Background(), int(llmJobID.Int64), userID)
		if err != nil {
			// Job may have already started processing - still update summarization status
			log.Printf("Failed to cancel LLM job %d: %v", llmJobID.Int64, err)
		}
	}

	// Update summarization status
	_, err = h.DB.Exec(`UPDATE summarizations SET status='failed', updated_at=NOW() WHERE id=$1`, summarizationID)
	if err != nil {
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
