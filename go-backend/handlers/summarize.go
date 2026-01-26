package handlers

import (
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
	"time"

	"github.com/gorilla/mux"
)

// removeReferences removes card reference lines from text before summarization
// References are lines that start with [tag] - title and end with newline
func removeReferences(text string) string {
	// Pattern matches: [anything] - anything followed by newline
	// Also handles the case where reference is at the end without trailing newline
	referencePattern := regexp.MustCompile(`\[[^\]]+\] - [^\n]*\n?`)
	result := referencePattern.ReplaceAllString(text, "")

	// Clean up any resulting double newlines to avoid empty lines
	doubleNewlinePattern := regexp.MustCompile(`\n\n+`)
	result = doubleNewlinePattern.ReplaceAllString(result, "\n\n")

	// Trim trailing whitespace
	result = regexp.MustCompile(`\s+$`).ReplaceAllString(result, "")

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
	userID := r.Context().Value("current_user").(int)
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
	userID := r.Context().Value("current_user").(int)

	summaries, err := h.querySummarizations(userID, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// querySummarizations is a shared helper that queries summarizations for a user,
// optionally filtered by card_pk
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

// CreateSummarizationRoute creates a summarization job and runs it asynchronously
func (h *Handler) CreateSummarizationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Check rate limit
	if !h.checkSummarizationRateLimit(userID) {
		log.Printf("[RATE_LIMIT] User %d exceeded summarization rate limit", userID)
		http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Check concurrent job limit
	if !h.acquireSummarizationJobSlot(userID) {
		log.Printf("[CONCURRENCY_LIMIT] User %d reached maximum concurrent summarizations", userID)
		http.Error(w, "Too many concurrent jobs. Please wait for existing jobs to complete.", http.StatusTooManyRequests)
		return
	}

	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		h.releaseSummarizationJobSlot(userID)
		return
	}

	var jobID int
	err := h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, input_text, status, created_at, updated_at)
			VALUES ($1, $2, 'pending', NOW(), NOW())
			RETURNING id
		`, userID, "").Scan(&jobID)

	if err != nil {
		log.Printf("error starting summarization %v", err)
		h.releaseSummarizationJobSlot(userID)
		return
	}
	_, _ = h.DB.Exec(`UPDATE summarizations SET status='processing', updated_at=$2 WHERE id=$1`, jobID, time.Now())
	client := services.NewDefaultClient(h.DB, userID)
	client.RequestType = "analysis"
	processedText := prepareTextForAnalysis(req.Title, req.Text)
	analyses, facts, usage, err := services.ExtractThesesAndArguments(client, processedText)
	id, err := h.runSummarizationJob(userID, analyses, facts, usage, nil, jobID)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, "Failed to create summarization job", http.StatusInternalServerError)
		h.releaseSummarizationJobSlot(userID)
		return
	}

	resp := SummarizeJobResponse{ID: id, Status: "pending"}
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

	// Check concurrent job limit (but skip rate limit for background jobs)
	if !h.acquireSummarizationJobSlot(userID) {
		log.Printf("[CONCURRENCY_LIMIT] User %d reached maximum concurrent summarizations for background job", userID)
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
		h.releaseSummarizationJobSlot(userID)
		return
	}
	_, _ = h.DB.Exec(`UPDATE summarizations SET status='processing', updated_at=$2 WHERE id=$1`, jobID, time.Now())

	go func() {
		// Release the job slot when done
		defer h.releaseSummarizationJobSlot(userID)

		// Ensure LinkCardToEntityIfPossible is always called exactly once, regardless of success or failure
		defer h.LinkCardToEntityIfPossible(userID, card)

		client := services.NewDefaultClient(h.DB, userID)
		client.RequestType = "analysis"
		processedText := prepareTextForAnalysis(card.Title, card.Body)
		analyses, facts, usage, err := services.ExtractThesesAndArguments(client, processedText)
		if err != nil {
			log.Printf("Fact extraction failed: %v", err)
			return
		}

		// Run the summarization job to get a job ID
		jobID, err := h.runSummarizationJob(userID, analyses, facts, usage, &card.ID, jobID)
		if err != nil {
			log.Printf("Failed to run summarization job: %v", err)
			return
		}

		// Save the detailed analysis linked to the job ID
		if err := h.SaveAnalysis(userID, card.ID, jobID, analyses); err != nil {
			log.Printf("Failed to save analysis: %v", err)
			// Even if saving analysis fails, we can still try to link entities via defer
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

	tx, err := h.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error, if commit fails

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

	return tx.Commit()
}

// GetCardAnalysisRoute retrieves the analysis for the most recent summarization of a card.
func (h *Handler) GetCardAnalysisRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
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

// runSummarizationJob inserts a summarization job and runs it asynchronously.
func (h *Handler) runSummarizationJob(userID int, analyses []models.SectionAnalysis, facts []string, usage models.Usage, cardPK *int, jobID int) (int, error) {
	// Background job
	go func(jobID int, analyses []models.SectionAnalysis, facts []string, usage models.Usage, uid int) {
		// Release the job slot when done (defer ensures it runs on both success and failure)
		defer h.releaseSummarizationJobSlot(uid)

		client := services.NewDefaultClient(h.DB, uid)
		client.RequestType = "analysis"
		_, _ = h.DB.Exec(`UPDATE summarizations SET status='processing', updated_at=$2 WHERE id=$1`, jobID, time.Now())

		result, _, usage, err := services.AnalyzeAndSummarizeText(client, analyses, facts, usage)
		if err != nil {
			_, _ = h.DB.Exec(`UPDATE summarizations SET status='failed', result=$2, updated_at=$3 WHERE id=$1`,
				jobID, err.Error(), time.Now())
			return
		}

		// modelName := client.Model.ModelIdentifier

		_, _ = h.DB.Exec(`UPDATE summarizations
			SET status='complete', result=$2, prompt_tokens=$3, completion_tokens=$4, total_tokens=$5, cost=$6, model=$7, updated_at=$8
			WHERE id=$1`,
			jobID, result, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.TotalCost, "deprecated", time.Now())

	}(jobID, analyses, facts, usage, userID)

	return jobID, nil
}

// GetSummarizationRoute fetches a summarization job by id
func (h *Handler) GetSummarizationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
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
