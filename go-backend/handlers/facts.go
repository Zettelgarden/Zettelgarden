package handlers

import (
	"context"
	"encoding/json"
	"go-backend/llms"
	"go-backend/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ExtractSaveCardFacts deletes and re-inserts facts for a given card.
func (s *Handler) ExtractSaveCardFacts(userID int, cardPK int, facts []string) ([]models.Fact, error) {
	var results []models.Fact

	tx, _ := s.DB.Begin()
	// First, delete junction links for this card, but only if the fact is not linked to any other cards
	_, err := tx.Exec(`
		DELETE FROM fact_card_junction fcj
		WHERE fcj.card_pk = $1 AND fcj.user_id = $2
		  AND NOT EXISTS (
		    SELECT 1 FROM fact_card_junction fcj2
		    WHERE fcj2.fact_id = fcj.fact_id
		      AND fcj2.card_pk != $1
		  )
	`, cardPK, userID)
	if err != nil {
		log.Printf("error deleting fact-card links: %v", err)
		tx.Rollback()
		return results, err
	}

	// Then, delete orphaned facts whose origin was this card and are not linked elsewhere
	_, err = tx.Exec(`
		DELETE FROM facts f
		WHERE f.card_pk = $1 AND f.user_id = $2
		  AND NOT EXISTS (
		    SELECT 1 FROM fact_card_junction fcj WHERE fcj.fact_id = f.id
		  )
	`, cardPK, userID)
	if err != nil {
		log.Printf("error deleting orphaned facts: %v", err)
		tx.Rollback()
		return results, err
	}

	for _, fact := range facts {
		if fact == "" {
			continue
		}
		embedding, err := llms.GetEmbedding1024(fact, false)
		if err != nil {
			log.Printf("error generating embedding for fact: %v", err)
			tx.Rollback()
			return results, err
		}
		var factID int
		err = tx.QueryRow(`
			INSERT INTO facts (card_pk, user_id, fact, embedding_1024, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id
		`, cardPK, userID, fact, embedding).Scan(&factID)
		if err != nil {
			log.Printf("error inserting fact: %v", err)
			tx.Rollback()
			return results, err
		}

		_, err = tx.Exec(`
			INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
			VALUES ($1, $2, $3, TRUE, NOW(), NOW())
			ON CONFLICT (fact_id, card_pk) DO UPDATE SET updated_at = NOW()
		`, factID, cardPK, userID)
		if err != nil {
			log.Printf("error inserting fact_card_junction: %v", err)
			tx.Rollback()
			return results, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return results, err
	}

	// Fetch the saved facts back, now with IDs, so we can run entity extraction
	rows, err := s.DB.Query(`
		SELECT f.id, f.user_id, fcj.card_pk, f.fact, f.created_at, f.updated_at,
		c.id, c.card_id, c.title, c.parent_id
		FROM facts f
		JOIN fact_card_junction fcj ON f.id = fcj.fact_id
		JOIN cards c ON c.id = f.card_pk
		WHERE fcj.card_pk = $1 AND fcj.user_id = $2
	`, cardPK, userID)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.Fact
		var c models.PartialCard
		if err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.CardPK,
			&f.Fact,
			&f.CreatedAt,
			&f.UpdatedAt,
			&c.ID,
			&c.CardID,
			&c.Title,
			&c.ParentID,
		); err != nil {
			return results, err
		}
		s.upsertFactToTypesense(f, c)
		results = append(results, f)
	}

	// Call entity extraction on the saved facts
	// if err := s.ExtractSaveFactEntities(userID, card, dbFacts); err != nil {
	// 	log.Printf("error extracting entities from facts: %v", err)
	// 	return err
	// }

	return results, nil
}

