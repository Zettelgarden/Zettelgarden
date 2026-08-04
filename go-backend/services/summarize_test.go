package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go-backend/models"

	openai "github.com/sashabaranov/go-openai"
)

// respOrErr is a scripted response for the llmRequestFunc override.
type respOrErr struct {
	content string
	err     error
}

// scriptLLM overrides the summarizer's LLM call seam with a per-call script.
// Calls beyond the script length repeat the last entry. The original seam is
// restored automatically via t.Cleanup.
func scriptLLM(t *testing.T, script ...respOrErr) {
	t.Helper()
	orig := llmRequestFunc
	var call int
	llmRequestFunc = func(ctx context.Context, c *models.LLMClient, msgs []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
		i := call
		call++
		if i >= len(script) {
			i = len(script) - 1
		}
		r := script[i]
		if r.err != nil {
			return openai.ChatCompletionResponse{}, r.err
		}
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: r.content,
				},
			}},
			Usage: openai.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}, nil
	}
	t.Cleanup(func() { llmRequestFunc = orig })
}

func testClient() *models.LLMClient {
	return &models.LLMClient{Model: DefaultSummarizeModel}
}

// TestSummarizeChunks_HappyPath verifies the MAP step returns one plain-text
// summary per chunk and accumulates usage across all chunk calls.
func TestSummarizeChunks_HappyPath(t *testing.T) {
	// Force small chunks so a modest input produces several.
	t.Setenv("ZETTEL_CHUNK_SIZE", "40")

	// Provide enough scripted responses for however many chunks are produced.
	scriptLLM(t,
		respOrErr{content: "summary-alpha"},
		respOrErr{content: "summary-beta"},
		respOrErr{content: "summary-gamma"},
		respOrErr{content: "summary-delta"},
	)

	summaries, usage, err := SummarizeChunks(testClient(),
		strings.Repeat("This is a test sentence with some words. ", 200))
	if err != nil {
		t.Fatalf("SummarizeChunks: unexpected error: %v", err)
	}
	if len(summaries) < 2 {
		t.Fatalf("expected multiple chunk summaries, got %d", len(summaries))
	}
	for i, s := range summaries {
		if s == "" {
			t.Errorf("chunk summary %d is empty", i)
		}
	}
	// Each successful call contributes 100 prompt + 50 completion tokens.
	wantPrompt := 100 * len(summaries)
	wantCompletion := 50 * len(summaries)
	if usage.PromptTokens != wantPrompt || usage.CompletionTokens != wantCompletion {
		t.Errorf("usage = (prompt=%d completion=%d), want (prompt=%d completion=%d)",
			usage.PromptTokens, usage.CompletionTokens, wantPrompt, wantCompletion)
	}
	if usage.TotalTokens != wantPrompt+wantCompletion {
		t.Errorf("TotalTokens = %d, want %d", usage.TotalTokens, wantPrompt+wantCompletion)
	}
}

// TestSummarizeChunks_FailureThreshold verifies that when more than
// MaxChunkFailureRate of chunks fail, SummarizeChunks returns an error.
func TestSummarizeChunks_FailureThreshold(t *testing.T) {
	// A single-chunk input that fails = 100% failure rate (> 50% threshold).
	scriptLLM(t, respOrErr{err: fmt.Errorf("boom")})

	_, _, err := SummarizeChunks(testClient(), "short single-chunk input")
	if err == nil {
		t.Fatal("expected error when all chunks fail, got nil")
	}
	if !strings.Contains(err.Error(), "too many chunks failed") {
		t.Errorf("expected 'too many chunks failed' error, got: %v", err)
	}
}

// TestSummarizeChunks_PartialFailure verifies that failures within the
// threshold still yield the successful summaries.
func TestSummarizeChunks_PartialFailure(t *testing.T) {
	t.Setenv("ZETTEL_CHUNK_SIZE", "40")

	// Fail every call: with many chunks the failure rate is 100% -> error.
	// Instead, alternate fail/succeed so ~half succeed (within threshold).
	script := []respOrErr{
		{err: fmt.Errorf("transient")},
		{content: "ok-1"},
		{err: fmt.Errorf("transient")},
		{content: "ok-2"},
		{err: fmt.Errorf("transient")},
		{content: "ok-3"},
	}
	scriptLLM(t, script...)

	summaries, _, err := SummarizeChunks(testClient(),
		strings.Repeat("This is a test sentence with some words. ", 200))
	if err != nil {
		t.Fatalf("expected success within threshold, got error: %v", err)
	}
	if len(summaries) == 0 {
		t.Fatal("expected at least one summary despite partial failures")
	}
}

