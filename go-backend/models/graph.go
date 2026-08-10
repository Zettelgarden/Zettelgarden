package models

// GraphNode is a node in the knowledge graph: a card, entity, or tag.
type GraphNode struct {
	ID     string `json:"id"`                // "card:123", "entity:456", "tag:789"
	Type   string `json:"type"`              // "card" | "entity" | "tag"
	Label  string `json:"label"`             // card title or entity/tag name
	CardID string `json:"card_id,omitempty"` // human card id (cards only)
	Sub    string `json:"sub,omitempty"`     // entity type or tag color (optional)
}

// GraphEdge connects two graph nodes.
type GraphEdge struct {
	Source string `json:"source"` // node id
	Target string `json:"target"` // node id
	Type   string `json:"type"`   // "reference" | "parent" | "entity" | "tag"
}

// GraphData is the full knowledge graph payload for a user's vault.
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
