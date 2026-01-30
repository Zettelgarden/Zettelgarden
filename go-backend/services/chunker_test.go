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
		MaxChunkSize: 50,
		MinChunkSize: 10,
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
		MaxChunkSize: 100,
		MinChunkSize: 20,
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
		MaxChunkSize: 100,
		MinChunkSize: 30,
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
}

// TestChunker_Abbreviations tests that abbreviations are not treated as sentence ends
func TestChunker_Abbreviations(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	input := "Dr. Smith went to Washington. Mr. Johnson arrived later. We visited the U.S. Capitol. " +
		"The cost was $5.50. The study used i.e. and e.g. correctly. " +
		"The time was 9:30 a.m. and we ended at 5:00 p.m. The company is Inc. and Corp. works too."

	result := chunker.Chunk(input)

	// With proper sentence boundary detection, abbreviations should not split sentences
	// The text should be returned as a single chunk since it's under MaxChunkSize
	if len(result) != 1 {
		t.Errorf("Expected single chunk for abbreviation test, got %d chunks", len(result))
	}

	// Verify the full text is preserved (abbreviations shouldn't break the text incorrectly)
	if result[0] != input {
		t.Errorf("Text not preserved correctly with abbreviations.\nExpected: %q\nGot: %q", input, result[0])
	}
}

// TestChunker_Decimals tests that decimal numbers are not treated as sentence ends
func TestChunker_Decimals(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	input := "The value of pi is approximately 3.14. The stock rose by 2.5%. The temperature was 98.6 degrees. " +
		"The price was $19.99. We measured 12.345 inches."

	result := chunker.Chunk(input)

	// Decimals should not split the text into separate sentences
	if len(result) != 1 {
		t.Errorf("Expected single chunk for decimal test, got %d chunks", len(result))
	}

	if result[0] != input {
		t.Errorf("Text not preserved correctly with decimals.\nExpected: %q\nGot: %q", input, result[0])
	}
}

// TestChunker_Ellipsis tests that ellipsis are handled correctly
func TestChunker_Ellipsis(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	input := "The story continued... and then ended. Another example here... which continues properly. " +
		"Waiting... still waiting... done!"

	result := chunker.Chunk(input)

	// Ellipsis should not cause incorrect splitting
	if len(result) != 1 {
		t.Errorf("Expected single chunk for ellipsis test, got %d chunks", len(result))
	}

	if result[0] != input {
		t.Errorf("Text not preserved correctly with ellipsis.\nExpected: %q\nGot: %q", input, result[0])
	}
}

// TestChunker_URLs tests that URLs with dots are handled correctly
func TestChunker_URLs(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	input := "Visit https://example.com for more info. Also check https://sub.domain.co.uk. " +
		"The site at www.test.org is great. See docs.example.com/path/to/page.html for details."

	result := chunker.Chunk(input)

	// URLs should not cause incorrect splitting
	if len(result) != 1 {
		t.Errorf("Expected single chunk for URL test, got %d chunks", len(result))
	}

	if result[0] != input {
		t.Errorf("Text not preserved correctly with URLs.\nExpected: %q\nGot: %q", input, result[0])
	}
}

// TestChunker_CombinedEdgeCases tests a combination of edge cases
func TestChunker_CombinedEdgeCases(t *testing.T) {
	config := ChunkerConfig{
		MaxChunkSize: 500,
		MinChunkSize: 50,
	}
	chunker := NewChunker(config)

	input := "Dr. Alice visited the U.S. and saw Washington D.C. at 9:30 a.m. " +
		"She calculated pi as 3.14 and the cost as $99.99. " +
		"Visit https://example.com for details... more info available. " +
		"The study used i.e. and e.g. properly. At 5:00 p.m. she returned."

	result := chunker.Chunk(input)

	// Should be single chunk since it's under 500 chars
	if len(result) != 1 {
		t.Errorf("Expected single chunk, got %d chunks: %v", len(result), result)
	}

	if result[0] != input {
		t.Errorf("Combined edge cases not handled correctly.\nExpected: %q\nGot: %q", input, result[0])
	}
}
