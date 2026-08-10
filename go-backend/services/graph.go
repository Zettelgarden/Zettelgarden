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

// FindPathBetweenCards returns the shortest undirected path (as cards) between
// two of the user's cards, using backlink and parent-child edges. Returns an
// empty slice when the cards are disconnected or the endpoints are equal.
func FindPathBetweenCards(db models.Database, userID int, fromID, toID int) ([]models.PartialCard, error) {
	if fromID == toID {
		return []models.PartialCard{}, nil
	}

	// All user cards.
	rows, err := db.Query(`
		SELECT id, card_id, user_id, title, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND is_deleted = FALSE
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards for path: %w", err)
	}
	defer rows.Close()

	cardsByID := make(map[int]models.PartialCard)
	for rows.Next() {
		card := models.PartialCard{}
		if err := rows.Scan(&card.ID, &card.CardID, &card.UserID, &card.Title, &card.CreatedAt, &card.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan card for path: %w", err)
		}
		cardsByID[card.ID] = card
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cards for path: %w", err)
	}

	if _, ok := cardsByID[fromID]; !ok {
		return []models.PartialCard{}, nil
	}
	if _, ok := cardsByID[toID]; !ok {
		return []models.PartialCard{}, nil
	}

	// Undirected adjacency: backlink edges both ways, plus parent-child both ways.
	adj := make(map[int][]int, len(cardsByID))
	addEdge := func(a, b int) {
		if _, ok := cardsByID[a]; !ok {
			return
		}
		if _, ok := cardsByID[b]; !ok {
			return
		}
		adj[a] = append(adj[a], b)
		adj[b] = append(adj[b], a)
	}

	refRows, err := db.Query(`
		SELECT DISTINCT b.source_id_int, b.target_id_int
		FROM backlinks b
		JOIN cards cs ON b.source_id_int = cs.id
		JOIN cards ct ON b.target_id_int = ct.id
		WHERE cs.user_id = $1 AND ct.user_id = $1
		  AND cs.is_deleted = FALSE AND ct.is_deleted = FALSE
		  AND b.source_id_int != b.target_id_int
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backlinks for path: %w", err)
	}
	defer refRows.Close()
	for refRows.Next() {
		var src, tgt int
		if err := refRows.Scan(&src, &tgt); err != nil {
			return nil, fmt.Errorf("failed to scan backlink for path: %w", err)
		}
		addEdge(src, tgt)
	}
	if err = refRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backlinks for path: %w", err)
	}

	parRows, err := db.Query(`
		SELECT id, parent_id
		FROM cards
		WHERE user_id = $1 AND is_deleted = FALSE
		  AND parent_id IS NOT NULL AND parent_id != id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query parents for path: %w", err)
	}
	defer parRows.Close()
	for parRows.Next() {
		var id, parentID int
		if err := parRows.Scan(&id, &parentID); err != nil {
			return nil, fmt.Errorf("failed to scan parent for path: %w", err)
		}
		addEdge(parentID, id)
	}
	if err = parRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating parents for path: %w", err)
	}

	// BFS from fromID to toID.
	prev := make(map[int]int)
	visited := map[int]bool{fromID: true}
	queue := []int{fromID}
	for len(queue) > 0 && !visited[toID] {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range adj[cur] {
			if !visited[nb] {
				visited[nb] = true
				prev[nb] = cur
				queue = append(queue, nb)
			}
		}
	}

	if !visited[toID] {
		return []models.PartialCard{}, nil
	}

	// Reconstruct path from -> ... -> to.
	pathIDs := []int{toID}
	for cur := toID; cur != fromID; cur = prev[cur] {
		pathIDs = append([]int{prev[cur]}, pathIDs...)
	}

	path := make([]models.PartialCard, 0, len(pathIDs))
	for _, id := range pathIDs {
		if card, ok := cardsByID[id]; ok {
			path = append(path, card)
		}
	}
	return path, nil
}
