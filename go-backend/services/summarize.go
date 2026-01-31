package services

import (
	"context"
	"database/sql"
	"encoding/json"
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
	DefaultSummarizeModel = "google/gemini-3-pro-preview"

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

// MaxChunkFailureRate is the maximum percentage of chunks that can fail before returning an error
const MaxChunkFailureRate = 0.5 // 50%

// ExtractThesesAndArguments processes input text into SectionAnalysis entries,
// aggregating theses, facts, and arguments from each chunk.
// Returns all analyses and usage statistics.
func ExtractThesesAndArguments(c *models.LLMClient, input string) ([]models.SectionAnalysis, []string, models.Usage, error) {
	start := time.Now()
	chunks := ChunkText(input)

	// Log input metrics
	log.Printf("[METRICS] ExtractThesesAndArguments started - input_size=%d chars, chunks=%d, model=%s",
		len(input), len(chunks), c.Model)

	var facts []string
	var jsonRepairAttempts int
	var jsonRepairSuccesses int
	var failedChunks int // Track chunks that failed to process

	totalPromptTokens := 0
	totalCompletionTokens := 0
	var completedSections []models.SectionAnalysis      // Store completed sections
	var currentSectionAnalyses []models.SectionAnalysis // Current working sections
	var lastSectionName string                          // Track the last section name to detect transitions
	var cachedSectionJSON string                        // Cache marshaled JSON to avoid re-marshaling on every chunk
	var cacheValid bool                                 // Track if the cached JSON is valid

	for _, chunk := range chunks {
		// Build context intro from existing analyses
		var contextIntro string
		contextIntro, cachedSectionJSON, cacheValid = buildContextIntro(currentSectionAnalyses, cachedSectionJSON, cacheValid)

		// Build user content for this chunk
		userContent := buildUserContent(chunk, contextIntro, lastSectionName)

		messages := []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `You are an assistant that extracts theses, facts, and arguments from text.
				We are trying to come up with a coherent summary of the article/podcast/book/etc. You will be looking at
				some or all of the writing and need to extract certain things from it.
				Inside the <existing_analyses> block, you will find the current section being analyzed.
				Use this to understand the context and continue building on the current section.

Instructions:
- Respond ONLY in pure JSON with the following format.
- Do not add commentary, explanations, or non‑JSON text.
- If an item cannot be extracted, return an empty string or empty list.
- Importance must be an integer on a scale of 1–10 (10 = crucial to the central thesis, 1 = marginal).
- Facts should be discrete, verifiable statements (events, statistics, claims of evidence).
- Facts should *not* be about the text itself. For example, if the text is about a meeting on January 1, 2025, we do not need a fact that states that.
- Do not use pronouns (he, she, this, that, etc) unless it directly refers to an object in the fact. Facts will likely be viewed out of context and will not make sense otherwise.
- When you detect a new section, give it a descriptive name based on the text.
- Start with section 1. If a section has no theses, arguments or facts, still include the section in the output, just empty
- Continue working on the current section unless you detect a clear section break in the text.
- Output only the sections you are currently working on (current section + any new sections detected).

Format Example:
[
{
  "section": "Section [number]: [title]",
  "theses": [
    {
      "thesis": "...",
      "facts": ["...", "..."],
      "arguments": [
        {"argument": "...", "importance": 8},
        {"argument": "...", "importance": 5}
      ]
    }
  ]
},
{
  "section": "Section [number]: [title]",
  "theses": [
    {
      "thesis": "...",
      "facts": ["...", "..."],
      "arguments": [
        {"argument": "...", "importance": 8},
        {"argument": "...", "importance": 5}
      ]
    }
  ]
}

]`,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userContent,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), LLMRequestTimeout)
		resp, err := executeLLMRequestWithRetry(ctx, c, messages)
		cancel()
		if err != nil {
			return nil, facts, models.Usage{}, err
		}
		if len(resp.Choices) == 0 {
			log.Printf("[ERROR] LLM returned no choices for chunk %d/%d", len(completedSections)+1, len(chunks))
			failedChunks++
			// Save current section data before skipping this chunk to avoid data loss
			if len(currentSectionAnalyses) > 0 {
				completedSections = append(completedSections, currentSectionAnalyses...)
				log.Printf("Saving current section data before skipping chunk with no LLM choices")
				currentSectionAnalyses = nil
				lastSectionName = ""
				cacheValid = false
			}
			continue
		}

		content := cleanContent(resp.Choices[0].Message.Content)
		var analysis []models.SectionAnalysis
		if err := json.Unmarshal([]byte(content), &analysis); err != nil {
			log.Printf("[METRICS] JSON parse error - chunk_index=%d, error=%v", len(completedSections), err)
			log.Printf("content: %v", content)

			// Try to repair the JSON by asking the LLM to fix it
			jsonRepairAttempts++
			repairMessages := []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `You are a JSON repair assistant. Fix the following invalid JSON and return ONLY valid JSON.
The JSON should be an array of section analyses with this structure:
[{
  "section": "Section N: Title",
  "theses": [{
    "thesis": "...",
    "facts": ["...", "..."],
    "arguments": [{"argument": "...", "importance": N}]
  }]
}]`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Fix this invalid JSON:\n\n" + content,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), LLMRequestTimeout)
			repairResp, err := executeLLMRequestWithRetry(ctx, c, repairMessages)
			cancel()
			if err != nil {
				log.Printf("[ERROR] JSON repair request failed - chunk_index=%d, error=%v", len(completedSections), err)
				failedChunks++
				// Save current section data before skipping this chunk to avoid data loss
				if len(currentSectionAnalyses) > 0 {
					completedSections = append(completedSections, currentSectionAnalyses...)
					log.Printf("Saving current section data before skipping chunk due to JSON error")
					currentSectionAnalyses = nil
					lastSectionName = ""
					cacheValid = false
				}
				continue
			}

			if len(repairResp.Choices) == 0 {
				log.Printf("[ERROR] LLM returned no choices for JSON repair - chunk_index=%d", len(completedSections))
				failedChunks++
				if len(currentSectionAnalyses) > 0 {
					completedSections = append(completedSections, currentSectionAnalyses...)
					log.Printf("Saving current section data before skipping chunk due to JSON error")
					currentSectionAnalyses = nil
					lastSectionName = ""
					cacheValid = false
				}
				continue
			}

			repairContent := cleanContent(repairResp.Choices[0].Message.Content)
			if err := json.Unmarshal([]byte(repairContent), &analysis); err != nil {
				log.Printf("[ERROR] Repaired JSON still invalid - chunk_index=%d, error=%v", len(completedSections), err)
				log.Printf("repaired content: %v", repairContent)
				failedChunks++
				// Save current section data before skipping this chunk to avoid data loss
				if len(currentSectionAnalyses) > 0 {
					completedSections = append(completedSections, currentSectionAnalyses...)
					log.Printf("Saving current section data before skipping chunk due to JSON error")
					currentSectionAnalyses = nil
					lastSectionName = ""
					cacheValid = false
				}
				continue
			}
			log.Printf("[METRICS] JSON repair succeeded - chunk_index=%d", len(completedSections))
			jsonRepairSuccesses++
		}
		log.Printf("all analysis %v", analysis)

		// Check for section transitions and manage completed sections
		if len(analysis) > 0 {
			// Process each section in the analysis to properly merge/update currentSectionAnalyses
			for _, section := range analysis {
				sectionTitle := strings.TrimSpace(section.Section)
				if sectionTitle == "" {
					continue
				}

				// Check if this section already exists in currentSectionAnalyses
				existingSectionIndex := -1
				for i, currentSec := range currentSectionAnalyses {
					if strings.TrimSpace(currentSec.Section) == sectionTitle {
						existingSectionIndex = i
						break
					}
				}

				if existingSectionIndex >= 0 {
					// Section exists - update it by merging theses
					existingSection := &currentSectionAnalyses[existingSectionIndex]
					for _, thesis := range section.Theses {
						// Skip empty theses
						thesisText := strings.TrimSpace(thesis.Thesis)
						if thesisText == "" {
							continue
						}
						// Check if this thesis already exists (by text) to avoid duplicates
						thesisExists := false
						for _, existingThesis := range existingSection.Theses {
							if strings.TrimSpace(existingThesis.Thesis) == thesisText {
								thesisExists = true
								break
							}
						}
						if !thesisExists {
							existingSection.Theses = append(existingSection.Theses, thesis)
						}
					}
				} else {
					// New section - check if we need to save the previous section
					if lastSectionName != "" && sectionTitle != lastSectionName && len(currentSectionAnalyses) > 0 {
						// Save the previous completed section
						completedSections = append(completedSections, currentSectionAnalyses...)
						log.Printf("Completed section %s, saving to completed sections", lastSectionName)
						currentSectionAnalyses = nil
					}
					// Add the new section
					analysesWithoutFacts, newFacts := RemoveFactsFromAnalyses([]models.SectionAnalysis{section})
					facts = append(facts, newFacts...)
					if currentSectionAnalyses == nil {
						currentSectionAnalyses = analysesWithoutFacts
					} else {
						currentSectionAnalyses = append(currentSectionAnalyses, analysesWithoutFacts...)
					}
					lastSectionName = sectionTitle
				}
			}
			cacheValid = false // Invalidate cache when currentSectionAnalyses changes
		} else if len(analysis) == 0 && len(currentSectionAnalyses) > 0 {
			// LLM returned empty array - save current section to avoid data loss
			completedSections = append(completedSections, currentSectionAnalyses...)
			log.Printf("LLM returned empty analyses array, saving current section data")
			currentSectionAnalyses = nil
			lastSectionName = ""
			cacheValid = false // Invalidate cache
		}

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
			return nil, facts, models.Usage{}, fmt.Errorf("too many chunks failed: %d/%d (%.1f%%), threshold is %.1f%%",
				failedChunks, totalChunks, failureRate*100, MaxChunkFailureRate*100)
		}
		if failedChunks > 0 {
			log.Printf("[WARNING] Some chunks failed but within threshold - failed=%d/%d (%.1f%%), threshold=%.1f%%",
				failedChunks, totalChunks, failureRate*100, MaxChunkFailureRate*100)
		}
	}

	// Combine completed sections with current working sections for final output
	var allAnalyses []models.SectionAnalysis
	allAnalyses = append(allAnalyses, completedSections...)
	allAnalyses = append(allAnalyses, currentSectionAnalyses...)

	if len(allAnalyses) == 0 {
		log.Printf("[METRICS] ExtractThesesAndArguments failed - no valid analyses returned")
		return nil, facts, models.Usage{}, errors.New("no valid analyses returned")
	}

	// Log completion metrics
	elapsed := time.Since(start)
	promptCost := float64(totalPromptTokens) / 1_000_000 * PromptCostPerMillion
	completionCost := float64(totalCompletionTokens) / 1_000_000 * CompletionCostPerMillion
	totalCost := promptCost + completionCost

	log.Printf("[METRICS] ExtractThesesAndArguments completed - "+
		"duration=%v, sections=%d, theses=%d, facts=%d, "+
		"prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, cost=$%.4f, "+
		"json_repairs=%d/%d, failed_chunks=%d/%d, model=%s",
		elapsed, len(allAnalyses), countTheses(allAnalyses), len(facts),
		totalPromptTokens, totalCompletionTokens, totalPromptTokens+totalCompletionTokens, totalCost,
		jsonRepairSuccesses, jsonRepairAttempts, failedChunks, totalChunks, c.Model)

	return allAnalyses, facts, models.Usage{
		PromptTokens:     totalPromptTokens,
		CompletionTokens: totalCompletionTokens,
		TotalTokens:      totalPromptTokens + totalCompletionTokens,
	}, nil
}

