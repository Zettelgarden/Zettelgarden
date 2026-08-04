package services

import (
	"context"
	"errors"
	"fmt"
	"go-backend/models"
	"log"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// MaxChunkSize is the maximum size of a text chunk for LLM processing
	MaxChunkSize = 15000

	// DefaultSummarizeModel is the default model used for summarization
	DefaultSummarizeModel = "glm-5.1"

	// EnvSummarizeModel is the environment variable name for the summarize model
	EnvSummarizeModel = "ZETTEL_LLM_SUMMARIZE_MODEL"

	// Cost per million tokens for summarization (using OpenAI-compatible pricing as baseline)
	PromptCostPerMillion     = 1.25
	CompletionCostPerMillion = 10.0

	// LLMRequestTimeout is the maximum time to wait for an LLM request to complete
	LLMRequestTimeout = 5 * time.Minute

	// MaxLLMRetries is the maximum number of retry attempts for transient LLM failures
	MaxLLMRetries = 3

	// InitialRetryDelay is the initial delay before the first retry (will be doubled each time)
	InitialRetryDelay = 500 * time.Millisecond
)

// MaxChunkFailureRate is the maximum percentage of chunks that can fail before
// SummarizeChunks returns an error. Successful summaries are still returned
// when the failure rate is within this threshold.
const MaxChunkFailureRate = 0.5 // 50%

// llmRequestFunc is the LLM call used by the summarizer. It defaults to the
// transient-failure retry wrapper; tests may override it to inject responses.
var llmRequestFunc = executeLLMRequestWithRetry

// isSummarizationRetryableError checks if an error is transient and should be retried for summarization
func isSummarizationRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for common transient error patterns
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"rate limit",
		"too many requests",
		"server error",
		"503",
		"502",
		"500",
		"context deadline exceeded",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errMsg), pattern) {
			return true
		}
	}
	return false
}

// executeLLMRequestWithRetry wraps ExecuteLLMRequest with retry logic for transient failures
func executeLLMRequestWithRetry(ctx context.Context, c *models.LLMClient, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	var lastErr error
	var resp openai.ChatCompletionResponse

	for attempt := 0; attempt <= MaxLLMRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 1s, 2s, etc.
			delay := InitialRetryDelay * time.Duration(1<<(attempt-1))
			log.Printf("LLM request failed (attempt %d/%d), retrying after %v: %v", attempt, MaxLLMRetries, delay, lastErr)
			time.Sleep(delay)
		}

		resp, lastErr = ExecuteLLMRequest(ctx, c, messages)
		if lastErr == nil {
			if attempt > 0 {
				log.Printf("LLM request succeeded after %d retries", attempt)
			}
			return resp, nil
		}

		// If error is not retryable, fail immediately
		if !isSummarizationRetryableError(lastErr) {
			log.Printf("LLM request failed with non-retryable error: %v", lastErr)
			return openai.ChatCompletionResponse{}, lastErr
		}
	}

	return openai.ChatCompletionResponse{}, fmt.Errorf("LLM request failed after %d retries: %w", MaxLLMRetries, lastErr)
}

// mapSystemPrompt instructs the model to produce a plain-text interim summary
// of a single chunk. No JSON, so there is nothing to parse or repair.
const mapSystemPrompt = `You are an assistant that summarizes a section of a longer document.

Preserve the key theses and the supporting arguments, evidence, and examples
from this section. Write a faithful, self-contained interim summary in plain
text (no markdown headings, no JSON, no preamble).

Length: aim for roughly 150–250 words. Do not write an introduction or
conclusion — this summary will be combined with others into a final summary.`

