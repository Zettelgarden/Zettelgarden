package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-backend/tests"

	"github.com/gorilla/mux"
)

func TestLinkFactToCardHandler(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	log.Printf("aoeao")
	// Insert a fact manually linked to card 1
	var factID int
	err := s.Server.Tx.QueryRow(`
        INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
        VALUES ($1, $2, 'test fact', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        RETURNING id
    `, 1, 1).Scan(&factID)
	if err != nil {
		t.Fatalf("failed to insert fact: %v", err)
	}
}

// Test ExtractSaveCardFacts preserves facts linked to multiple cards
func TestExtractSaveCardFacts_MultiCardFactPreserved(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Insert a fact linked to two cards
	var factID int
	_ = s.Server.Tx.QueryRow(`INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
                        VALUES (1, 1, 'shared fact', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`).Scan(&factID)
	_, _ = s.Server.Tx.Exec(`INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
                       VALUES ($1, 1, 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, factID)
	_, _ = s.Server.Tx.Exec(`INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
                       VALUES ($1, 2, 1, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, factID)

	// Call ExtractSaveCardFacts on card 1
	_, err := s.ExtractSaveCardFacts(1, 1, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert fact still exists
	var exists bool
	err = s.Server.Tx.QueryRow(`SELECT true FROM facts WHERE id=$1`, factID).Scan(&exists)
	if err != nil || !exists {
		t.Fatalf("expected fact to remain, but it was deleted")
	}

	// Assert junctions still exist for both cards
	for _, cardPK := range []int{1, 2} {
		err = s.Server.Tx.QueryRow(`SELECT true FROM fact_card_junction WHERE fact_id=$1 AND card_pk=$2`, factID, cardPK).Scan(&exists)
		if err != nil || !exists {
			t.Fatalf("expected junction for card %d to remain", cardPK)
		}
	}
}

// Test ExtractSaveCardFacts deletes orphaned facts only linked to one card
func TestExtractSaveCardFacts_SingleCardFactDeleted(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Insert a fact linked only to card 1
	var factID int
	_ = s.Server.Tx.QueryRow(`INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
                        VALUES (1, 1, 'orphan fact', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`).Scan(&factID)
	_, _ = s.Server.Tx.Exec(`INSERT INTO fact_card_junction (fact_id, card_pk, user_id, is_origin, created_at, updated_at)
                       VALUES ($1, 1, 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, factID)

	// Call ExtractSaveCardFacts on card 1
	_, err := s.ExtractSaveCardFacts(1, 1, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert fact is deleted
	var exists bool
	err = s.Server.Tx.QueryRow(`SELECT true FROM facts WHERE id=$1`, factID).Scan(&exists)
	if err == nil {
		t.Fatalf("expected fact to be deleted, but it still exists")
	}

	// Assert junction is deleted
	err = s.Server.Tx.QueryRow(`SELECT true FROM fact_card_junction WHERE fact_id=$1 AND card_pk=1`, factID).Scan(&exists)
	if err == nil {
		t.Fatalf("expected junction to be deleted, but it still exists")
	}
}

// Test merging two facts successfully
func TestMergeFactsRoute_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var fact1ID, fact2ID int
	_ = s.Server.Tx.QueryRow(`INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
					   VALUES (1, 1, 'fact 1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`).Scan(&fact1ID)
	_ = s.Server.Tx.QueryRow(`INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
					   VALUES (1, 1, 'fact 2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`).Scan(&fact2ID)

	token, _ := tests.GenerateTestJWT(1)
	payload := fmt.Sprintf(`{"fact1_id": %d, "fact2_id": %d}`, fact1ID, fact2ID)
	req := httptest.NewRequest("POST", "/api/facts/merge", bytes.NewReader([]byte(payload)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/facts/merge", s.JwtMiddleware(s.MergeFactsRoute)).Methods("POST")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Check that fact2 is deleted
	var exists bool
	err := s.Server.Tx.QueryRow("SELECT true FROM facts WHERE id=$1", fact2ID).Scan(&exists)
	if err == nil {
		t.Fatalf("expected fact2 to be deleted but found")
	}
}

// Test merge with same fact IDs should error
func TestMergeFactsRoute_SelfMergeError(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var factID int
	_ = s.Server.Tx.QueryRow(`INSERT INTO facts (user_id, card_pk, fact, created_at, updated_at)
					   VALUES (1, 1, 'fact x', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id`).Scan(&factID)

	token, _ := tests.GenerateTestJWT(1)
	payload := fmt.Sprintf(`{"fact1_id": %d, "fact2_id": %d}`, factID, factID)
	req := httptest.NewRequest("POST", "/api/facts/merge", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/facts/merge", s.JwtMiddleware(s.MergeFactsRoute)).Methods("POST")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