// countTheses counts the total number of theses across all analyses
func countTheses(analyses []models.SectionAnalysis) int {
	total := 0
	for _, analysis := range analyses {
		total += len(analysis.Theses)
	}
	return total
}

// Clean possible markdown wrappers
func cleanContent(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return content
}

// buildContextIntro creates a cached JSON representation of current analyses
// Returns the context string and updated cache state
func buildContextIntro(currentSectionAnalyses []models.SectionAnalysis, cachedSectionJSON string, cacheValid bool) (string, string, bool) {
	if len(currentSectionAnalyses) == 0 {
		return "", cachedSectionJSON, cacheValid
	}
	if !cacheValid {
		existingAnalysesJSON, err := json.Marshal(currentSectionAnalyses)
		if err == nil {
			cachedSectionJSON = "<existing_analyses>\n" + string(existingAnalysesJSON) + "\n</existing_analyses>\n"
			cacheValid = true
		}
	}
	return cachedSectionJSON, cachedSectionJSON, cacheValid
}

// buildUserContent constructs the user content for LLM extraction
func buildUserContent(chunk, contextIntro, lastSectionName string) string {
	sectionHint := lastSectionName
	if sectionHint == "" {
		sectionHint = "1: Introduction"
	}
	return contextIntro +
		fmt.Sprintf("The last analyzed chunk ended in Section %s.\n", sectionHint) +
		"Now analyze the following text. " +
		"If you believe the author has started a new section (e.g., with a title), create a new section with a descriptive name (e.g., \"Section 2: [New Section Title]\"). " +
		"Otherwise, continue assigning output under the previous section. " +
		"Always include \"section\" explicitly in your JSON output.\n\n<CHUNK>\n\n" + chunk
}