// GetEntityFacts returns all facts for a given entity, including PartialCard information
func (s *Handler) GetEntityFacts(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	entityIDStr := vars["id"]
	entityID, err := strconv.Atoi(entityIDStr)
	if err != nil {
		log.Printf("err 1 %v", err)
		http.Error(w, "Invalid entity id", http.StatusBadRequest)
		return
	}
	log.Printf("entity id %v", entityID)

	rows, err := s.DB.Query(`
		SELECT f.id, f.fact, f.created_at, f.updated_at,
		       c.id, c.card_id, c.user_id, c.title, c.parent_id,
		       c.created_at, c.updated_at
		FROM facts f
		JOIN entity_fact_junction efj ON f.id = efj.fact_id
		JOIN cards c ON f.card_pk = c.id
		WHERE efj.entity_id = $1 AND efj.user_id = $2
		ORDER BY f.created_at DESC
	`, entityID, userID)
	if err != nil {
		log.Printf("err 2 %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	log.Printf("rows %v", rows)

	type FactWithCard struct {
		ID        int                `json:"id"`
		Fact      string             `json:"fact"`
		CreatedAt time.Time          `json:"created_at"`
		UpdatedAt time.Time          `json:"updated_at"`
		Card      models.PartialCard `json:"card"`
	}

	var facts []FactWithCard

	for rows.Next() {
		var fact FactWithCard
		err := rows.Scan(
			&fact.ID,
			&fact.Fact,
			&fact.CreatedAt,
			&fact.UpdatedAt,
			&fact.Card.ID,
			&fact.Card.CardID,
			&fact.Card.UserID,
			&fact.Card.Title,
			&fact.Card.ParentID,
			&fact.Card.CreatedAt,
			&fact.Card.UpdatedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		facts = append(facts, fact)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(facts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetCardFacts returns all facts for a given card
func (s *Handler) GetCardFacts(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	cardIDStr := vars["id"]
	cardID, err := strconv.Atoi(cardIDStr)
	if err != nil {
		http.Error(w, "Invalid card id", http.StatusBadRequest)
		return
	}

	rows, err := s.DB.Query(`
		SELECT f.id, f.fact, f.created_at, f.updated_at
		FROM facts f
		JOIN fact_card_junction fcj ON f.id = fcj.fact_id
		WHERE fcj.card_pk = $1 AND fcj.user_id = $2
		ORDER BY f.created_at DESC
	`, cardID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var f models.Fact
		if err := rows.Scan(&f.ID, &f.Fact, &f.CreatedAt, &f.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		facts = append(facts, f)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(facts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetFactEntities returns all entities linked to a given fact
func (s *Handler) GetFactEntities(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	factIDStr := vars["id"]
	factID, err := strconv.Atoi(factIDStr)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	rows, err := s.DB.Query(`
		SELECT e.id, e.name, e.description, e.type, e.created_at, e.updated_at
		FROM entities e
		JOIN entity_fact_junction efj ON efj.entity_id = e.id
		WHERE efj.fact_id = $1 AND efj.user_id = $2
	`, factID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.Type, &e.CreatedAt, &e.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entities = append(entities, e)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entities); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type FactWithCard struct {
	ID        int                `json:"id"`
	Fact      string             `json:"fact"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Card      models.PartialCard `json:"card"`
}

// GetAllFacts returns all facts for the current user
func (s *Handler) GetAllFacts(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	rows, err := s.DB.Query(`
		SELECT id, user_id, card_pk, fact, created_at, updated_at
		FROM facts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rows2, err := s.DB.Query(`
		SELECT f.id, f.fact, f.created_at, f.updated_at,
		       c.id, c.card_id, c.user_id, c.title, c.parent_id,
		       c.created_at, c.updated_at
		FROM facts f
		JOIN cards c ON f.card_pk = c.id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows2.Close()

	var facts []FactWithCard
	for rows2.Next() {
		var f FactWithCard
		if err := rows2.Scan(
			&f.ID,
			&f.Fact,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.Card.ID,
			&f.Card.CardID,
			&f.Card.UserID,
			&f.Card.Title,
			&f.Card.ParentID,
			&f.Card.CreatedAt,
			&f.Card.UpdatedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		facts = append(facts, f)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(facts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ExtractSaveFactEntities runs entity extraction on facts and links them in entity_fact_junction
func (s *Handler) ExtractSaveFactEntities(userID int, card models.Card, factObjs []models.Fact) error {
	client := llms.NewDefaultClient(s.DB, userID)

	factEntities, err := llms.FindEntitiesBatch(client, factObjs)
	if err != nil {
		log.Printf("find entities batch err %v", err)
	}
	for i, entities := range factEntities {
		fact := factObjs[i]

		for _, entity := range entities {
			// similarEntities, err := s.FindPotentialDuplicates(userID, entity)
			// if err != nil {
			// 	return err
			// }
			// entity, err = llms.CheckExistingEntities(client, similarEntities, entity)
			// if err != nil {
			// 	log.Printf("error checking existing entities: %v", err)
			// 	return err
			// }
			log.Printf("entity %v", entity.Name)

			var entityID int
			err = s.DB.QueryRow(`
				SELECT id FROM entities WHERE user_id = $1 AND name = $2
			`, userID, entity.Name).Scan(&entityID)

			if err != nil {
				// no entity found, insert
				err = s.DB.QueryRow(`
					INSERT INTO entities (user_id, name, description, type, embedding_1024, card_pk)
					VALUES ($1, $2, $3, $4, $5, $6)
					RETURNING id
				`, userID, entity.Name, entity.Description, entity.Type, entity.Embedding, entity.CardPK).Scan(&entityID)
				if err != nil {
					log.Printf("error inserting entity (from fact): %v", err)
					continue
				}
			} else {
				// entity exists, update
				_, err = s.DB.Exec(`
					UPDATE entities SET description=$1, type=$2, updated_at=NOW() WHERE id=$3
				`, entity.Description, entity.Type, entityID)
				if err != nil {
					log.Printf("error updating entity (from fact): %v", err)
					continue
				}
			}

			// link entity to fact
			_, err = s.DB.Exec(`
				INSERT INTO entity_fact_junction (user_id, entity_id, fact_id, created_at, updated_at)
				VALUES ($1, $2, $3, NOW(), NOW())
				ON CONFLICT (entity_id, fact_id) DO UPDATE SET updated_at = NOW()
			`, userID, entityID, fact.ID)
			if err != nil {
				log.Printf("error linking entity to fact: %v", err)
				continue
			}

			// link entity to fact
			_, err = s.DB.Exec(`
				INSERT INTO entity_card_junction (user_id, entity_id, card_pk, chunk_id)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (entity_id, card_pk) DO UPDATE SET updated_at = NOW()
			`, userID, entityID, card.ID, 0)
			if err != nil {
				log.Printf("error linking entity to card: %v", err)
				continue
			}
		}
	}
	return nil
}

// MergeFacts merges fact2 into fact1 for a given user and deletes fact2
func (s *Handler) MergeFacts(userID int, fact1ID int, fact2ID int) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure both facts exist and belong to the user
	var f1, f2 models.Fact
	err = tx.QueryRow(`SELECT id, user_id, fact FROM facts WHERE id=$1 AND user_id=$2`, fact1ID, userID).
		Scan(&f1.ID, &f1.UserID, &f1.Fact)
	if err != nil {
		return err
	}
	err = tx.QueryRow(`SELECT id, user_id, fact FROM facts WHERE id=$1 AND user_id=$2`, fact2ID, userID).
		Scan(&f2.ID, &f2.UserID, &f2.Fact)
	if err != nil {
		return err
	}

	// Move card relationships
	_, err = tx.Exec(`
		INSERT INTO fact_card_junction (user_id, fact_id, card_pk, is_origin, created_at, updated_at)
		SELECT user_id, $1, card_pk, is_origin, created_at, updated_at
		FROM fact_card_junction WHERE fact_id=$2
		ON CONFLICT (fact_id, card_pk) DO NOTHING
	`, fact1ID, fact2ID)
	if err != nil {
		return err
	}

	// Move entity relationships
	_, err = tx.Exec(`
		INSERT INTO entity_fact_junction (user_id, entity_id, fact_id, created_at, updated_at)
		SELECT user_id, entity_id, $1, created_at, updated_at
		FROM entity_fact_junction WHERE fact_id=$2
		ON CONFLICT (entity_id, fact_id) DO NOTHING
	`, fact1ID, fact2ID)
	if err != nil {
		return err
	}

	// Delete old relationships for fact2
	_, _ = tx.Exec(`DELETE FROM fact_card_junction WHERE fact_id=$1`, fact2ID)
	_, _ = tx.Exec(`DELETE FROM entity_fact_junction WHERE fact_id=$1`, fact2ID)

	// Delete fact2
	_, err = tx.Exec(`DELETE FROM facts WHERE id=$1 AND user_id=$2`, fact2ID, userID)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	s.deleteFactTypesense(fact2ID)
	return nil
}

type MergeFactsRequest struct {
	Fact1ID int `json:"fact1_id"`
	Fact2ID int `json:"fact2_id"`
}

func (s *Handler) MergeFactsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	var req MergeFactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Fact1ID == 0 || req.Fact2ID == 0 {
		http.Error(w, "Both fact IDs are required", http.StatusBadRequest)
		return
	}
	if req.Fact1ID == req.Fact2ID {
		http.Error(w, "Cannot merge a fact with itself", http.StatusBadRequest)
		return
	}
	if err := s.MergeFacts(userID, req.Fact1ID, req.Fact2ID); err != nil {
		log.Printf("Error merging facts: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Facts merged successfully"})
}

// LinkFactToCardHandler links an existing fact to an existing card via fact_card_junction
func (s *Handler) LinkFactToCardHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	factIDStr := vars["factID"]
	cardIDStr := vars["cardID"]

	factID, err := strconv.ParseInt(factIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	cardID, err := strconv.ParseInt(cardIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid card id", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("current_user").(int)
	err = models.LinkFactToCard(s.DB, factID, cardID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
}

// GetFact returns a single fact with its associated card
func (s *Handler) GetFact(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	factIDStr := vars["id"]
	factID, err := strconv.Atoi(factIDStr)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	var fact FactWithCard
	err = s.DB.QueryRow(`
		SELECT f.id, f.fact, f.created_at, f.updated_at,
		       c.id, c.card_id, c.user_id, c.title, c.parent_id,
		       c.created_at, c.updated_at
		FROM facts f
		JOIN cards c ON f.card_pk = c.id
		WHERE f.id = $1 AND f.user_id = $2
	`, factID, userID).Scan(
		&fact.ID,
		&fact.Fact,
		&fact.CreatedAt,
		&fact.UpdatedAt,
		&fact.Card.ID,
		&fact.Card.CardID,
		&fact.Card.UserID,
		&fact.Card.Title,
		&fact.Card.ParentID,
		&fact.Card.CreatedAt,
		&fact.Card.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "Fact not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fact); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetSimilarFacts returns facts with embeddings similar to a target fact
func (s *Handler) GetSimilarFacts(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	factIDStr := vars["id"]
	factID, err := strconv.Atoi(factIDStr)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	limit := 10
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	rows, err := s.DB.Query(`
		SELECT f.id, f.fact, f.created_at, f.updated_at,
		       c.id, c.card_id, c.user_id, c.title, c.parent_id,
		       c.created_at, c.updated_at
		FROM facts f
		JOIN cards c ON f.card_pk = c.id
		WHERE f.user_id = $2 AND f.id != $1
		ORDER BY embedding_1024 <-> (SELECT embedding_1024 FROM facts WHERE id=$1 AND user_id=$2)
		LIMIT $3
	`, factID, userID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var facts []FactWithCard
	for rows.Next() {
		var f FactWithCard
		if err := rows.Scan(
			&f.ID,
			&f.Fact,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.Card.ID,
			&f.Card.CardID,
			&f.Card.UserID,
			&f.Card.Title,
			&f.Card.ParentID,
			&f.Card.CreatedAt,
			&f.Card.UpdatedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		facts = append(facts, f)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(facts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetFactCards returns all cards linked to a given fact
// DeleteFactRoute deletes a fact and its relationships
func (s *Handler) DeleteFactRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	factIDStr := vars["id"]

	factID, err := strconv.Atoi(factIDStr)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	tx, err := s.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Verify fact belongs to user
	var exists bool
	err = tx.QueryRow(`SELECT true FROM facts WHERE id=$1 AND user_id=$2`, factID, userID).Scan(&exists)
	if err != nil {
		http.Error(w, "Fact not found", http.StatusNotFound)
		return
	}

	// Delete relationships
	_, _ = tx.Exec(`DELETE FROM fact_card_junction WHERE fact_id=$1 AND user_id=$2`, factID, userID)
	_, _ = tx.Exec(`DELETE FROM entity_fact_junction WHERE fact_id=$1 AND user_id=$2`, factID, userID)

	// Delete fact
	_, err = tx.Exec(`DELETE FROM facts WHERE id=$1 AND user_id=$2`, factID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.deleteFactTypesense(factID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

type UpdateFactRequest struct {
	Fact string `json:"fact"`
}

func (s *Handler) UpdateFact(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	factIDStr := vars["id"]
	factID, err := strconv.Atoi(factIDStr)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	var req UpdateFactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Fact == "" {
		http.Error(w, "Fact text cannot be empty", http.StatusBadRequest)
		return
	}

	embedding, err := llms.GetEmbedding1024(req.Fact, false)
	if err != nil {
		log.Printf("error generating embedding for fact: %v", err)
		http.Error(w, "Failed to generate embedding", http.StatusInternalServerError)
		return
	}

	tx, err := s.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE facts
		SET fact = $1, embedding_1024 = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
	`, req.Fact, embedding, factID, userID)
	if err != nil {
		http.Error(w, "Failed to update fact", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-fetch the fact and card info to update Typesense
	var fact models.Fact
	var card models.PartialCard
	err = s.DB.QueryRow(`
		SELECT f.id, f.user_id, fcj.card_pk, f.fact, f.created_at, f.updated_at,
		c.id, c.card_id, c.title, c.parent_id
		FROM facts f
		JOIN fact_card_junction fcj ON f.id = fcj.fact_id
		JOIN cards c ON c.id = f.card_pk
		WHERE f.id = $1 AND f.user_id = $2
	`, factID, userID).Scan(
		&fact.ID,
		&fact.UserID,
		&fact.CardPK,
		&fact.Fact,
		&fact.CreatedAt,
		&fact.UpdatedAt,
		&card.ID,
		&card.CardID,
		&card.Title,
		&card.ParentID,
	)
	if err != nil {
		log.Printf("Failed to fetch fact for typesense update: %v", err)
		// Non-fatal, as the main update succeeded
	} else {
		s.upsertFactToTypesense(fact, card)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Fact updated successfully"})
}

func (s *Handler) GetFactCards(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)

	factIDStr := vars["id"]
	factID, err := strconv.Atoi(factIDStr)
	if err != nil {
		http.Error(w, "Invalid fact id", http.StatusBadRequest)
		return
	}

	rows, err := s.DB.Query(`
		SELECT c.id, c.card_id, c.user_id, c.title, c.parent_id, c.created_at, c.updated_at
		FROM cards c
		JOIN fact_card_junction fcj ON c.id = fcj.card_pk
		WHERE fcj.fact_id = $1 AND fcj.user_id = $2
	`, factID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var c models.PartialCard
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cards = append(cards, c)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cards); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Handler) upsertFactToTypesense(fact models.Fact, card models.PartialCard) {
	if s.Server.Testing {
		return
	}
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	doc := map[string]interface{}{
		"id":                    "fact-" + strconv.Itoa(fact.ID),
		"fact_pk":               fact.ID,
		"card_id":               "",
		"card_pk":               -1,
		"entity_pk":             -1,
		"user_id":               fact.UserID,
		"type":                  "fact",
		"title":                 fact.Fact,
		"preview":               "",
		"parent_id":             -1,
		"created_at":            fact.CreatedAt.Unix(),
		"updated_at":            fact.UpdatedAt.Unix(),
		"linked_card_id":        card.CardID,
		"linked_card_pk":        fact.CardPK,
		"linked_card_title":     card.Title,
		"linked_card_parent_id": card.ParentID,
	}

	_, err := s.Server.TypesenseClient.Collection(collectionName).
		Documents().Upsert(context.Background(), doc)
	if err != nil {
		log.Printf("failed to upsert fact ID %d: %v", fact.ID, err)
	}
}

func (s *Handler) deleteFactTypesense(factPK int) {
	if s.Server.Testing {
		return
	}
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	_, err := s.Server.TypesenseClient.Collection(collectionName).
		Document("fact-" + strconv.Itoa(factPK)).Delete(context.Background())
	if err != nil {
		log.Printf("failed to delete fact ID %d: %v", factPK, err)
	}
}
