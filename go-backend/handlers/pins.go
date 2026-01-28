package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// StarCardRoute handles the request to star a card
func (s *Handler) StarCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify the card exists and belongs to the user
	_, err = s.QueryFullCard(userID, cardID)
	if err != nil {
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	// Star the card
	_, err = s.DB.Exec(
		"INSERT INTO starred_cards (card_pk, user_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (card_pk, user_id) DO NOTHING",
		cardID, userID,
	)
	if err != nil {
		log.Printf("Error starring card: %v", err)
		http.Error(w, "Failed to star card", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UnstarCardRoute handles the request to unstar a card
func (s *Handler) UnstarCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Delete the star
	result, err := s.DB.Exec(
		"DELETE FROM starred_cards WHERE card_pk = $1 AND user_id = $2",
		cardID, userID,
	)
	if err != nil {
		log.Printf("Error unstarring card: %v", err)
		http.Error(w, "Failed to unstar card", http.StatusInternalServerError)
		return
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		http.Error(w, "Failed to unstar card", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Card was not starred", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStarredCardsRoute handles the request to get all starred cards for a user
func (s *Handler) GetStarredCardsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Query for starred cards with full card data
	rows, err := s.DB.Query(`
		SELECT
			sc.id, sc.card_pk, sc.user_id, sc.created_at,
			c.id, c.card_id, c.user_id, c.title, c.body, c.link, c.parent_id, c.created_at, c.updated_at
		FROM starred_cards sc
		JOIN cards c ON sc.card_pk = c.id
		WHERE sc.user_id = $1 AND c.is_deleted = FALSE
		ORDER BY sc.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("Error querying starred cards: %v", err)
		http.Error(w, "Failed to retrieve starred cards", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var starredCards []models.StarredCardResponse
	for rows.Next() {
		var starredCard models.StarredCardResponse
		var card models.Card

		err := rows.Scan(
			&starredCard.ID, &starredCard.CardPK, &starredCard.UserID, &starredCard.CreatedAt,
			&card.ID, &card.CardID, &card.UserID, &card.Title, &card.Body, &card.Link,
			&card.ParentID, &card.CreatedAt, &card.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning starred card: %v", err)
			http.Error(w, "Failed to process starred cards", http.StatusInternalServerError)
			return
		}

		// Get parent card
		parent, err := s.QueryPartialCardByID(userID, card.ParentID)
		if err != nil {
			log.Printf("Error getting parent card: %v", err)
			// Continue even if parent can't be found
		}
		card.Parent = parent

		// Get tags for the card
		tags, err := services.QueryTagsForCard(s.DB, userID, card.ID)
		if err != nil {
			log.Printf("Error getting tags: %v", err)
			// Continue even if tags can't be found
		}
		card.Tags = tags

		starredCard.Card = card
		starredCards = append(starredCards, starredCard)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating starred cards: %v", err)
		http.Error(w, "Failed to process starred cards", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(starredCards)
}

// IsCardStarred checks if a card is starred by the current user
func (s *Handler) IsCardStarred(userID, cardID int) (bool, error) {
	var count int
	err := s.GetDB().QueryRow(
		"SELECT COUNT(*) FROM starred_cards WHERE card_pk = $1 AND user_id = $2",
		cardID, userID,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error checking if card is starred: %w", err)
	}

	return count > 0, nil
}

// StarSearchRoute handles the request to star a search
func (s *Handler) StarSearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse the request body
	var req models.StarredSearchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Insert the starred search
	var id int
	err = s.DB.QueryRow(
		"INSERT INTO starred_searches (user_id, title, search_term, search_config, created_at) VALUES ($1, $2, $3, $4, NOW()) RETURNING id",
		userID, req.Title, req.SearchTerm, req.SearchConfig,
	).Scan(&id)
	if err != nil {
		log.Printf("Error starring search: %v", err)
		http.Error(w, "Failed to star search", http.StatusInternalServerError)
		return
	}

	// Return the ID of the newly created starred search
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// UnstarSearchRoute handles the request to unstar a search
func (s *Handler) UnstarSearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	searchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid search ID", http.StatusBadRequest)
		return
	}

	// Delete the star
	result, err := s.DB.Exec(
		"DELETE FROM starred_searches WHERE id = $1 AND user_id = $2",
		searchID, userID,
	)
	if err != nil {
		log.Printf("Error unstarring search: %v", err)
		http.Error(w, "Failed to unstar search", http.StatusInternalServerError)
		return
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		http.Error(w, "Failed to unstar search", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Search was not starred", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStarredSearchesRoute handles the request to get all starred searches for a user
func (s *Handler) GetStarredSearchesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Query for starred searches
	rows, err := s.DB.Query(`
		SELECT id, user_id, title, search_term, search_config, created_at
		FROM starred_searches
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		log.Printf("Error querying starred searches: %v", err)
		http.Error(w, "Failed to retrieve starred searches", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var starredSearches []models.StarredSearch
	for rows.Next() {
		var starredSearch models.StarredSearch
		err := rows.Scan(
			&starredSearch.ID,
			&starredSearch.UserID,
			&starredSearch.Title,
			&starredSearch.SearchTerm,
			&starredSearch.SearchConfig,
			&starredSearch.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning starred search: %v", err)
			http.Error(w, "Failed to process starred searches", http.StatusInternalServerError)
			return
		}

		starredSearches = append(starredSearches, starredSearch)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating starred searches: %v", err)
		http.Error(w, "Failed to process starred searches", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(starredSearches)
}
