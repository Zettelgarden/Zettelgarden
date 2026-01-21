package server

import (
	"context"
	"fmt"
	"go-backend/models"
	"log"
	"os"

	"github.com/typesense/typesense-go/typesense/api"
)

// SimilarEntity represents an entity with its similarity score
type SimilarEntity struct {
	ID    int
	Score float64
}

// SimilarFact represents a fact with its similarity score
type SimilarFact struct {
	ID    int
	Score float64
}

// FindSimilarEntities finds entities similar to the given entity using Typesense
func (s *Server) FindSimilarEntities(ctx context.Context, entity models.Entity, limit int) ([]SimilarEntity, error) {
	if s.TypesenseClient == nil {
		log.Printf("Typesense client not available")
		return nil, nil // Return empty slice, will fallback to text similarity
	}

	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		log.Printf("TYPESENSE_COLLECTION env var not set")
		return nil, nil
	}

	filter := fmt.Sprintf("user_id:=%d && type:=entity", entity.UserID)
	perPage := limit

	searchParams := &api.SearchCollectionParams{
		Q:        entity.Name,
		QueryBy:  "title,embedding",
		FilterBy: &filter,
		PerPage:  &perPage,
	}

	searchResult, err := s.TypesenseClient.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		log.Printf("Typesense similarity search failed: %v", err)
		return nil, nil // Return empty slice to trigger fallback
	}

	var similarEntities []SimilarEntity
	if searchResult.Hits != nil {
		for _, hit := range *searchResult.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				if pk, ok := doc["entity_pk"].(float64); ok && int(pk) != entity.ID {
					score := 0.0
					// Extract vector distance if available (lower distance = more similar)
					if hit.VectorDistance != nil {
						distance := float64(*hit.VectorDistance)
						// Convert distance to similarity score (0-1 range, where 1 = most similar)
						// For cosine distance: 0 = identical, 2 = opposite
						// Similarity = 1 - (distance / 2) gives us 0-1 range
						score = 1.0 - (distance / 2.0)
						// Clamp to 0-1 range
						if score < 0 {
							score = 0
						} else if score > 1 {
							score = 1
						}
					}
					similarEntities = append(similarEntities, SimilarEntity{
						ID:    int(pk),
						Score: score,
					})
				}
			}
		}
	}

	return similarEntities, nil
}

// FindSimilarFacts finds facts similar to the given fact using Typesense
func (s *Server) FindSimilarFacts(ctx context.Context, fact models.Fact, limit int) ([]SimilarFact, error) {
	if s.TypesenseClient == nil {
		log.Printf("Typesense client not available")
		return nil, nil // Return empty slice, will fallback to text similarity
	}

	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		log.Printf("TYPESENSE_COLLECTION env var not set")
		return nil, nil
	}

	filter := fmt.Sprintf("user_id:=%d && type:=fact", fact.UserID)
	perPage := limit

	searchParams := &api.SearchCollectionParams{
		Q:        fact.Fact,
		QueryBy:  "title,embedding",
		FilterBy: &filter,
		PerPage:  &perPage,
	}

	searchResult, err := s.TypesenseClient.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		log.Printf("Typesense similarity search failed: %v", err)
		return nil, nil // Return empty slice to trigger fallback
	}

	var similarFacts []SimilarFact
	if searchResult.Hits != nil {
		for _, hit := range *searchResult.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				if pk, ok := doc["fact_pk"].(float64); ok && int(pk) != fact.ID {
					score := 0.0
					// Extract vector distance if available (lower distance = more similar)
					if hit.VectorDistance != nil {
						distance := float64(*hit.VectorDistance)
						log.Printf("DEBUG: fact ID %d, vector distance: %f", int(pk), distance)
						// Convert distance to similarity score (0-1 range, where 1 = most similar)
						// For cosine distance: 0 = identical, 2 = opposite
						// Similarity = 1 - (distance / 2) gives us 0-1 range
						score = 1.0 - (distance / 2.0)
						// Clamp to 0-1 range
						if score < 0 {
							score = 0
						} else if score > 1 {
							score = 1
						}
					}
					log.Printf("DEBUG: Similar fact ID %d, final score: %f", int(pk), score)
					similarFacts = append(similarFacts, SimilarFact{
						ID:    int(pk),
						Score: score,
					})
				}
			}
		}
	}

	return similarFacts, nil
}

