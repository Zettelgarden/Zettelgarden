package services

import (
	"fmt"
	"go-backend/models"
	"strings"
)

// GraphNodeTypes are the supported node types in the knowledge graph.
var GraphNodeTypes = map[string]bool{
	"card":   true,
	"entity": true,
	"tag":    true,
}

// ParseGraphTypes parses a comma-separated types filter (e.g. "card,entity").
// Empty input means all types. Unknown types are ignored; if the result is
// empty, the default set is returned.
func ParseGraphTypes(raw string) map[string]bool {
	if strings.TrimSpace(raw) == "" {
		return copyStringSet(GraphNodeTypes)
	}
	types := make(map[string]bool)
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if GraphNodeTypes[t] {
			types[t] = true
		}
	}
	if len(types) == 0 {
		return copyStringSet(GraphNodeTypes)
	}
	return types
}

func copyStringSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k := range in {
		out[k] = true
	}
	return out
}

// GetGraphData returns the user's knowledge graph: card/entity/tag nodes and
// reference/parent/entity/tag edges, filtered to the requested node types.
func GetGraphData(db models.Database, userID int, types map[string]bool) (models.GraphData, error) {
	data := models.GraphData{
		Nodes: []models.GraphNode{},
		Edges: []models.GraphEdge{},
	}

	cardIDs := make(map[int]bool)   // card node ids present
	entityIDs := make(map[int]bool) // entity node ids present
	tagIDs := make(map[int]bool)    // tag node ids present

	includeCards := types["card"]
	includeEntities := types["entity"]
	includeTags := types["tag"]

	// Cards
	if includeCards {
		rows, err := db.Query(`
			SELECT id, card_id, title
			FROM cards
			WHERE user_id = $1 AND is_deleted = FALSE
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph cards: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int
			var cardID, title string
			if err := rows.Scan(&id, &cardID, &title); err != nil {
				return data, fmt.Errorf("failed to scan graph card: %w", err)
			}
			cardIDs[id] = true
			data.Nodes = append(data.Nodes, models.GraphNode{
				ID:     fmt.Sprintf("card:%d", id),
				Type:   "card",
				Label:  title,
				CardID: cardID,
			})
		}
		if err = rows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph cards: %w", err)
		}
	}

	// Entities
	if includeEntities {
		rows, err := db.Query(`
			SELECT id, name, type
			FROM entities
			WHERE user_id = $1
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph entities: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int
			var name, sub string
			if err := rows.Scan(&id, &name, &sub); err != nil {
				return data, fmt.Errorf("failed to scan graph entity: %w", err)
			}
			entityIDs[id] = true
			data.Nodes = append(data.Nodes, models.GraphNode{
				ID:    fmt.Sprintf("entity:%d", id),
				Type:  "entity",
				Label: name,
				Sub:   sub,
			})
		}
		if err = rows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph entities: %w", err)
		}
	}

	// Tags
	if includeTags {
		rows, err := db.Query(`
			SELECT id, name, color
			FROM tags
			WHERE user_id = $1 AND is_deleted = FALSE
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph tags: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int
			var name, color string
			if err := rows.Scan(&id, &name, &color); err != nil {
				return data, fmt.Errorf("failed to scan graph tag: %w", err)
			}
			tagIDs[id] = true
			data.Nodes = append(data.Nodes, models.GraphNode{
				ID:    fmt.Sprintf("tag:%d", id),
				Type:  "tag",
				Label: name,
				Sub:   color,
			})
		}
		if err = rows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph tags: %w", err)
		}
	}

	// Reference edges (card -> card via backlinks, ownership on both sides).
	if includeCards {
		rows, err := db.Query(`
			SELECT DISTINCT b.source_id_int, b.target_id_int
			FROM backlinks b
			JOIN cards cs ON b.source_id_int = cs.id
			JOIN cards ct ON b.target_id_int = ct.id
			WHERE cs.user_id = $1 AND ct.user_id = $1
			  AND cs.is_deleted = FALSE AND ct.is_deleted = FALSE
			  AND b.source_id_int != b.target_id_int
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph reference edges: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var src, tgt int
			if err := rows.Scan(&src, &tgt); err != nil {
				return data, fmt.Errorf("failed to scan graph reference edge: %w", err)
			}
			data.Edges = append(data.Edges, models.GraphEdge{
				Source: fmt.Sprintf("card:%d", src),
				Target: fmt.Sprintf("card:%d", tgt),
				Type:   "reference",
			})
		}
		if err = rows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph reference edges: %w", err)
		}

		// Parent edges (card -> card).
		prows, err := db.Query(`
			SELECT id, parent_id
			FROM cards
			WHERE user_id = $1 AND is_deleted = FALSE
			  AND parent_id IS NOT NULL AND parent_id != id
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph parent edges: %w", err)
		}
		defer prows.Close()
		for prows.Next() {
			var id, parentID int
			if err := prows.Scan(&id, &parentID); err != nil {
				return data, fmt.Errorf("failed to scan graph parent edge: %w", err)
			}
			if cardIDs[id] && cardIDs[parentID] {
				data.Edges = append(data.Edges, models.GraphEdge{
					Source: fmt.Sprintf("card:%d", parentID),
					Target: fmt.Sprintf("card:%d", id),
					Type:   "parent",
				})
			}
		}
		if err = prows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph parent edges: %w", err)
		}
	}

	// Entity edges (card -> entity).
	if includeCards && includeEntities {
		rows, err := db.Query(`
			SELECT ecj.card_pk, ecj.entity_id
			FROM entity_card_junction ecj
			JOIN cards c ON ecj.card_pk = c.id
			WHERE ecj.user_id = $1 AND c.is_deleted = FALSE
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph entity edges: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var cardPK, entityID int
			if err := rows.Scan(&cardPK, &entityID); err != nil {
				return data, fmt.Errorf("failed to scan graph entity edge: %w", err)
			}
			if cardIDs[cardPK] && entityIDs[entityID] {
				data.Edges = append(data.Edges, models.GraphEdge{
					Source: fmt.Sprintf("card:%d", cardPK),
					Target: fmt.Sprintf("entity:%d", entityID),
					Type:   "entity",
				})
			}
		}
		if err = rows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph entity edges: %w", err)
		}
	}

	// Tag edges (card -> tag).
	if includeCards && includeTags {
		rows, err := db.Query(`
			SELECT ct.card_pk, ct.tag_id
			FROM card_tags ct
			JOIN cards c ON ct.card_pk = c.id
			WHERE c.user_id = $1 AND c.is_deleted = FALSE
		`, userID)
		if err != nil {
			return data, fmt.Errorf("failed to query graph tag edges: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var cardPK, tagID int
			if err := rows.Scan(&cardPK, &tagID); err != nil {
				return data, fmt.Errorf("failed to scan graph tag edge: %w", err)
			}
			if cardIDs[cardPK] && tagIDs[tagID] {
				data.Edges = append(data.Edges, models.GraphEdge{
					Source: fmt.Sprintf("card:%d", cardPK),
					Target: fmt.Sprintf("tag:%d", tagID),
					Type:   "tag",
				})
			}
		}
		if err = rows.Err(); err != nil {
			return data, fmt.Errorf("error iterating graph tag edges: %w", err)
		}
	}

	return data, nil
}