// SummarizeChunks is the MAP step of the summarizer. It splits the input into
// chunks and summarizes each one independently into a plain-text interim
// summary. Chunks are independent, so this is trivially parallelizable.
//
// It returns one summary per successfully processed chunk and accumulated
// usage. If more than MaxChunkFailureRate of the chunks fail, it returns an
// error. (Successful summaries within the threshold are still returned.)
func SummarizeChunks(c *models.LLMClient, input string) ([]string, models.Usage, error) {
	start := time.Now()
	chunks := ChunkText(input)

	log.Printf("[METRICS] SummarizeChunks started - input_size=%d chars, chunks=%d, model=%s",
		len(input), len(chunks), c.Model)

	totalPromptTokens := 0
	totalCompletionTokens := 0
	failedChunks := 0
	summaries := make([]string, 0, len(chunks))

	for i, chunk := range chunks {
		userContent := fmt.Sprintf(
			"This is chunk %d of %d of a larger work. Summarize this section:\n\n%s",
			i+1, len(chunks), chunk)

		messages := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: mapSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		}

		ctx, cancel := context.WithTimeout(context.Background(), LLMRequestTimeout)
		resp, err := llmRequestFunc(ctx, c, messages)
		cancel()
		if err != nil {
			log.Printf("[WARNING] chunk %d/%d failed: %v", i+1, len(chunks), err)
			failedChunks++
			continue
		}
		if len(resp.Choices) == 0 {
			log.Printf("[WARNING] chunk %d/%d returned no choices", i+1, len(chunks))
			failedChunks++
			continue
		}

		summaries = append(summaries, strings.TrimSpace(resp.Choices[0].Message.Content))
		totalPromptTokens += resp.Usage.PromptTokens
		totalCompletionTokens += resp.Usage.CompletionTokens
	}

	// Check if too many chunks failed
	totalChunks := len(chunks)
	if totalChunks > 0 {
		failureRate := float64(failedChunks) / float64(totalChunks)
		if failureRate > MaxChunkFailureRate {
			log.Printf("[ERROR] Too many chunks failed - failed=%d/%d (%.1f%%), threshold=%.1f%%",
				failedChunks, totalChunks, failureRate*100, MaxChunkFailureRate*100)
			return nil, models.Usage{}, fmt.Errorf("too many chunks failed: %d/%d (%.1f%%), threshold is %.1f%%",
				failedChunks, totalChunks, failureRate*100, MaxChunkFailureRate*100)
		}
		if failedChunks > 0 {
			log.Printf("[WARNING] Some chunks failed but within threshold - failed=%d/%d (%.1f%%), threshold=%.1f%%",
				failedChunks, totalChunks, failureRate*100, MaxChunkFailureRate*100)
		}
	}

	if len(summaries) == 0 {
		log.Printf("[METRICS] SummarizeChunks failed - no chunk summaries returned")
		return nil, models.Usage{}, errors.New("no chunk summaries returned")
	}

	usage := models.Usage{
		PromptTokens:     totalPromptTokens,
		CompletionTokens: totalCompletionTokens,
		TotalTokens:      totalPromptTokens + totalCompletionTokens,
	}

	elapsed := time.Since(start)
	log.Printf("[METRICS] SummarizeChunks completed - "+
		"duration=%v, chunks=%d, summaries=%d, failed_chunks=%d/%d, "+
		"prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, model=%s",
		elapsed, totalChunks, len(summaries), failedChunks, totalChunks,
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, c.Model)

	return summaries, usage, nil
}

