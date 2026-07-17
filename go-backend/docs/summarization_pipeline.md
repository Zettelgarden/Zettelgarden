# Summarization Pipeline

This document describes the summarization and analysis pipeline for Zettelgarden cards.

## Overview

When a card is created or updated, the system can asynchronously analyze its content to extract:
- **Theses**: Main claims or arguments presented in the text
- **Arguments**: Supporting points for each thesis with importance ratings (1-10)
- **Facts**: Verifiable statements, statistics, or evidence mentioned
- **Summary**: A concise two-part summary (Executive Summary + Reference Summary)

## Architecture

The pipeline consists of two main flows:

### 1. Entity/Fact Extraction Flow (`ProcessEntitiesAndFacts`)

Triggered automatically when a card is created (unless in testing mode).

```
Card Creation
    ↓
ProcessEntitiesAndFacts (handlers/summarize.go:188)
    ↓
Create summarization job in DB (status: pending)
    ↓
Run in goroutine with defer for LinkCardToEntityIfPossible
    ↓
ExtractThesesAndArguments (services/summarize.go:36)
    ↓
SaveAnalysis (handlers/summarize.go:248)
    ↓
ExtractSaveCardFacts (handlers/entity.go)
    ↓
ExtractSaveFactEntities (handlers/entity.go)
    ↓
LinkCardToEntityIfPossible (handlers/entity.go)
```

### 2. On-Demand Summarization Flow (`CreateSummarizationRoute`)

Triggered by user request via API.

```
POST /api/summarize
    ↓
CreateSummarizationRoute (handlers/summarize.go:140)
    ↓
Create summarization job in DB (status: pending)
    ↓
ExtractThesesAndArguments
    ↓
runSummarizationJob (handlers/summarize.go:334)
    ↓
AnalyzeAndSummarizeText (services/summarize.go:258)
    ↓
Update job status to 'complete'
```

## Detailed Flow

### Step 1: Text Preparation (`prepareTextForAnalysis`)

**Location**: `handlers/summarize.go:38`

1. Combines title and body into markdown format
2. Removes card reference lines (e.g., `[1/A.1] - Card Title`)
3. Cleans up extra whitespace

Example:
```go
// Input: title="Meeting Notes", body="Discussed budget\n[REF-001] - Previous Meeting"
// Output: "# Meeting Notes\n\nDiscussed budget"
```

### Step 2: Chunking (`chunkText`)

**Location**: `services/summarize.go:402`

Text is split into chunks of `MaxChunkSize` (15000 characters) at sentence boundaries to avoid breaking mid-thought.

### Step 3: Thesis/Argument Extraction (`ExtractThesesAndArguments`)

**Location**: `services/summarize.go:36`

For each chunk:
1. **Build Context**: Marshals current section analyses to JSON for continuity
2. **Build Messages**: Creates system prompt + user content with current chunk
3. **Execute LLM**: Calls OpenAI-compatible API
4. **Parse Response**: Unmarshals JSON into `SectionAnalysis` structures
5. **Merge Results**: Combines with existing analyses, handling section transitions

**Data Structure**:
```go
type SectionAnalysis struct {
    Section string         // e.g., "Section 1: Introduction"
    Theses  []ThesisEntry
}

type ThesisEntry struct {
    Thesis    string      // Main claim
    Facts     []string    // Verifiable facts (extracted separately)
    Arguments []Argument  // Supporting points
}

type Argument struct {
    Argument    string // Supporting point text
    Importance  int    // 1-10 scale
}
```

**Section Transition Logic**:
- When LLM detects a new section (e.g., "Section 2: Analysis"), the previous section is saved to `completedSections`
- Current working section (`currentSectionAnalyses`) is cleared
- New section is added
- This allows progressive building of a complete document analysis

### Step 4: Save Analysis (`SaveAnalysis`)

**Location**: `handlers/summarize.go:248`

Persists the structured analysis to relational tables:

```
summarizations (job metadata)
    ↓
summary_sections (sections per job)
    ↓
summary_theses (theses per section)
    ↓
summary_arguments (arguments per thesis)
```

