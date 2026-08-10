package models

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

type Card struct {
	ID             int           `json:"id"`
	CardID         string        `json:"card_id"`
	UserID         int           `json:"user_id"`
	Title          string        `json:"title"`
	Body           string        `json:"body"`
	Link           string        `json:"link"`
	IsDeleted      bool          `json:"is_deleted"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	ParentID       *int          `json:"parent_id"`
	Parent         PartialCard   `json:"parent"`
	Files          []File        `json:"files"`
	Children       []PartialCard `json:"children"`
	References     []PartialCard `json:"references"`
	Keywords       []Keyword     `json:"keywords"`
	Tags           []Tag         `json:"tags"`
	Tasks          []Task        `json:"tasks"`
	Entities       []Entity      `json:"entities"`
	TagCount       int
	IsStarred      bool          `json:"is_starred"`
	SchemaID       *int          `json:"schema_id,omitempty"`
	StructuredData *json.RawMessage `json:"structured_data,omitempty"`
	SourceArticle  *RSSArticle   `json:"source_article,omitempty"`
}

func ScanCards(rows *sql.Rows) ([]Card, error) {
	var cards []Card

	for rows.Next() {
		var card Card
		var parentID sql.NullInt64
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.Body,
			&card.Link,
			&parentID,
			&card.CreatedAt,
			&card.UpdatedAt,
			&card.TagCount,
		); err != nil {
			log.Printf(" query full err %v", err)
			return cards, err
		}
		if parentID.Valid {
			card.ParentID = new(int)
			*card.ParentID = int(parentID.Int64)
		}
		cards = append(cards, card)
	}

	return cards, nil
}

func ScanPartialCards(rows *sql.Rows) ([]PartialCard, error) {
	var cards []PartialCard

	for rows.Next() {
		var card PartialCard
		var parentID sql.NullInt64
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&parentID,
			&card.CreatedAt,
			&card.UpdatedAt,
		); err != nil {
			log.Printf("err %v", err)
			return cards, err
		}
		if parentID.Valid {
			card.ParentID = new(int)
			*card.ParentID = int(parentID.Int64)
		}
		cards = append(cards, card)

	}
	return cards, nil

}

type PartialCard struct {
	ID        int       `json:"id"`
	CardID    string    `json:"card_id"`
	UserID    int       `json:"user_id"`
	Title     string    `json:"title"`
	ParentID  *int      `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Tags      []Tag     `json:"tags"`
}

// CategorizedReferences represents references categorized by their relationship type
type CategorizedReferences struct {
	Bidirectional []PartialCard `json:"bidirectional"` // Two-way links (mutual references)
	Outgoing      []PartialCard `json:"outgoing"`      // One-way links (this card references them)
	Incoming      []PartialCard `json:"incoming"`      // One-way links (they reference this card)
}

type CardWithDescendants struct {
	ID           int                    `json:"id"`
	CardID       string                 `json:"card_id"`
	UserID       int                    `json:"user_id"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Link         string                 `json:"link"`
	ParentID     int                    `json:"parent_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Depth        int                    `json:"depth"`
	Descendants  []CardWithDescendants  `json:"descendants"`
}

// RelatedCard represents a card with its relatedness score and reasons
type RelatedCard struct {
	Card    PartialCard `json:"card"`
	Score   float64     `json:"score"`
	Reasons []string    `json:"reasons"` // human-readable strings, e.g. "3 shared entities: Python, LLM"
}

// UnlinkedMention represents a card whose body mentions another card's
// card_id without linking to it.
type UnlinkedMention struct {
	Card           PartialCard `json:"card"`
	MentionCount   int         `json:"mention_count"`
	ContextSnippet string      `json:"context_snippet"`
}

// SharedMatch represents cards that share entities or tags with a source card
type SharedMatch struct {
	Count int      // number of shared entities/tags
	Names []string // names of the shared entities/tags
}

func ConvertCardToPartialCard(input Card) PartialCard {
	return PartialCard{
		ID:        input.ID,
		CardID:    input.CardID,
		UserID:    input.UserID,
		Title:     input.Title,
		ParentID:  input.ParentID,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.UpdatedAt,
		Tags:      input.Tags,
	}
}

type EditCardParams struct {
	CardID                  string           `json:"card_id"`
	Title                   string           `json:"title"`
	Body                    string           `json:"body"`
	Link                    string           `json:"link"`
	ProcessEntitiesAndFacts *bool            `json:"process_entities_and_facts,omitempty"`
	SchemaID                *int             `json:"schema_id,omitempty"`
	StructuredData          *json.RawMessage `json:"structured_data,omitempty"`
	ClearSchema             bool             `json:"clear_schema,omitempty"` // Explicit flag to remove schema association
}

type NextIDParams struct {
	CardType string `json:"card_type"`
}

type NextIDResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	NextID  string `json:"new_id"`
}

type CardChunk struct {
	ID               int       `json:"id"`
	CardID           string    `json:"card_id"`
	UserID           int       `json:"user_id"`
	Title            string    `json:"title"`
	Chunk            string    `json:"body"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ParentID         *int      `json:"parent_id"`
	Ranking          float64   `json:"ranking"`
	SharedEntities   int       `json:"shared_entities"`
	EntitySimilarity float64   `json:"entity_similarity"`
	CombinedScore    float64   `json:"combined_score"`
}

func ScanCardChunks(rows *sql.Rows) ([]CardChunk, error) {
	var cards []CardChunk

	for rows.Next() {
		var card CardChunk
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.Chunk,
			&card.CreatedAt,
			&card.UpdatedAt,
			&card.ParentID,
			&card.Ranking,
			&card.SharedEntities,
			&card.EntitySimilarity,
			&card.CombinedScore,
		); err != nil {
			return cards, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func ConvertCardToChunk(input Card) CardChunk {
	return CardChunk{
		ID:        input.ID,
		CardID:    input.CardID,
		UserID:    input.UserID,
		Title:     input.Title,
		Chunk:     input.Body,
		ParentID:  input.ParentID,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.UpdatedAt,
	}
}