// reducePrompt is the final-summarization prompt. It is identical to the one
// the old pipeline's final call used, so the user-visible output format
// (Executive Summary + Reference Summary) is preserved. The concatenated
// chunk-summaries are fed in as the <analysis> block.
const reducePrompt = `
Summarize the following aggregated analysis into a two-part markdown summary. 
The output should be **structured, concise, and tailored to distinct audiences**.

### Instructions:
1. **Format:** Use headings, subheadings, and bullets for clarity.
    - Do not include any other text in your output, including follow up questions or any pleasantries, just respond with the summarized info.

2. **Section 1: Executive Summary**  
   - Audience: Senior management, decision-makers, or non-specialist readers.  
   - Style: Concise, strategic, and outcome-focused.  
   - Length: ~4–6 bullet points.  
   - Emphasize:  
     - Main conclusions or big-picture trends.  
     - Strategic implications.  
     - Key trade-offs or future outlook.  
   - Avoid: Technical jargon, long lists, or granular details.

3. **Section 2: Reference Summary**  
   - Audience: Researchers, analysts, technical leads, or specialists.  
   - Style: Well-structured, factual, and precise.  
   - Include:  
     - **Main Theses** (core claims or insights), organize by section.  
     - **Supporting Arguments** (reasoning behind these theses).  
     - **Key Evidence** (5–8 of the most decisive data points, milestones, or examples).  
   - Present information in a hierarchy (for each thesis, show its supporting arguments).  
   - Exclude secondary/tangential details.

4. **General Guidelines:**  
   - Focus only on what is **strategically or academically important** to understand the subject.  
   - Omit extraneous digressions, trivia, or minor historical detail.  
   - Deduplicate overlapping points from the interim summaries.  
   - Keep each section readable on its own.  
   - Do not return anything other than the details, no pleasantries!
   - Tone:  
     - Section 1 → plain, polished, and accessible ("boardroom-ready").  
     - Section 2 → objective, precise, and reference-style ("briefing document").  

Input (interim summaries of each section of the source document):
`

// SummarizeReduce is the REDUCE step of the summarizer. It concatenates the
// chunk-summaries produced by SummarizeChunks into the final two-section
// markdown summary. Deduplication is handled inline by the prompt — there is
// no separate dedup/rank LLM call.
//
// usage carries the MAP step's accumulated token counts so the returned Usage
// reflects the whole map-reduce.
func SummarizeReduce(c *models.LLMClient, chunkSummaries []string, usage models.Usage) (string, models.Usage, error) {
	start := time.Now()
	c.Model = os.Getenv(EnvSummarizeModel)
	if c.Model == "" {
		c.Model = DefaultSummarizeModel
	}

	log.Printf("[METRICS] SummarizeReduce started - chunk_summaries=%d, model=%s", len(chunkSummaries), c.Model)

	analysis := "<analysis>\n" + strings.Join(chunkSummaries, "\n\n---\n\n") + "\n</analysis>"

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: "You are an assistant that summarizes text clearly and concisely."},
		{Role: openai.ChatMessageRoleUser, Content: reducePrompt + analysis},
	}

	ctx, cancel := context.WithTimeout(context.Background(), LLMRequestTimeout)
	resp, err := llmRequestFunc(ctx, c, messages)
	cancel()
	if err != nil {
		return "", models.Usage{}, err
	}
	if len(resp.Choices) == 0 {
		return "", models.Usage{}, errors.New("no summary returned")
	}

	totalPromptTokens := usage.PromptTokens + resp.Usage.PromptTokens
	totalCompletionTokens := usage.CompletionTokens + resp.Usage.CompletionTokens
	totalCost := float64(totalPromptTokens)/1_000_000*PromptCostPerMillion +
		float64(totalCompletionTokens)/1_000_000*CompletionCostPerMillion

	resultUsage := models.Usage{
		PromptTokens:     totalPromptTokens,
		CompletionTokens: totalCompletionTokens,
		TotalTokens:      totalPromptTokens + totalCompletionTokens,
		TotalCost:        totalCost,
	}

	summary := resp.Choices[0].Message.Content

	elapsed := time.Since(start)
	log.Printf("[METRICS] SummarizeReduce completed - "+
		"duration=%v, summary_length=%d chars, chunk_summaries=%d, "+
		"prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, cost=$%.4f, model=%s",
		elapsed, len(summary), len(chunkSummaries),
		resultUsage.PromptTokens, resultUsage.CompletionTokens, resultUsage.TotalTokens, resultUsage.TotalCost, c.Model)

	return summary, resultUsage, nil
}