**Validation**:
- `cardPK` must be positive (> 0)
- Empty/whitespace section titles are skipped
- Empty/whitespace theses are skipped
- Transactional: all or nothing

### Step 5: Fact Extraction (`ExtractSaveCardFacts`)

**Location**: `handlers/entity.go`

Extracts facts separately from theses (facts are embedded in LLM response but stored separately):
- Creates `fact` records linked to card
- Stores fact text for later retrieval

### Step 6: Entity Extraction (`ExtractSaveFactEntities`)

**Location**: `handlers/entity.go`

Extracts named entities from facts:
- People, organizations, locations, etc.
- Creates `entity` records if needed
- Links facts to entities

### Step 7: Entity Linking (`LinkCardToEntityIfPossible`)

**Location**: `handlers/entity.go:842`

**Guaranteed execution** via `defer` in goroutine:
- Runs even if errors occur in fact extraction
- Skipped during testing (checks `h.Server.Testing`)
- Analyzes card content for entity mentions
- Creates card-entity relationships

### Step 8: Final Summarization (`AnalyzeAndSummarizeText`)

**Location**: `services/summarize.go:258`

Creates the final markdown summary:

1. **Aggregate**: Collects all theses, facts, and arguments
2. **Deduplicate**: LLM call to remove duplicates and rank importance
3. **Summarize**: LLM call to generate two-part summary:
   - **Executive Summary**: 4-6 bullet points, strategic, for decision-makers
   - **Reference Summary**: Detailed, organized by theses with supporting arguments and facts

**Cost Calculation**:
- Prompt: `$1.25 per million tokens`
- Completion: `$10.00 per million tokens`
- Logged for observability, NOT included in result

## Constants and Configuration

**Environment Variables**:
- `ZETTEL_LLM_SUMMARIZE_MODEL`: Model for final summarization (default: "glm-5.2")

**Constants** (`services/summarize.go:18`):
- `MaxChunkSize`: 15000 characters
- `DefaultSummarizeModel`: "glm-5.2"
- `EnvSummarizeModel`: "ZETTEL_LLM_SUMMARIZE_MODEL"
- `PromptCostPerMillion`: 1.25
- `CompletionCostPerMillion`: 10.0

## Database Schema

```
summarizations
├── id (PK)
├── user_id (FK)
├── card_pk (FK, nullable)
├── input_text
├── status (pending|processing|complete|failed)
├── result (final markdown summary)
├── prompt_tokens
├── completion_tokens
├── total_tokens
├── cost
├── model
└── created_at, updated_at

summary_sections
├── id (PK)
├── summarization_id (FK)
├── section_title
└── section_order

summary_theses
├── id (PK)
├── section_id (FK)
└── thesis

summary_arguments
├── id (PK)
├── thesis_id (FK)
├── argument
└── importance (1-10)
```

## Testing

**Test Files**:
- `handlers/summarize_test.go`: Tests for SaveAnalysis, removeReferences, prepareTextForAnalysis
- `services/summarize_test.go`: Tests for RemoveFactsFromAnalyses

**Test Utilities**:
- `setup()`: Creates test handler with database
- `tests.Teardown()`: Cleans up test database

## Error Handling

**JSON Parsing Errors**:
- Logged with content preview
- Current sections saved to prevent data loss
- Processing continues with next chunk

**Empty LLM Response**:
- Current sections saved to prevent data loss
- Cache invalidated

**LLM Request Errors**:
- Job status set to 'failed' with error message
- Processing stops

## Performance Considerations

1. **Chunking**: Large texts processed in chunks to fit LLM context windows
2. **Caching**: Marshaled section analyses cached to avoid re-marshaling on each chunk
3. **Goroutines**: Entity processing runs asynchronously
4. **Pre-allocation**: Slices pre-allocated where possible to reduce allocations

## Future Improvements

See beads in Zettelgarden-ids2 epic:
- Migrate to LLM job queue system (Zettelgarden-4uqe)
- Add comprehensive tests for ExtractThesesAndArguments (Zettelgarden-ids2.6)
- Further refactoring of complex functions (Zettelgarden-ids2.4)