// TestSummarizeReduce verifies the REDUCE step concatenates the
// chunk-summaries via the prompt and returns the model's markdown, while
// folding the MAP usage into the returned totals.
func TestSummarizeReduce(t *testing.T) {
	scriptLLM(t, respOrErr{content: "## Executive Summary\n\n- point one\n\n## Reference Summary\n\n- detail"})

	mapUsage := models.Usage{PromptTokens: 300, CompletionTokens: 150, TotalTokens: 450}
	chunkSummaries := []string{"alpha summary", "beta summary"}

	out, usage, err := SummarizeReduce(testClient(), chunkSummaries, mapUsage)
	if err != nil {
		t.Fatalf("SummarizeReduce: unexpected error: %v", err)
	}
	if !strings.Contains(out, "Executive Summary") {
		t.Errorf("expected markdown summary, got: %q", out)
	}
	// Reduce call adds 100 prompt + 50 completion on top of the map usage.
	if usage.PromptTokens != 400 || usage.CompletionTokens != 200 {
		t.Errorf("reduce usage = (prompt=%d completion=%d), want (400, 200)",
			usage.PromptTokens, usage.CompletionTokens)
	}
	if usage.TotalCost <= 0 {
		t.Errorf("expected positive TotalCost, got %v", usage.TotalCost)
	}
}

// TestProcessSummarizationJob verifies the end-to-end job: it loads the
// input_text from the summarizations row, runs map-reduce with the stubbed
// client, and writes the result back as status='complete'.
func TestProcessSummarizationJob(t *testing.T) {
	db := freshSQLiteDB(t)
	// Minimal table covering only the columns the job touches (no FKs, to
	// keep this a focused unit test of the summarizer logic).
	_, err := db.Exec(`CREATE TABLE summarizations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		input_text TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		result TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		prompt_tokens INTEGER DEFAULT 0,
		completion_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		cost REAL DEFAULT 0,
		model TEXT DEFAULT ''
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	const userID = 1
	var summarizationID int
	err = db.QueryRow(`INSERT INTO summarizations (user_id, input_text, status) VALUES ($1, $2, 'pending') RETURNING id`,
		userID, "Some input text to summarize.").Scan(&summarizationID)
	if err != nil {
		t.Fatalf("insert row: %v", err)
	}

	scriptLLM(t,
		respOrErr{content: "interim chunk summary"},
		respOrErr{content: "## Executive Summary\n\n- the point"},
	)

	p := NewLLMJobProcessor(db)
	job := &models.LLMJob{
		UserID:  userID,
		JobType: models.JobTypeSummarization,
		Payload: map[string]interface{}{
			"summarization_id": float64(summarizationID),
		},
	}

	res, err := p.processSummarizationJob(context.Background(), job)
	if err != nil {
		t.Fatalf("processSummarizationJob: unexpected error: %v", err)
	}
	if got := res["status"]; got != "completed" {
		t.Errorf("job result status = %v, want \"completed\"", got)
	}
	if got, _ := res["result"].(string); got == "" {
		t.Errorf("job result result is empty")
	}

	var status, result, model string
	var totalTokens int
	err = db.QueryRow(`SELECT status, result, model, total_tokens FROM summarizations WHERE id = $1`, summarizationID).
		Scan(&status, &result, &model, &totalTokens)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}
	if status != "complete" {
		t.Errorf("row status = %q, want \"complete\"", status)
	}
	if result == "" {
		t.Error("row result is empty; expected the reduce output")
	}
	if model == "" {
		t.Error("row model is empty; expected it to be recorded")
	}
	if totalTokens == 0 {
		t.Error("row total_tokens = 0; expected accumulated usage")
	}
}

// TestProcessSummarizationJob_MissingPayload verifies the job fails fast when
// the payload lacks summarization_id.
func TestProcessSummarizationJob_MissingPayload(t *testing.T) {
	db := freshSQLiteDB(t)
	p := NewLLMJobProcessor(db)
	_, err := p.processSummarizationJob(context.Background(), &models.LLMJob{
		UserID:  1,
		JobType: models.JobTypeSummarization,
		Payload: map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected error for missing summarization_id, got nil")
	}
}