// buildExtractionMessages creates the system and user messages for thesis/argument extraction
func buildExtractionMessages(userContent string) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are an assistant that extracts theses, facts, and arguments from text.
				We are trying to come up with a coherent summary of the article/podcast/book/etc. You will be looking at
				some or all of the writing and need to extract certain things from it.
				Inside the <existing_analyses> block, you will find the current section being analyzed.
				Use this to understand the context and continue building on the current section.

Instructions:
- Respond ONLY in pure JSON with the following format.
- Do not add commentary, explanations, or non‑JSON text.
- If an item cannot be extracted, return an empty string or empty list.
- Importance must be an integer on a scale of 1–10 (10 = crucial to the central thesis, 1 = marginal).
- Facts should be discrete, verifiable statements (events, statistics, claims of evidence).
- Facts should *not* be about the text itself. For example, if the text is about a meeting on January 1, 2025, we do not need a fact that states that.
- Do not use pronouns (he, she, this, that, etc) unless it directly refers to an object in the fact. Facts will likely be viewed out of context and will not make sense otherwise.
- When you detect a new section, give it a descriptive name based on the text.
- Start with section 1. If a section has no theses, arguments or facts, still include the section in the output, just empty
- Continue working on the current section unless you detect a clear section break in the text.
- Output only the sections you are currently working on (current section + any new sections detected).