// MergeEntities merges entity2 into entity1
func (s *Server) MergeEntities(ctx context.Context, userID int, entity1ID int, entity2ID int) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify both entities exist and belong to the user
	var entity1, entity2 models.Entity
	err = tx.QueryRow(`
		SELECT id, user_id, name, description, type, card_pk
		FROM entities
		WHERE id = $1 AND user_id = $2`,
		entity1ID, userID).Scan(
		&entity1.ID, &entity1.UserID, &entity1.Name,
		&entity1.Description, &entity1.Type, &entity1.CardPK)
	if err != nil {
		return fmt.Errorf("failed to find entity1: %w", err)
	}

	err = tx.QueryRow(`
		SELECT id, user_id, name, description, type, card_pk
		FROM entities
		WHERE id = $1 AND user_id = $2`,
		entity2ID, userID).Scan(
		&entity2.ID, &entity2.UserID, &entity2.Name,
		&entity2.Description, &entity2.Type, &entity2.CardPK)
	if err != nil {
		return fmt.Errorf("failed to find entity2: %w", err)
	}

	// Move all card relationships from entity2 to entity1
	_, err = tx.Exec(`
		INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
		SELECT user_id, $1, card_pk
		FROM entity_card_junction
		WHERE entity_id = $2
		ON CONFLICT (entity_id, card_pk) DO NOTHING`,
		entity1.ID, entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to merge card relationships: %w", err)
	}

	// Move all fact relationships from entity2 to entity1
	_, err = tx.Exec(`
		INSERT INTO entity_fact_junction (user_id, entity_id, fact_id)
		SELECT user_id, $1, fact_id
		FROM entity_fact_junction
		WHERE entity_id = $2
		ON CONFLICT (entity_id, fact_id) DO NOTHING`,
		entity1.ID, entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to merge fact relationships: %w", err)
	}

	// Delete entity2's relationships
	_, err = tx.Exec(`
		DELETE FROM entity_card_junction
		WHERE entity_id = $1`,
		entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to delete entity2 card relationships: %w", err)
	}

	_, err = tx.Exec(`
		DELETE FROM entity_fact_junction
		WHERE entity_id = $1`,
		entity2.ID)
	if err != nil {
		return fmt.Errorf("failed to delete entity2 fact relationships: %w", err)
	}

	// Combine descriptions
	newDescription := fmt.Sprintf("%s\n\nMerged from duplicates:\n---\n%s (%s)",
		entity1.Description, entity2.Name, entity2.Description)
	if entity1.Description == "" {
		newDescription = fmt.Sprintf("Merged from duplicates:\n---\n%s (%s)",
			entity2.Name, entity2.Description)
	}

	// Preserve card_pk from either entity (prefer entity1, fallback to entity2)
	var cardPK *int
	if entity1.CardPK != nil {
		cardPK = entity1.CardPK
	} else {
		cardPK = entity2.CardPK
	}

	_, err = tx.Exec(`UPDATE entities SET description = $1, card_pk = $2, updated_at = NOW() WHERE id = $3`,
		newDescription, cardPK, entity1.ID)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	// Delete entity2
	_, err = tx.Exec(`
		DELETE FROM entities
		WHERE id = $1 AND user_id = $2`,
		entity2.ID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete entity2: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// MergeFacts merges fact2 into fact1
func (s *Server) MergeFacts(ctx context.Context, userID int, fact1ID int, fact2ID int) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	return nil
}
