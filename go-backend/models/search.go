package models

import (
	"time"
)

type SearchResult struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Title     string      `json:"title"`
	Preview   string      `json:"preview"`
	Score     float64     `json:"score"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Metadata  interface{} `json:"metadata"`
	Tags      []Tag       `json:"tags"`
	CardID    string      `json:"card_id"`
}

func CardChunkToSearchResult(chunk CardChunk) SearchResult {
	return SearchResult{
		ID:        chunk.CardID,
		Type:      "card",
		Title:     chunk.Title,
		Preview:   chunk.Chunk,
		Score:     chunk.CombinedScore,
		CreatedAt: chunk.CreatedAt,
		UpdatedAt: chunk.UpdatedAt,
		Metadata: map[string]interface{}{
			"id":                chunk.ID,
			"parent_id":         chunk.ParentID,
			"shared_entities":   chunk.SharedEntities,
			"entity_similarity": chunk.EntitySimilarity,
			"semantic_ranking":  chunk.Ranking,
		},
	}
}
