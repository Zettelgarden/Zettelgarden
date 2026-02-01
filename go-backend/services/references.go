package services

import (
	"go-backend/models"
	"log"
	"sort"
)

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
