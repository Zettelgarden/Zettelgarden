package services

import (
	"os"
	"strconv"
	"strings"

	"github.com/neurosnap/sentences"
)

// ChunkerConfig holds configuration for text chunking behavior
type ChunkerConfig struct {
	// MaxChunkSize is the maximum size (in characters) for each chunk
	MaxChunkSize int

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
		MaxChunkSize: maxSize,
		MinChunkSize: maxSize / 4, // Minimum 25% of max size
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

// splitIntoSentences splits text into sentences using proper sentence boundary detection.
// This handles abbreviations (Dr., Mr., etc.), decimals (3.14), ellipsis (...), and other edge cases.
func (c *Chunker) splitIntoSentences(input string) []string {
	// Create the sentence tokenizer with default English training data
	storage := sentences.NewStorage()
	tokenizer := sentences.NewSentenceTokenizer(storage)

	// Tokenize the input into sentences
	sentenceTokens := tokenizer.Tokenize(input)

	// Extract the text from each sentence
	result := make([]string, len(sentenceTokens))
	for i, s := range sentenceTokens {
		result[i] = s.Text
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
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return result
}
