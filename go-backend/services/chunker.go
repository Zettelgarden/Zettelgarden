package services

import (
	"os"
	"strings"
	"unicode"
)

// ChunkerConfig holds configuration for text chunking behavior
type ChunkerConfig struct {
	// MaxChunkSize is the maximum size (in characters) for each chunk
	MaxChunkSize int

	// SentenceDelimiters are the characters that mark sentence boundaries
	SentenceDelimiters []string

	// MinChunkSize is the minimum size for a chunk (prevents tiny chunks)
	MinChunkSize int
}

// DefaultChunkerConfig returns the default configuration for chunking
func DefaultChunkerConfig() ChunkerConfig {
	maxSize := MaxChunkSize // Default from summarize.go
	if env := os.Getenv("ZETTEL_CHUNK_SIZE"); env != "" {
		if size := parseInt(env); size > 0 {
			maxSize = size
		}
	}

	return ChunkerConfig{
		MaxChunkSize:      maxSize,
		SentenceDelimiters: []string{". ", "! ", "? ", ".\n", "!\n", "?\n"},
		MinChunkSize:      maxSize / 4, // Minimum 25% of max size
	}
}

// Chunker handles text chunking with configurable behavior
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new Chunker with the given configuration
func NewChunker(config ChunkerConfig) *Chunker {
	return &Chunker{config: config}
}

// NewDefaultChunker creates a new Chunker with default configuration
func NewDefaultChunker() *Chunker {
	return &Chunker{config: DefaultChunkerConfig()}
}

// Chunk splits input into segments of approximately MaxChunkSize characters,
// breaking at sentence boundaries when possible.
func (c *Chunker) Chunk(input string) []string {
	if len(input) <= c.config.MaxChunkSize {
		return []string{input}
	}

	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0

	// Split into sentences using multiple delimiters
	sentences := c.splitIntoSentences(input)

	for i, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		sentenceSize := len(sentence) + 1 // +1 for space/separator

		// If adding this sentence would exceed max size
		if currentSize > 0 && currentSize+sentenceSize > c.config.MaxChunkSize {
			// Only start a new chunk if current one is large enough
			if currentSize >= c.config.MinChunkSize {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
				currentSize = 0
			}
		}

		if currentSize > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		currentSize += sentenceSize

		// Handle last sentence - always add it
		if i == len(sentences)-1 && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
		}
	}

	return chunks
}

// splitIntoSentences splits text into sentences using configured delimiters
func (c *Chunker) splitIntoSentences(input string) []string {
	text := input
	var result []string

	for {
		// Find the earliest delimiter
		earliestPos := -1
		usedDelimiter := ""

		for _, delimiter := range c.config.SentenceDelimiters {
			if pos := strings.Index(text, delimiter); pos != -1 {
				if earliestPos == -1 || pos < earliestPos {
					earliestPos = pos
					usedDelimiter = delimiter
				}
			}
		}

		if earliestPos == -1 {
			// No more delimiters found
			if text != "" {
				result = append(result, text)
			}
			break
		}

		// Extract the sentence up to and including the delimiter
		sentenceEnd := earliestPos + len(usedDelimiter)
		sentence := text[:sentenceEnd]
		result = append(result, sentence)

		// Move to next sentence
		text = text[sentenceEnd:]
	}

	return result
}

// ChunkText is a convenience function that uses the default chunker
func ChunkText(input string) []string {
	chunker := NewDefaultChunker()
	return chunker.Chunk(input)
}

// parseInt safely parses an integer from a string, returning 0 on error
func parseInt(s string) int {
	var result int
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return 0
		}
		result = result*10 + int(r-'0')
	}
	return result
}