Format Example:
[
{
  "section": "Section [number]: [title]",
  "theses": [
    {
      "thesis": "...",
      "facts": ["...", "..."],
      "arguments": [
        {"argument": "...", "importance": 8},
        {"argument": "...", "importance": 5}
      ]
    }
  ]
},
{
  "section": "Section [number]: [title]",
  "theses": [
    {
      "thesis": "...",
      "facts": ["...", "..."],
      "arguments": [
        {"argument": "...", "importance": 8},
        {"argument": "...", "importance": 5}
      ]
    }
  ]

}

]`,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userContent,
		},
	}
}

// mergeSectionAnalyses merges new analysis results into current section analyses,
// handling section transitions and returning updated facts
func mergeSectionAnalyses(
	analysis []models.SectionAnalysis,
	currentSectionAnalyses []models.SectionAnalysis,
	lastSectionName string,
	facts []string,
) ([]models.SectionAnalysis, string, []string) {
	if len(analysis) == 0 {
		if len(currentSectionAnalyses) > 0 {
			// Save current section to avoid data loss when LLM returns empty array
			return nil, "", append(facts, make([]string, 0)...) // Return nil to signal save occurred
		}
		return currentSectionAnalyses, lastSectionName, facts
	}

	for _, section := range analysis {
		sectionTitle := strings.TrimSpace(section.Section)
		if sectionTitle == "" {
			continue
		}

		// Check if this section already exists in currentSectionAnalyses
		existingSectionIndex := -1
		for i, currentSec := range currentSectionAnalyses {
			if strings.TrimSpace(currentSec.Section) == sectionTitle {
				existingSectionIndex = i
				break
			}
		}

		if existingSectionIndex >= 0 {
			// Section exists - merge theses
			existingSection := &currentSectionAnalyses[existingSectionIndex]
			for _, thesis := range section.Theses {
				thesisText := strings.TrimSpace(thesis.Thesis)
				if thesisText == "" {
					continue
				}
				// Check for duplicates
				thesisExists := false
				for _, existingThesis := range existingSection.Theses {
					if strings.TrimSpace(existingThesis.Thesis) == thesisText {
						thesisExists = true
						break
					}
				}
				if !thesisExists {
					existingSection.Theses = append(existingSection.Theses, thesis)
				}
			}
		} else {
			// New section - save previous if needed
			if lastSectionName != "" && sectionTitle != lastSectionName && len(currentSectionAnalyses) > 0 {
				// Signal caller to save completed sections
				return currentSectionAnalyses, lastSectionName, facts
			}
			// Add new section
			analysesWithoutFacts, newFacts := RemoveFactsFromAnalyses([]models.SectionAnalysis{section})
			facts = append(facts, newFacts...)
			if currentSectionAnalyses == nil {
				currentSectionAnalyses = analysesWithoutFacts
			} else {
				currentSectionAnalyses = append(currentSectionAnalyses, analysesWithoutFacts...)
			}
			lastSectionName = sectionTitle
		}
	}

	return currentSectionAnalyses, lastSectionName, facts
}

// AnalyzeAndSummarizeText: the advanced pipeline
func AnalyzeAndSummarizeText(c *models.LLMClient, allAnalyses []models.SectionAnalysis, facts []string, usage models.Usage) (string, []models.SectionAnalysis, models.Usage, error) {
	start := time.Now()
	c.Model = os.Getenv(EnvSummarizeModel)
	if c.Model == "" {
		c.Model = DefaultSummarizeModel
	}

	// Log input metrics
	log.Printf("[METRICS] AnalyzeAndSummarizeText started - sections=%d, theses=%d, facts=%d, model=%s",
		len(allAnalyses), countTheses(allAnalyses), len(facts), c.Model)

	// Start with existing usage counts and accumulate
	totalPromptTokens := usage.PromptTokens
	totalCompletionTokens := usage.CompletionTokens

	// Aggregate all results into one string
	theses := []string{}

	for _, sec := range allAnalyses {
		for _, th := range sec.Theses {
			if th.Thesis != "" {
				theses = append(theses, th.Thesis)
			}
		}
	}

	// Deduplicate and rank with another LLM call
	// Helper function to format arguments with their importance values
	formatArguments := func(args []models.Argument) string {
		var out []string
		for _, a := range args {
			out = append(out, fmt.Sprintf("(importance %d) %s", a.Importance, a.Argument))
		}
		return strings.Join(out, "\n- ")
	}
	dedupInput := "Theses: " + strings.Join(theses, "; ") +
		"\nFacts: " + strings.Join(facts, "; ") +
		"\nCollected Arguments (with importance):\n- " + formatArguments(flattenArguments(allAnalyses))

	dedupMessages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are an assistant that deduplicates and ranks extracted information.
Respond ONLY in JSON with the following format:
{
  "theses": [{"thesis": "...", "rank": 1}, {"thesis": "...", "rank": 2}],
  "facts": ["...", "..."],
  "arguments": [{"argument": "...", "rank": 1}, {"argument": "...", "rank": 2}]
}
Important: Consider the full set of arguments with their importance values when performing deduplication and ranking.`,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: dedupInput,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), LLMRequestTimeout)
	dedupResp, err := executeLLMRequestWithRetry(ctx, c, dedupMessages)
	cancel()
	if err != nil {
		return "", nil, models.Usage{}, err
	}
	if len(dedupResp.Choices) == 0 {
		return "", nil, models.Usage{}, errors.New("no deduplicated results returned")
	}
	dedupContent := strings.TrimSpace(dedupResp.Choices[0].Message.Content)
	dedupContent = strings.TrimPrefix(dedupContent, "```json")
	dedupContent = strings.TrimPrefix(dedupContent, "```")
	dedupContent = strings.TrimSuffix(dedupContent, "```")
	aggregation := dedupContent
	totalPromptTokens += dedupResp.Usage.PromptTokens
	totalCompletionTokens += dedupResp.Usage.CompletionTokens

	// Final summarization
	finalMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are an assistant that summarizes text clearly and concisely.",
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: `
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
     - **Key Evidence or Facts** (5–8 of the most decisive data points, milestones, or examples).  
   - Present information in a hierarchy (for each theses, show its supporting arguments → facts).  
   - Exclude secondary/tangential details.

4. **General Guidelines:**  
   - Focus only on what is **strategically or academically important** to understand the subject.  
   - Omit extraneous digressions, trivia, or minor historical detail.  
   - Keep each section readable on its own.  
   - Do not return anything other than the details, no pleasantries!
   - Tone:  
     - Section 1 → plain, polished, and accessible ("boardroom-ready").  
     - Section 2 → objective, precise, and reference-style ("briefing document").  

Input (including deduplicated theses, facts, and arguments with importance/rank):
<analysis>\n` + aggregation,
		},
	}

	var finalResp openai.ChatCompletionResponse
	ctx, cancel = context.WithTimeout(context.Background(), LLMRequestTimeout)
	finalResp, err = executeLLMRequestWithRetry(ctx, c, finalMessages)
	cancel()
	if err != nil {
		return "", nil, models.Usage{}, err
	}
	if len(finalResp.Choices) == 0 {
		return "", nil, models.Usage{}, errors.New("no summary returned")
	}
	totalPromptTokens += finalResp.Usage.PromptTokens
	totalCompletionTokens += finalResp.Usage.CompletionTokens

	summary := finalResp.Choices[0].Message.Content

	promptCost := float64(totalPromptTokens) / 1_000_000 * PromptCostPerMillion
	completionCost := float64(totalCompletionTokens) / 1_000_000 * CompletionCostPerMillion
	totalCost := promptCost + completionCost

	// Log completion metrics with structured format
	elapsed := time.Since(start)
	log.Printf("[METRICS] AnalyzeAndSummarizeText completed - "+
		"duration=%v, summary_length=%d chars, sections=%d, theses=%d, facts=%d, "+
		"prompt_tokens=%d, completion_tokens=%d, total_tokens=%d, cost=$%.4f, model=%s",
		elapsed, len(summary), len(allAnalyses), countTheses(allAnalyses), len(facts),
		totalPromptTokens, totalCompletionTokens, totalPromptTokens+totalCompletionTokens, totalCost, c.Model)

	// Return new Usage struct with accumulated totals (don't mutate input parameter)
	resultUsage := models.Usage{
		PromptTokens:     totalPromptTokens,
		CompletionTokens: totalCompletionTokens,
		TotalTokens:      totalPromptTokens + totalCompletionTokens,
		TotalCost:        totalCost,
	}

	return summary, allAnalyses, resultUsage, nil
}

