package services

import (
	"fmt"
	"go-backend/models"
	"log"
	"regexp"
	"sort"
	"strings"
)

// hasAlphabeticChar reports whether s contains at least one ASCII letter.
// Numeric-only card_ids (e.g. "1", "42") appear too commonly in prose to be
// meaningful unlinked-mention targets, so they are skipped.
func hasAlphabeticChar(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}
	return false
}

// snippetAround returns up to radius runes on each side of the byte range
// [loc[0], loc[1]) in body, with ellipses when trimmed.
func snippetAround(body string, loc []int, radius int) string {
	start := loc[0] - radius
	if start < 0 {
		start = 0
	}
	end := loc[1] + radius
	if end > len(body) {
		end = len(body)
	}
	snippet := strings.TrimSpace(body[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(body) {
		snippet = snippet + "..."
	}
	return snippet
}

// GetUnlinkedMentions finds cards that mention the source card's card_id in
// their body without linking to it. Cards that already link to the source
// (via [[card_id]], [[card_id|label]], or legacy [card_id] syntax) are
// excluded. Returns the mention count and a context snippet per card.
func GetUnlinkedMentions(db models.Database, userID int, sourceCard models.Card) ([]models.UnlinkedMention, error) {
	// Numeric-only card_ids would match nearly every body; skip them.
	if !hasAlphabeticChar(sourceCard.CardID) {
		return []models.UnlinkedMention{}, nil
	}

	rows, err := db.Query(`
		SELECT id, body
		FROM cards
		WHERE user_id = $1 AND is_deleted = FALSE AND id != $2 AND instr(body, $3) > 0
	`, userID, sourceCard.ID, sourceCard.CardID)
	if err != nil {
		return nil, fmt.Errorf("failed to query mention candidates: %w", err)
	}
	defer rows.Close()

	mentionRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceCard.CardID) + `\b`)
	alreadyLinked := make(map[int]bool)

	type candidate struct {
		id    int
		count int
		loc   []int
		body  string
	}
	var candidates []candidate
	for rows.Next() {
		var id int
		var body string
		if err := rows.Scan(&id, &body); err != nil {
			return nil, fmt.Errorf("failed to scan mention candidate: %w", err)
		}

		// Skip cards that already link to the source card.
		if !alreadyLinked[id] {
			for _, linked := range ExtractBacklinks(body) {
				if linked == sourceCard.CardID {
					alreadyLinked[id] = true
					break
				}
			}
		}
		if alreadyLinked[id] {
			continue
		}

		locs := mentionRe.FindAllStringIndex(body, -1)
		if len(locs) > 0 {
			candidates = append(candidates, candidate{id: id, count: len(locs), loc: locs[0], body: body})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mention candidates: %w", err)
	}

	mentions := make([]models.UnlinkedMention, 0, len(candidates))
	for _, cand := range candidates {
		partialCard, err := GetPartialCard(db, userID, cand.id)
		if err != nil {
			log.Printf("Failed to get partial card %d: %v", cand.id, err)
			continue
		}
		tags, err := QueryTagsForCard(db, userID, cand.id)
		if err != nil {
			log.Printf("Failed to fetch tags for card %d: %v", cand.id, err)
			partialCard.Tags = []models.Tag{}
		} else {
			partialCard.Tags = tags
		}

		mentions = append(mentions, models.UnlinkedMention{
			Card:           partialCard,
			MentionCount:   cand.count,
			ContextSnippet: snippetAround(cand.body, cand.loc, 40),
		})
	}

	// Sort by mention count descending, then card id.
	sort.Slice(mentions, func(i, j int) bool {
		if mentions[i].MentionCount != mentions[j].MentionCount {
			return mentions[i].MentionCount > mentions[j].MentionCount
		}
		return mentions[i].Card.ID < mentions[j].Card.ID
	})

	return mentions, nil
}

// GetOrphanCards returns the user's cards with no connections: no incoming or
// outgoing references (self-links ignored), no children, and no entities or
// tags shared with any other card. Uses cheap junction queries only — it never
// runs the semantic related-score path (which would hit Typesense per card).
func GetOrphanCards(db models.Database, userID int) ([]models.PartialCard, error) {
	// All non-deleted cards for the user.
	rows, err := db.Query(`
		SELECT id, card_id, user_id, title, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND is_deleted = FALSE
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards for orphans: %w", err)
	}
	defer rows.Close()

	cardsByID := make(map[int]models.PartialCard)
	for rows.Next() {
		card := models.PartialCard{}
		if err := rows.Scan(&card.ID, &card.CardID, &card.UserID, &card.Title, &card.CreatedAt, &card.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan card for orphans: %w", err)
		}
		cardsByID[card.ID] = card
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cards for orphans: %w", err)
	}

	connected := make(map[int]bool, len(cardsByID))

	// 1. Cards with at least one reference in or out (self-links ignored).
	//    backlinks has no user_id, so join through cards on either endpoint.
	refRows, err := db.Query(`
		SELECT DISTINCT b.source_id_int, b.target_id_int
		FROM backlinks b
		JOIN cards c ON c.id IN (b.source_id_int, b.target_id_int)
		WHERE c.user_id = $1 AND c.is_deleted = FALSE
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backlinks for orphans: %w", err)
	}
	defer refRows.Close()
	for refRows.Next() {
		var src, tgt int
		if err := refRows.Scan(&src, &tgt); err != nil {
			return nil, fmt.Errorf("failed to scan backlink for orphans: %w", err)
		}
		if src == tgt {
			continue
		}
		connected[src] = true
		connected[tgt] = true
	}
	if err = refRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backlinks for orphans: %w", err)
	}

	// 2. Cards that have children (parent_id pointing at another card; the
	//    root convention parent_id == id does not count).
	parentRows, err := db.Query(`
		SELECT DISTINCT parent_id
		FROM cards
		WHERE user_id = $1 AND is_deleted = FALSE AND parent_id IS NOT NULL AND parent_id != id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query parents for orphans: %w", err)
	}
	defer parentRows.Close()
	for parentRows.Next() {
		var parentID int
		if err := parentRows.Scan(&parentID); err != nil {
			return nil, fmt.Errorf("failed to scan parent for orphans: %w", err)
		}
		connected[parentID] = true
	}
	if err = parentRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating parents for orphans: %w", err)
	}

	// 3. Cards that share at least one entity with another card.
	entRows, err := db.Query(`
		SELECT DISTINCT ecj1.card_pk, ecj2.card_pk
		FROM entity_card_junction ecj1
		JOIN entity_card_junction ecj2
		  ON ecj1.entity_id = ecj2.entity_id AND ecj1.card_pk < ecj2.card_pk
		WHERE ecj1.user_id = $1 AND ecj2.user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shared entities for orphans: %w", err)
	}
	defer entRows.Close()
	for entRows.Next() {
		var a, b int
		if err := entRows.Scan(&a, &b); err != nil {
			return nil, fmt.Errorf("failed to scan shared entity pair for orphans: %w", err)
		}
		connected[a] = true
		connected[b] = true
	}
	if err = entRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared entity pairs for orphans: %w", err)
	}

	// 4. Cards that share at least one tag with another card.
	tagRows, err := db.Query(`
		SELECT DISTINCT ct1.card_pk, ct2.card_pk
		FROM card_tags ct1
		JOIN card_tags ct2 ON ct1.tag_id = ct2.tag_id AND ct1.card_pk < ct2.card_pk
		JOIN cards c1 ON ct1.card_pk = c1.id
		JOIN cards c2 ON ct2.card_pk = c2.id
		WHERE c1.user_id = $1 AND c2.user_id = $1
		  AND c1.is_deleted = FALSE AND c2.is_deleted = FALSE
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shared tags for orphans: %w", err)
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var a, b int
		if err := tagRows.Scan(&a, &b); err != nil {
			return nil, fmt.Errorf("failed to scan shared tag pair for orphans: %w", err)
		}
		connected[a] = true
		connected[b] = true
	}
	if err = tagRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared tag pairs for orphans: %w", err)
	}

	orphans := make([]models.PartialCard, 0, len(cardsByID))
	for id, card := range cardsByID {
		if !connected[id] {
			orphans = append(orphans, card)
		}
	}
	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].ID < orphans[j].ID
	})

	return orphans, nil
}

// GetDirectLinks extracts and resolves direct links (cards referenced in body) from a card
func GetDirectLinks(db models.Database, userID int, card models.Card) ([]models.PartialCard, error) {
	backlinkIDs := ExtractBacklinks(card.Body)
	var directLinks []models.PartialCard

	for _, cardID := range backlinkIDs {
		partialCard, err := GetPartialCardByCardID(db, userID, cardID)
		if err == nil {
			directLinks = append(directLinks, partialCard)
		}
	}

	return getUniqueCards(directLinks), nil
}

// getUniqueCards removes duplicate cards from a slice based on CardID
func getUniqueCards(input []models.PartialCard) []models.PartialCard {
	u := make([]models.PartialCard, 0, len(input))
	m := make(map[string]bool)

	for _, card := range input {
		if _, ok := m[card.CardID]; !ok {
			m[card.CardID] = true
			u = append(u, card)
		}
	}
	return u
}

// GetReferences returns all unique references (direct links + backlinks) for a card with tags
func GetReferences(db models.Database, userID int, card models.Card) ([]models.PartialCard, error) {
	directLinks, err := GetDirectLinks(db, userID, card)
	if err != nil {
		return nil, err
	}

	backlinks, err := GetBacklinks(db, userID, card.CardID)
	if err != nil {
		return nil, err
	}

	links := append(directLinks, backlinks...)
	if len(links) == 0 {
		return []models.PartialCard{}, nil
	}

	// Sort by card_id descending
	sort.Slice(links, func(x, y int) bool {
		return links[x].CardID > links[y].CardID
	})

	links = getUniqueCards(links)

	// Fetch tags for each card
	for i := range links {
		tags, err := QueryTagsForCard(db, userID, links[i].ID)
		if err != nil {
			log.Printf("Failed to fetch tags for card ID %d: %v", links[i].ID, err)
			// Continue without tags rather than failing entirely
			links[i].Tags = []models.Tag{}
		} else {
			links[i].Tags = tags
		}
	}

	return links, nil
}

// CategorizeReferences categorizes direct links and backlinks into bidirectional, outgoing, and incoming
func CategorizeReferences(directLinks, backlinks []models.PartialCard) models.CategorizedReferences {
	// Create maps for quick lookup
	directMap := make(map[int]models.PartialCard)
	backMap := make(map[int]models.PartialCard)

	for _, card := range directLinks {
		directMap[card.ID] = card
	}
	for _, card := range backlinks {
		backMap[card.ID] = card
	}

	categorized := models.CategorizedReferences{
		Bidirectional: []models.PartialCard{},
		Outgoing:      []models.PartialCard{},
		Incoming:      []models.PartialCard{},
	}

	// Find bidirectional links (cards in both direct and back)
	for id, card := range directMap {
		if _, exists := backMap[id]; exists {
			categorized.Bidirectional = append(categorized.Bidirectional, card)
		} else {
			categorized.Outgoing = append(categorized.Outgoing, card)
		}
	}

	// Find incoming-only links (cards only in backlinks)
	for id, card := range backMap {
		if _, exists := directMap[id]; !exists {
			categorized.Incoming = append(categorized.Incoming, card)
		}
	}

	// Sort each category by card_id descending
	sort.Slice(categorized.Bidirectional, func(i, j int) bool {
		return categorized.Bidirectional[i].CardID > categorized.Bidirectional[j].CardID
	})
	sort.Slice(categorized.Outgoing, func(i, j int) bool {
		return categorized.Outgoing[i].CardID > categorized.Outgoing[j].CardID
	})
	sort.Slice(categorized.Incoming, func(i, j int) bool {
		return categorized.Incoming[i].CardID > categorized.Incoming[j].CardID
	})

	return categorized
}

// GetCategorizedReferences returns categorized references for a card with tags loaded
func GetCategorizedReferences(db models.Database, userID int, card models.Card) (models.CategorizedReferences, error) {
	directLinks, err := GetDirectLinks(db, userID, card)
	if err != nil {
		return models.CategorizedReferences{}, err
	}

	backlinks, err := GetBacklinks(db, userID, card.CardID)
	if err != nil {
		return models.CategorizedReferences{}, err
	}

	// Fetch tags for all cards
	allCards := append(directLinks, backlinks...)
	for i := range allCards {
		tags, err := QueryTagsForCard(db, userID, allCards[i].ID)
		if err != nil {
			log.Printf("Failed to fetch tags for card ID %d: %v", allCards[i].ID, err)
			allCards[i].Tags = []models.Tag{}
		} else {
			allCards[i].Tags = tags
		}
	}

	// Rebuild slices with tags
	dlinksWithTags := allCards[:len(directLinks)]
	backlinksWithTags := allCards[len(directLinks):]

	return CategorizeReferences(dlinksWithTags, backlinksWithTags), nil
}
