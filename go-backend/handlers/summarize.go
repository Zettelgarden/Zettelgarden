package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/llms"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// SummarizeRequest defines the payload for creating a summarization job
type SummarizeRequest struct {
	Text  string `json:"text"`
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

	rows, err := h.DB.Query(`
		SELECT id, status, COALESCE(result, '')
		FROM summarizations
		WHERE user_id = $1 AND card_pk = $2
		ORDER BY created_at DESC
	`, userID, cardID)
	if err != nil {
		http.Error(w, "Failed to query summarizations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var summaries []SummarizeJobResponse
	for rows.Next() {
		var job SummarizeJobResponse
		if err := rows.Scan(&job.ID, &job.Status, &job.Result); err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}
		summaries = append(summaries, job)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// ListSummarizationsRoute returns all summarization jobs (lightweight view) for the current user
func (h *Handler) ListSummarizationsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	rows, err := h.DB.Query(`
		SELECT id, status, COALESCE(result, '')
		FROM summarizations
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, "Failed to query summarizations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var jobs []SummarizeJobResponse
	for rows.Next() {
		var job SummarizeJobResponse
		if err := rows.Scan(&job.ID, &job.Status, &job.Result); err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}
		jobs = append(jobs, job)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// CreateSummarizationRoute creates a summarization job and runs it asynchronously
func (h *Handler) CreateSummarizationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	client := llms.NewDefaultClient(h.DB, userID)
	client.Model.ModelIdentifier = "openai/gpt-5-chat"
	analyses, usage, err := llms.ExtractThesesAndArguments(client, req.Text)
	id, err := h.runSummarizationJob(userID, analyses, usage, nil)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, "Failed to create summarization job", http.StatusInternalServerError)
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

	//wordCount := len(strings.Fields(card.Body))
	go func() {
		client := llms.NewDefaultClient(h.DB, userID)
		client.Model.ModelIdentifier = "openai/gpt-5-chat"
		analyses, usage, err := llms.ExtractThesesAndArguments(client, card.Body)
		if err != nil {
			log.Printf("Fact extraction failed: %v", err)

			// todo think about how this should really work, this is a hack to make sure this happens regardless
			h.LinkCardToEntityIfPossible(userID, card)
			return
		}

		// Run the summarization job to get a job ID
		jobID, err := h.runSummarizationJob(userID, analyses, usage, &card.ID)
		if err != nil {
			log.Printf("Failed to run summarization job: %v", err)
			return
		}

		// Save the detailed analysis linked to the job ID
		if err := h.SaveAnalysis(userID, card.ID, jobID, analyses); err != nil {
			log.Printf("Failed to save analysis: %v", err)
			// Even if saving analysis fails, we can still try to link entities
		}

		// Extract entities from the originally extracted facts and link them.
		// This can be refactored to use the newly saved facts if needed, but for now, retains existing entity logic.
		var allFacts []string
		for _, analysis := range analyses {
			for _, th := range analysis.Theses {
				allFacts = append(allFacts, th.Facts...)
			}
		}
		if len(allFacts) > 0 {
			// Note: This creates facts that are not linked to a thesis/summarization.
			// The SaveAnalysis function already saves facts correctly.
			// We are only running this to feed the entity extractor.
			// This could be refactored to extract entities from the `analyses` directly.
			facts, _ := h.ExtractSaveCardFacts(userID, card.ID, allFacts)
			_ = h.ExtractSaveFactEntities(userID, card, facts)
		}

		h.LinkCardToEntityIfPossible(userID, card)
	}()
}

// SaveAnalysis persists the structured analysis from the LLM into the database.
func (h *Handler) SaveAnalysis(userID, cardPK, summarizationID int, analyses []llms.SectionAnalysis) error {
	tx, err := h.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error, if commit fails

	for _, analysis := range analyses {
		// Insert Section
		var sectionID int
		// Use ON CONFLICT DO NOTHING to handle unique constraint violation gracefully, though it implies the section already exists for this job.
		// A better approach might be to query first, but for this workflow, assuming titles are unique per job run is okay.
		err := tx.QueryRow(`
			INSERT INTO summary_sections (user_id, card_pk, summarization_id, section_title)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (summarization_id, section_title) DO UPDATE SET section_title = EXCLUDED.section_title
			RETURNING id
		`, userID, cardPK, summarizationID, analysis.Section).Scan(&sectionID)
		if err != nil {
			return fmt.Errorf("failed to insert or get section: %w", err)
		}

		for _, thesisEntry := range analysis.Theses {
			if thesisEntry.Thesis == "" {
				continue
			}

			// Insert Thesis
			var thesisID int
			err := tx.QueryRow(`
				INSERT INTO summary_theses (user_id, card_pk, summarization_id, section_id, thesis)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, userID, cardPK, summarizationID, sectionID, thesisEntry.Thesis).Scan(&thesisID)
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

// runSummarizationJob inserts a summarization job and runs it asynchronously.
func (h *Handler) runSummarizationJob(userID int, analyses []llms.SectionAnalysis, usage llms.Usage, cardPK *int) (int, error) {
	var id int
	var err error

	if cardPK != nil {
		err = h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, card_pk, input_text, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'pending', NOW(), NOW())
			RETURNING id
		`, userID, *cardPK, "").Scan(&id)
	} else {
		err = h.DB.QueryRow(`
			INSERT INTO summarizations (user_id, input_text, status, created_at, updated_at)
			VALUES ($1, $2, 'pending', NOW(), NOW())
			RETURNING id
		`, userID, "").Scan(&id)
	}
	if err != nil {
		return 0, err
	}

	// Background job
	go func(jobID int, analyses []llms.SectionAnalysis, usage llms.Usage, uid int) {
		client := llms.NewDefaultClient(h.DB, uid)
		_, _ = h.DB.Exec(`UPDATE summarizations SET status='processing', updated_at=$2 WHERE id=$1`, jobID, time.Now())

		result, _, usage, err := llms.AnalyzeAndSummarizeText(client, analyses, usage)
		if err != nil {
			_, _ = h.DB.Exec(`UPDATE summarizations SET status='failed', result=$2, updated_at=$3 WHERE id=$1`,
				jobID, err.Error(), time.Now())
			return
		}

		modelName := client.Model.ModelIdentifier

		_, _ = h.DB.Exec(`UPDATE summarizations 
			SET status='complete', result=$2, prompt_tokens=$3, completion_tokens=$4, total_tokens=$5, cost=$6, model=$7, updated_at=$8 
			WHERE id=$1`,
			jobID, result, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.TotalCost, modelName, time.Now())

	}(id, analyses, usage, userID)

	return id, nil
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
	err = h.DB.QueryRow(`
		SELECT id, user_id, input_text, status, COALESCE(result, ''), 
		       prompt_tokens, completion_tokens, total_tokens, cost, model,
		       created_at, updated_at
		FROM summarizations
		WHERE id=$1 AND user_id=$2
	`, jobID, userID).Scan(
		&job.ID,
		&job.UserID,
		&job.InputText,
		&job.Status,
		&job.Result,
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