// flattenArguments combines arguments from multiple analyses
func flattenArguments(analyses []models.SectionAnalysis) []models.Argument {
	var args []models.Argument
	for _, sec := range analyses {
		for _, th := range sec.Theses {
			args = append(args, th.Arguments...)
		}
	}
	return args
}

// GetCardAnalysis reconstructs the analysis data structure from the database for a given card.
// It fetches the most recent summarization for the card.
func GetCardAnalysis(db *sql.DB, userID int, cardPK int) ([]models.SectionAnalysis, error) {
	// Find the most recent summarization ID for the card
	var summarizationID int
	err := db.QueryRow(`
		SELECT id FROM summarizations
		WHERE user_id = $1 AND card_pk = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, cardPK).Scan(&summarizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find summarization for card: %w", err)
	}

	log.Printf("getting %v", summarizationID)
	// Fetch sections
	sectionRows, err := db.Query(`
		SELECT id, section_title FROM summary_sections
		WHERE user_id = $1 AND summarization_id = $2
		ORDER BY COALESCE(section_order, 0), id
	`, userID, summarizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sections: %w", err)
	}
	defer sectionRows.Close()

	var analyses []models.SectionAnalysis
	for sectionRows.Next() {
		var sectionID int
		var section models.SectionAnalysis
		if err := sectionRows.Scan(&sectionID, &section.Section); err != nil {
			return nil, fmt.Errorf("failed to scan section: %w", err)
		}

		// Fetch theses for the current section
		thesisRows, err := db.Query(`
			SELECT id, thesis FROM summary_theses
			WHERE user_id = $1 AND section_id = $2
			ORDER BY id
		`, userID, sectionID)
		if err != nil {
			return nil, fmt.Errorf("failed to query theses for section %d: %w", sectionID, err)
		}

		var theses []models.ThesisEntry
		for thesisRows.Next() {
			var thesisID int
			var thesis models.ThesisEntry
			if err := thesisRows.Scan(&thesisID, &thesis.Thesis); err != nil {
				thesisRows.Close()
				return nil, fmt.Errorf("failed to scan thesis: %w", err)
			}

			// Fetch arguments for the current thesis
			argRows, err := db.Query(`
				SELECT argument, importance FROM summary_arguments
				WHERE user_id = $1 AND thesis_id = $2
				ORDER BY id
			`, userID, thesisID)
			if err != nil {
				thesisRows.Close()
				return nil, fmt.Errorf("failed to query arguments for thesis %d: %w", thesisID, err)
			}

			var arguments []models.Argument
			for argRows.Next() {
				var arg models.Argument
				if err := argRows.Scan(&arg.Argument, &arg.Importance); err != nil {
					argRows.Close()
					thesisRows.Close()
					return nil, fmt.Errorf("failed to scan argument: %w", err)
				}
				arguments = append(arguments, arg)
			}
			// Explicitly close argRows after use to avoid resource leaks
			if err := argRows.Close(); err != nil {
				thesisRows.Close()
				return nil, fmt.Errorf("failed to close argument rows: %w", err)
			}
			thesis.Arguments = arguments
			theses = append(theses, thesis)
		}
		// Explicitly close thesisRows after use to avoid resource leaks
		if err := thesisRows.Close(); err != nil {
			return nil, fmt.Errorf("failed to close thesis rows: %w", err)
		}
		section.Theses = theses
		analyses = append(analyses, section)
	}

	return analyses, nil
}

// RemoveFactsFromAnalyses creates a copy of the analyses structure without facts
// to reduce context size when feeding back to the LLM
func RemoveFactsFromAnalyses(analyses []models.SectionAnalysis) ([]models.SectionAnalysis, []string) {
	// Pre-allocate result slice to avoid multiple reallocations
	result := make([]models.SectionAnalysis, 0, len(analyses))
	facts := make([]string, 0, len(analyses)*3) // Pre-allocate facts with estimated capacity

	for _, section := range analyses {
		// Pre-allocate theses slice to avoid multiple reallocations
		newSection := models.SectionAnalysis{
			Section: section.Section,
			Theses:  make([]models.ThesisEntry, len(section.Theses)),
		}

		for i, thesis := range section.Theses {
			facts = append(facts, thesis.Facts...)
			// Reuse the Arguments slice reference - no need to copy the underlying data
			newSection.Theses[i] = models.ThesisEntry{
				Thesis:    thesis.Thesis,
				Facts:     []string{}, // Remove facts to reduce token count
				Arguments: thesis.Arguments,
			}
		}

		result = append(result, newSection)
	}

	return result, facts
}
