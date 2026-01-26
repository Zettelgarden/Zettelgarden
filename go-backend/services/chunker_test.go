package services

import (
	"strings"
	"testing"
)

// TestChunker_EmptyInput tests chunking with empty input
func TestChunker_EmptyInput(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())
	result := chunker.Chunk("")
	if len(result) != 1 || result[0] != "" {
		t.Errorf("Expected [\"\"], got %v", result)
	}
}

// TestChunker_ShortInput tests chunking with input shorter than max size
func TestChunker_ShortInput(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())
	input := "This is a short text."
	result := chunker.Chunk(input)
	if len(result) != 1 || result[0] != input {
		t.Errorf("Expected [%q], got %v", input, result)
	}
}

// TestChunker_LongInput tests chunking with input longer than max size
func TestChunker_LongInput(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())
	longInput := "This is sentence one. This is sentence two. This is sentence three. " +
		"This is sentence four. This is sentence five. This is sentence six. " +
		"This is sentence seven. This is sentence eight. This is sentence nine. " +
		"This is sentence ten. This is sentence eleven. This is sentence twelve."

	result := chunker.Chunk(longInput)

	if len(result) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	// Verify all chunks are within size limits
	for i, chunk := range result {
		if len(chunk) > chunker.config.MaxChunkSize {
			t.Errorf("Chunk %d exceeds max size: %d > %d", i, len(chunk), chunker.config.MaxChunkSize)
		}
	}

	// Verify total content is preserved
	reconstructed := ""
	for _, chunk := range result {
		reconstructed += chunk
		if chunk != result[len(result)-1] {
			reconstructed += " "
		}
	}
	if reconstructed != longInput {
		t.Errorf("Content not preserved.\nExpected: %q\nGot: %q", longInput, reconstructed)
	}
}

// TestChunker_CustomConfig tests chunking with custom configuration
func TestChunker_CustomConfig(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkSize:      50,
		SentenceDelimiters: []string{". ", "! ", "? "},
		MinChunkSize:      10,
	}
	chunker := NewChunker(config)
	input := "This is sentence one! This is sentence two. This is sentence three? This is sentence four."

	result := chunker.Chunk(input)

	if len(result) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	// Verify all chunks are within size limits
	for i, chunk := range result {
		if len(chunk) > config.MaxChunkSize {
			t.Errorf("Chunk %d exceeds max size: %d > %d", i, len(chunk), config.MaxChunkSize)
		}
	}
}

// TestChunker_MultipleDelimiters tests chunking with different sentence terminators
func TestChunker_MultipleDelimiters(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkSize:      100,
		SentenceDelimiters: []string{". ", "! ", "? ", ".\n", "!\n"},
		MinChunkSize:      20,
	}
	chunker := NewChunker(config)
	input := "First sentence here! Second sentence here. Third sentence? Fourth sentence.\nFifth sentence!"

	result := chunker.Chunk(input)

	if len(result) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	// Verify sentences are split correctly
	hasFirst := false
	hasSecond := false
	hasThird := false
	hasFourth := false
	hasFifth := false

	for _, chunk := range result {
		if strings.Contains(chunk, "First sentence here") {
			hasFirst = true
		}
		if strings.Contains(chunk, "Second sentence here") {
			hasSecond = true
		}
		if strings.Contains(chunk, "Third sentence") {
			hasThird = true
		}
		if strings.Contains(chunk, "Fourth sentence") {
			hasFourth = true
		}
		if strings.Contains(chunk, "Fifth sentence") {
			hasFifth = true
		}
	}

	if !hasFirst || !hasSecond || !hasThird || !hasFourth || !hasFifth {
		t.Errorf("Not all sentences preserved in chunks: %v", result)
	}
}

// TestChunker_MinChunkSize tests that chunks meet minimum size requirements
func TestChunker_MinChunkSize(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkSize:      100,
		SentenceDelimiters: []string{". "},
		MinChunkSize:      30,
	}
	chunker := NewChunker(config)
	input := "Short. A bit longer sentence here. Another medium length sentence. Final sentence here."

	result := chunker.Chunk(input)

	// Verify most chunks meet minimum size (except possibly the last one)
	for i, chunk := range result {
		if i < len(result)-1 && len(chunk) < config.MinChunkSize {
			t.Errorf("Chunk %d is below min size: %d < %d", i, len(chunk), config.MinChunkSize)
		}
	}
}

// TestChunkText_ConvenienceFunction tests the ChunkText convenience function
func TestChunkText_ConvenienceFunction(t *testing.T) {
	input := "This is a test. This is only a test."
	result := ChunkText(input)

	if len(result) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	if len(result) == 1 && result[0] != input {
		t.Errorf("Expected %q, got %q", input, result[0])
	}
}

// TestChunker_PreservesContent verifies that chunking preserves all content
func TestChunker_PreservesContent(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())
	input := "First. Second. Third. Fourth. Fifth."

	result := chunker.Chunk(input)

	reconstructed := ""
	for i, chunk := range result {
		reconstructed += chunk
		if i < len(result)-1 {
			reconstructed += " "
		}
	}

	if reconstructed != input {
		t.Errorf("Content not preserved.\nOriginal:  %q\nReconstructed: %q", input, reconstructed)
	}
}

// TestDefaultChunkerConfig tests the default configuration
func TestDefaultChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()

	if config.MaxChunkSize != MaxChunkSize {
		t.Errorf("Expected MaxChunkSize %d, got %d", MaxChunkSize, config.MaxChunkSize)
	}

	if config.MinChunkSize != MaxChunkSize/4 {
		t.Errorf("Expected MinChunkSize %d, got %d", MaxChunkSize/4, config.MinChunkSize)
	}

	if len(config.SentenceDelimiters) == 0 {
		t.Error("Expected at least one sentence delimiter")
	}
}
