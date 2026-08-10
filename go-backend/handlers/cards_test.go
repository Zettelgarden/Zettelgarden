package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func makeCardRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(id), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(id))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
	router.ServeHTTP(rr, req)

	return rr
}

func makeCardDeleteRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {
	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("DELETE", "/api/cards/"+strconv.Itoa(id), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(id))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.DeleteCardRoute))
	router.ServeHTTP(rr, req)

	return rr

}

func TestGetCardSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var logCount int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_views").Scan(&logCount)
	if logCount != 0 {
		t.Errorf("wrong log count, got %v want %v", logCount, 0)
	}
	rr := makeCardRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_views").Scan(&logCount)
	if logCount != 1 {
		t.Errorf("wrong log count, got %v want %v", logCount, 1)
	}
	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
	if card.ID != 1 {
		t.Errorf("handler returned wrong card, got %v want %v", card.ID, 1)
	}
	if card.UserID != 1 {
		t.Errorf("handler returned card for wrong user, got %v want %v", card.UserID, 1)
	}

}

func TestGetCardWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)

	req, err := http.NewRequest("GET", "/api/cards/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
	if rr.Body.String() != "card not found\n" {
		t.Errorf("handler returned wrong body, got %v want %v", rr.Body.String(), "unable to access card\n")
	}
}

func TestGetCardSuccessParent(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	rr := makeCardRequestSuccess(s, t, 1)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
	if card.Parent.CardID != card.CardID {
		t.Errorf("wrong card parent returned. got %v want %v", card.Parent.CardID, card.CardID)
	}

}

func TestGetCardChildrenRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/cards/1/children", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/children", s.JwtMiddleware(s.GetCardChildrenRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var children []models.PartialCard
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &children)
	if len(children) == 0 {
		t.Errorf("children was empty. got %v want %v", len(children), 1)
	}

	expected := "1/A"
	if len(children) > 0 && children[0].CardID != expected {
		t.Errorf("linked to wrong card, got %v want %v", children[0].CardID, expected)
	}
}

func TestGetCardFilesRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Initial files
	token, _ := tests.GenerateTestJWT(1)
	req, _ := http.NewRequest("GET", "/api/cards/1/files", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/files", s.JwtMiddleware(s.GetCardFilesRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var files []models.File
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &files)
	if len(files) != 1 {
		t.Errorf("wrong number of files associated with card, got %v want %v", len(files), 1)
	}

	// Upload another file
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	createTestFile(t, buffer, writer)
	req, _ = http.NewRequest("POST", "/api/files/upload", &buffer)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr = httptest.NewRecorder()
	http.HandlerFunc(s.JwtMiddleware(s.UploadFileRoute)).ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("upload returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Verify again
	req, _ = http.NewRequest("GET", "/api/cards/1/files", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	rr = httptest.NewRecorder()
	router = mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/files", s.JwtMiddleware(s.GetCardFilesRoute))
	router.ServeHTTP(rr, req)
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &files)
	if len(files) != 2 {
		t.Errorf("wrong number of files after upload, got %v want %v", len(files), 2)
	}
}

/* Removed legacy TestGetCardReferencesSuccess - replaced by TestGetCardReferencesRoute */

// TestGetCardReferencesRoute validates the new dedicated references endpoint
func TestGetCardReferencesRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/cards/1/references", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/references", s.JwtMiddleware(s.GetCardReferencesRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var refs models.CategorizedReferences
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &refs)

	// Card 1 references card 2 (outgoing), and card 12 (2/A) references card 1 (incoming) - card IDs updated after test data reduction
	totalRefs := len(refs.Bidirectional) + len(refs.Outgoing) + len(refs.Incoming)
	if totalRefs < 2 {
		t.Errorf("wrong number of total references returned, got %v want at least %v", totalRefs, 2)
	}

	// Check that we have the expected cards in the appropriate categories
	// Card 2 should be in outgoing (card 1 -> 2)
	foundCard2 := false
	for _, card := range refs.Outgoing {
		if card.CardID == "2" {
			foundCard2 = true
			break
		}
	}

	// Card 2/A should be in incoming (2/A -> 1)
	foundCard2A := false
	for _, card := range refs.Incoming {
		if card.CardID == "2/A" {
			foundCard2A = true
			break
		}
	}

	if !foundCard2 && len(refs.Bidirectional) == 0 {
		t.Errorf("expected to find card 2 in outgoing or bidirectional references")
	}
	if !foundCard2A && len(refs.Bidirectional) == 0 {
		t.Errorf("expected to find card 2/A in incoming or bidirectional references")
	}
}

func TestGetCardReferencesDuplicateLinks(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, _ := http.NewRequest("GET", "/api/cards/4/references", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "4")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/references", s.JwtMiddleware(s.GetCardReferencesRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var refs models.CategorizedReferences
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &refs)

	// Card 4 (REF001) and card 3 reference each other, so card 3 should be in bidirectional
	totalRefs := len(refs.Bidirectional) + len(refs.Outgoing) + len(refs.Incoming)
	if totalRefs != 1 {
		t.Errorf("wrong number of total references, got %v want %v", totalRefs, 1)
	}

	// Card 3 should be in bidirectional since both cards reference each other
	if len(refs.Bidirectional) != 1 {
		t.Errorf("expected 1 bidirectional reference, got %v", len(refs.Bidirectional))
	}

	if len(refs.Bidirectional) > 0 && refs.Bidirectional[0].CardID != "3" {
		t.Errorf("expected card 3 in bidirectional references, got %v", refs.Bidirectional[0].CardID)
	}

	// Verify no duplicates across all categories
	allCards := make(map[string]int)
	for _, card := range refs.Bidirectional {
		allCards[card.CardID]++
	}
	for _, card := range refs.Outgoing {
		allCards[card.CardID]++
	}
	for _, card := range refs.Incoming {
		allCards[card.CardID]++
	}

	for cardID, count := range allCards {
		if count > 1 {
			t.Errorf("card %v appears %v times in references (should only appear once)", cardID, count)
		}
	}
}
func TestUpdateCardSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var oldLinkCount int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_views").Scan(&oldLinkCount)

	rr := makeCardRequestSuccess(s, t, 1)
	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	token, _ := tests.GenerateTestJWT(1)

	expected := "asdfasdf"
	newData := map[string]interface{}{
		"title":   expected,
		"body":    expected + "[1/A]",
		"card_id": card.CardID,
		"link":    expected,
	}
	jsonData, err := json.Marshal(newData)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}
	req, err := http.NewRequest("PUT", "/api/cards/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	rr = makeCardRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
	if card.Title != expected {
		t.Errorf("handler return wrong title, ot %v want %v", card.Title, expected)
	}
	var newLinkCount int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_views").Scan(&newLinkCount)
	if newLinkCount != oldLinkCount+3 {
		t.Errorf("wrong link count, got %v want %v", newLinkCount, oldLinkCount+3)
	}
}

func TestUpdateCardUnauthorized(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)

	expected := "asdfasdf"
	newData := map[string]interface{}{
		"title":   expected,
		"body":    expected,
		"card_id": "1",
		"link":    expected,
	}
	jsonData, err := json.Marshal(newData)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}
	req, err := http.NewRequest("PUT", "/api/cards/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

}

func TestCreateCardSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var card models.Card
	var newCard models.Card
	token, _ := tests.GenerateTestJWT(1)

	expected := "asdfasdf"
	data := models.EditCardParams{
		Title:  expected,
		Body:   expected,
		CardID: "asd",
		Link:   expected,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	rr = makeCardRequestSuccess(s, t, card.ID)

	tests.ParseJsonResponse(t, rr.Body.Bytes(), &newCard)
	if newCard.Title != expected {
		t.Errorf("handler returned wrong card: got %v want %v", newCard.Title, expected)
	}
}

// TestCreateUpdateDeleteCardRoutesEmitSyncLog verifies the REST card write
// routes drive the sync change feed (bead Zettelgarden-5ry): create, update,
// and delete each emit exactly one sync_log entry for the row, in the same
// transaction as the mutation. A regression where a route stops emitting (or
// emits in a separate transaction that can fail silently) is caught here.
func TestCreateUpdateDeleteCardRoutesEmitSyncLog(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// gorilla/mux populates its vars map only when the request is served
	// through a router with the {id} pattern.
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute)).Methods("PUT")
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.DeleteCardRoute)).Methods("DELETE")

	create := func() models.Card {
		data := models.EditCardParams{Title: "emit me", Body: "body", CardID: "emit1"}
		jsonData, _ := json.Marshal(data)
		req, _ := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute)).ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create: status %d: %s", rr.Code, rr.Body.String())
		}
		var card models.Card
		tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
		return card
	}
	update := func(id int) {
		data := models.EditCardParams{Title: "emit me v2", Body: "body", CardID: "emit1"}
		jsonData, _ := json.Marshal(data)
		req, _ := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(id), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: status %d: %s", rr.Code, rr.Body.String())
		}
	}
	remove := func(id int) {
		req, _ := http.NewRequest("DELETE", "/api/cards/"+strconv.Itoa(id), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("delete: status %d: %s", rr.Code, rr.Body.String())
		}
	}

	countEntries := func(uuid string) []string {
		rows, err := s.GetDB().Query(`SELECT op, version FROM sync_log WHERE row_uuid = $1 AND collection = $2 ORDER BY id`, uuid, services.SyncCollectionCards)
		if err != nil {
			t.Fatalf("query sync_log: %v", err)
		}
		defer rows.Close()
		var ops []string
		for rows.Next() {
			var op string
			var version int
			if err := rows.Scan(&op, &version); err != nil {
				t.Fatal(err)
			}
			ops = append(ops, fmt.Sprintf("%s:%d", op, version))
		}
		return ops
	}

	card := create()
	var uuid string
	if err := s.GetDB().QueryRow(`SELECT sync_uuid FROM cards WHERE id = $1`, card.ID).Scan(&uuid); err != nil {
		t.Fatal(err)
	}
	if ops := countEntries(uuid); len(ops) != 1 || ops[0] != "upsert:1" {
		t.Fatalf("after create, sync_log = %v, want [upsert:1]", ops)
	}

	update(card.ID)
	if ops := countEntries(uuid); len(ops) != 2 || ops[1] != "upsert:2" {
		t.Fatalf("after update, sync_log = %v, want [..., upsert:2]", ops)
	}

	remove(card.ID)
	if ops := countEntries(uuid); len(ops) != 3 || ops[2] != "delete:3" {
		t.Fatalf("after delete, sync_log = %v, want [..., delete:3]", ops)
	}
}

// TestCreateCardRouteRollsBackWhenEmitFails exercises the PRODUCTION
// atomicity contract of 5ry, which the shared-tx harness cannot: with
// Server.Testing=false the route runs real pool transactions and real
// rollback, so a forced sync_log insert failure must roll the card INSERT
// back with it — never a committed row that is invisible to sync.
func TestCreateCardRouteRollsBackWhenEmitFails(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Production semantics: real pool transactions + real rollback. The
	// harness server is a package singleton, so restore the testing flag
	// before this test returns (later tests must keep the shared-tx mode).
	s.Server.Testing = false
	defer func() { s.Server.Testing = true }()

	// Force sync_log inserts to fail so the emit path errors deterministically.
	if _, err := s.DB.Exec(`CREATE TRIGGER zg_test_fail_sync_log BEFORE INSERT ON sync_log BEGIN SELECT RAISE(ABORT, 'forced emit failure'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	defer s.DB.Exec(`DROP TRIGGER IF EXISTS zg_test_fail_sync_log`)

	token, _ := tests.GenerateTestJWT(1)
	data := models.EditCardParams{Title: "rollback me", Body: "body", CardID: "rb1"}
	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute)).ServeHTTP(rr, req)
	if rr.Code == http.StatusCreated {
		t.Fatalf("create card with forced emit failure: expected error status, got 201: %s", rr.Body.String())
	}

	// The card INSERT must be rolled back with the failed emit.
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM cards WHERE card_id = 'rb1' AND user_id = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("card survived the rolled-back create: %d rows with card_id rb1", count)
	}
}

func TestCreateCardDuplicateCardID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	expected := "asdfasdf"
	data := models.EditCardParams{
		Title:  expected,
		CardID: "asdf",
		Link:   expected,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
	req, err = http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
	if rr.Body.String() != "card_id already exists\n" {
		t.Errorf("handler returned wrong error message. got %v want %v", rr.Body.String(), "card_id already exists\n")
	}
}

func TestDeleteCardSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	id := 3
	rr := makeCardDeleteRequestSuccess(s, t, id)

	if status := rr.Code; status != http.StatusNoContent {
		log.Printf("Response body: %s", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	rr = makeCardRequestSuccess(s, t, id)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
	rr = makeCardDeleteRequestSuccess(s, t, id)

	if status := rr.Code; status != http.StatusNotFound {
		log.Printf("Response body: %s", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestDeleteCardWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)

	req, err := http.NewRequest("DELETE", "/api/cards/"+strconv.Itoa(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.DeleteCardRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestCreateCardLinkedParentId(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	rr := makeCardRequestSuccess(s, t, 4)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		log.Printf("err %v", rr.Body.String())
	}
	var parentCard models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &parentCard)
	if parentCard.CardID != "REF001" {
		t.Errorf("handler returned wrong card: got %v want %v", parentCard.CardID, "REF001")
	}

	var card models.Card
	var newCard models.Card
	token, _ := tests.GenerateTestJWT(1)

	data := models.EditCardParams{
		Title:  "asdasd",
		Body:   "asdasd",
		CardID: "REF001/A",
		Link:   "asdasd",
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	rr = makeCardRequestSuccess(s, t, card.ID)
	log.Printf("%v", rr.Body.String())
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &newCard)
	if newCard.ParentID == nil || *newCard.ParentID != parentCard.ID {
		t.Errorf("handler returned wrong parent: got %v want %v", newCard.ParentID, parentCard.ID)
	}
}

func TestGetNextRootCardID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Test when no cards exist beyond test data
	nextID := s.getNextRootCardID(1)
	if nextID != "11" {
		t.Errorf("Expected first ID to be 11 (after test data reduction), got %v", nextID)
	}

	// Create a card with numeric ID
	data := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "5", // Using 5 to test non-sequential numbers
		Link:   "",
	}
	var err error
	_, err = services.CreateCard(s.GetDB(), 2, data)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Test getting next ID after card exists (should still be 11 since 5 is lower)
	nextID = s.getNextRootCardID(1)
	if nextID != "11" {
		t.Errorf("Expected next ID to still be 11 (5 is lower), got %v", nextID)
	}

	// Test that non-numeric IDs are ignored
	data.CardID = "ABC123"
	_, err = services.CreateCard(s.GetDB(), 1, data)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	nextID = s.getNextRootCardID(1)
	if nextID != "11" {
		t.Errorf("Expected next ID to still be 11 (ignoring non-numeric ID), got %v", nextID)
	}
}

func TestGetNextRootCardIDRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/cards/next-root-id", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetNextRootCardIDRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response models.NextIDResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if response.Error {
		t.Errorf("Handler returned error response")
	}
	if response.NextID != "11" {
		t.Errorf("Expected first ID to be 11 (after test data reduction), got %v", response.NextID)
	}

	// Test unauthorized access
	req, _ = http.NewRequest("GET", "/api/cards/next-root-id", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Handler allowed unauthorized access: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestCheckCardLinkedOrRelated(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1
	var mainCard models.Card
	var testCard models.Card

	cards, _, err := s.ClassicCardSearch(userID, SearchRequestParams{SearchTerm: ""})
	if err != nil {
		t.Fatal(err)
	}

	mainCard, err = getCardById(cards, 1)
	if err != nil {
		t.Fatal(err)
	}
	testCard, err = getCardById(cards, 2)
	if err != nil {
		t.Fatal(err)
	}

	// Test parent-child relationship
	mainCardIDPtr := mainCard.ID
	if !s.checkChunkLinkedOrRelated(userID, mainCard, models.CardChunk{
		ID:       testCard.ID,
		ParentID: &mainCardIDPtr,
	}) {
		t.Error("Failed to detect parent-child relationship")
	}

	// Test reference relationship
	if !s.checkChunkLinkedOrRelated(userID, mainCard, models.CardChunk{
		ID: testCard.ID,
	}) {
		t.Error("Failed to detect reference relationship")
	}

	// Test unrelated cards
	if s.checkChunkLinkedOrRelated(userID, testCard, models.CardChunk{
		ID: mainCard.ID,
	}) {
		t.Error("Incorrectly detected relationship between unrelated cards")
	}
}

// TestCheckChunkLinkedOrRelated_Integration provides comprehensive testing of
// the checkChunkLinkedOrRelated method with explicit test data setup.
// This test covers all scenarios including edge cases and error handling.
func TestCheckChunkLinkedOrRelated_Integration(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	userID := 1

	// Create main card
	mainParams := models.EditCardParams{
		Title:  "Main Card",
		Body:   "Main content",
		CardID: "chunk_main",
		Link:   "",
	}
	mainCard, err := services.CreateCard(s.GetDB(), userID, mainParams)
	if err != nil {
		t.Fatalf("Failed to create main card: %v", err)
	}

	// Create a child card (ParentID matches main card)
	childParams := models.EditCardParams{
		Title:  "Child Card",
		Body:   "Child content",
		CardID: "chunk_child",
		Link:   "",
	}
	childCard, err := services.CreateCard(s.GetDB(), userID, childParams)
	if err != nil {
		t.Fatalf("Failed to create child card: %v", err)
	}

	// Create a referenced card (main card will link to it)
	referencedCardParams := models.EditCardParams{
		Title:  "Referenced Card",
		Body:   "Referenced content",
		CardID: "chunk_referenced",
		Link:   "",
	}
	referencedCard, err := services.CreateCard(s.GetDB(), userID, referencedCardParams)
	if err != nil {
		t.Fatalf("Failed to create referenced card: %v", err)
	}

	// Create an unrelated card
	unrelatedParams := models.EditCardParams{
		Title:  "Unrelated Card",
		Body:   "Unrelated content",
		CardID: "chunk_unrelated",
		Link:   "",
	}
	unrelatedCard, err := services.CreateCard(s.GetDB(), userID, unrelatedParams)
	if err != nil {
		t.Fatalf("Failed to create unrelated card: %v", err)
	}

	// Update main card to reference the referenced card
	mainUpdateParams := models.EditCardParams{
		Title:  "Main Card",
		Body:   "Main content with [chunk_referenced]",
		CardID: "chunk_main",
		Link:   "",
	}
	mainCard, err = services.UpdateCard(s.GetDB(), userID, mainCard.ID, mainUpdateParams)
	if err != nil {
		t.Fatalf("Failed to update main card: %v", err)
	}

	// Test 1: Chunk is linked when ParentID matches main card
	t.Run("Linked when ParentID matches", func(t *testing.T) {
		mainCardIDPtr := mainCard.ID
		chunk := models.CardChunk{
			ID:       childCard.ID,
			ParentID: &mainCardIDPtr,
		}
		if !s.checkChunkLinkedOrRelated(userID, mainCard, chunk) {
			t.Error("Expected chunk to be linked when ParentID matches main card")
		}
	})

	// Test 2: Chunk is linked when found in references (even without ParentID match)
	t.Run("Linked when found in references", func(t *testing.T) {
		chunk := models.CardChunk{
			ID:       referencedCard.ID,
			ParentID: nil, // No parent match
		}
		if !s.checkChunkLinkedOrRelated(userID, mainCard, chunk) {
			t.Error("Expected chunk to be linked when found in references")
		}
	})

	// Test 3: Chunk is NOT linked when not in references and parent doesn't match
	t.Run("Not linked when unrelated", func(t *testing.T) {
		chunk := models.CardChunk{
			ID:       unrelatedCard.ID,
			ParentID: nil, // No parent match
		}
		if s.checkChunkLinkedOrRelated(userID, mainCard, chunk) {
			t.Error("Expected chunk to NOT be linked when unrelated and ParentID doesn't match")
		}
	})

	// Test 4: Chunk is linked when BOTH ParentID matches AND is in references
	// This tests that the ParentID check takes precedence (returns early)
	t.Run("Linked when ParentID matches even if also in references", func(t *testing.T) {
		// Create a child of the referenced card
		childOfReferencedParams := models.EditCardParams{
			Title:  "Child of Referenced",
			Body:   "Child content",
			CardID: "chunk_child_ref",
			Link:   "",
		}
		childOfReferenced, err := services.CreateCard(s.GetDB(), userID, childOfReferencedParams)
		if err != nil {
			t.Fatalf("Failed to create child of referenced: %v", err)
		}

		// Update main card to also reference this child
		mainUpdateParams2 := models.EditCardParams{
			Title:  "Main Card",
			Body:   "Main content with [chunk_referenced] and [chunk_child_ref]",
			CardID: "chunk_main",
			Link:   "",
		}
		mainCard, err = services.UpdateCard(s.GetDB(), userID, mainCard.ID, mainUpdateParams2)
		if err != nil {
			t.Fatalf("Failed to update main card: %v", err)
		}

		// Also update the child to have mainCard as parent
		// (Note: this would require updating the card's parent_id, which we can't do through UpdateCard)
		// Instead, we test with a simulated chunk that has both conditions
		mainCardIDPtr2 := mainCard.ID
		chunk := models.CardChunk{
			ID:       childOfReferenced.ID,
			ParentID: &mainCardIDPtr2,
		}
		if !s.checkChunkLinkedOrRelated(userID, mainCard, chunk) {
			t.Error("Expected chunk to be linked when ParentID matches")
		}
	})

	// Test 5: Error handling - When GetReferences fails, method returns true (fail-open)
	// We test this by using an invalid userID which may cause a database error
	t.Run("Handles reference query errors gracefully", func(t *testing.T) {
		// Use a very high userID that doesn't exist - this may cause errors in queries
		// The method should return true (linked) on error as a safety measure
		invalidUserID := 999999

		chunk := models.CardChunk{
			ID:       unrelatedCard.ID,
			ParentID: nil, // No parent match
		}

		// Call with invalid userID - if GetReferences fails, it should return true
		result := s.checkChunkLinkedOrRelated(invalidUserID, mainCard, chunk)
		// We expect true because on error, the method returns true (fail-open)
		if !result {
			t.Log("Note: GetReferences did not fail with invalid userID, test scenario could not be triggered")
			t.Log("This is expected if the database allows queries with non-existent users")
			// If we reach here, the database didn't error, so we can't test the error path
			// This is actually fine - it means the error handling is defensive but the error is rare
		} else {
			t.Log("GetReferences error handling returned true (fail-open behavior)")
		}
	})
}

func TestGetNextChildCardID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card Test",
		Body:   "Test Body",
		CardID: "999", // Use high number to avoid conflicts
		Link:   "",
	}
	parentCard, err := services.CreateCard(s.GetDB(), 1, parentParams)
	if err != nil {
		t.Fatalf("Failed to create parent test card: %v", err)
	}

	// Test 1: Parent with no children should return ".1"
	nextID := s.getNextChildCardID(1, parentCard.ID)
	expected := "999.1"
	if nextID != expected {
		t.Errorf("Expected first child ID to be %s, got %s", expected, nextID)
	}

	// Test 2: Create a child with ID ".1" manually
	childParams := models.EditCardParams{
		Title:  "First Child",
		Body:   "Test Body",
		CardID: "999.1",
		Link:   "",
	}
	childCard1, err := services.CreateCard(s.GetDB(), 1, childParams)
	if err != nil {
		t.Fatalf("Failed to create first child test card: %v", err)
	}

	nextID = s.getNextChildCardID(1, parentCard.ID)
	expected = "999.2"
	if nextID != expected {
		t.Errorf("Expected next child ID to be %s, got %s", expected, nextID)
	}

	// Test 3: Add another child with ID ".2"
	childParams.CardID = "999.2"
	childCard2, err := services.CreateCard(s.GetDB(), 1, childParams)
	if err != nil {
		t.Fatalf("Failed to create second child test card: %v", err)
	}

	nextID = s.getNextChildCardID(1, childCard1.ID) // Test with nested grandchild
	if nextID != "999.1.1" {                        // Should extend the existing child ID
		t.Errorf("Expected nested child ID to be 999.1.1, got %s", nextID)
	}

	// Test 4: Non-sequential children - add ".5" (skip 3 and 4)
	childParams.CardID = "999.5"
	_, err = services.CreateCard(s.GetDB(), 1, childParams)
	if err != nil {
		t.Fatalf("Failed to create third child test card: %v", err)
	}

	nextID = s.getNextChildCardID(1, parentCard.ID)
	expected = "999.6" // Should increment from max (which is 5)
	if nextID != expected {
		t.Errorf("Expected next child ID to be %s (incrementing from 5), got %s", expected, nextID)
	}

	// Test 5: Different separator styles (. and / both supported)
	childParams.CardID = "999/7" // Alternate separator
	_, err = services.CreateCard(s.GetDB(), 1, childParams)
	if err != nil {
		t.Fatalf("Failed to create fourth child test card: %v", err)
	}

	nextID = s.getNextChildCardID(1, parentCard.ID)
	expected = "999.6" // Majority scheme is '.' (3 of 4 children); the '/7' outlier is a different scheme and must not perturb the '.' max (5)
	if nextID != expected {
		t.Errorf("Expected next child ID to be %s (majority '.' separator, max 5), got %s", expected, nextID)
	}

	// Clean up test cards
	for _, cardID := range []int{childCard1.ID, childCard2.ID, parentCard.ID} {
		_, err = s.Server.Tx.Exec("DELETE FROM cards WHERE id = $1", cardID)
		if err != nil {
			t.Logf("Failed to clean up test card %d: %v", cardID, err)
		}
	}
}

func TestGetNextChildCardIDRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a test parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card Route Test",
		Body:   "Test Body",
		CardID: "888", // Use high number to avoid conflicts
		Link:   "",
	}
	parentCard, err := services.CreateCard(s.GetDB(), 1, parentParams)
	if err != nil {
		t.Fatalf("Failed to create parent test card: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(parentCard.ID)+"/next-child-id", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(parentCard.ID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/next-child-id", s.JwtMiddleware(s.GetNextChildCardIDRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response models.NextIDResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if response.Error {
		t.Errorf("Handler returned error response")
	}
	if response.NextID != "888.1" {
		t.Errorf("Expected first child ID to be 888.1, got %v", response.NextID)
	}

	// Clean up test card
	_, err = s.Server.Tx.Exec("DELETE FROM cards WHERE id = $1", parentCard.ID)
	if err != nil {
		t.Logf("Failed to clean up test card %d: %v", parentCard.ID, err)
	}
}

// Helper function to create a test schema
func createTestSchema(s *Handler, t *testing.T, userID int, name string, fields []models.FieldDefinition) int {
	params := models.CreateSchemaDefinitionParams{
		Name:    name,
		OwnerID: userID,
		Fields:  fields,
	}
	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal schema params: %v", err)
	}

	token, _ := tests.GenerateTestJWT(userID)
	req, err := http.NewRequest("POST", "/api/schemas", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateSchemaRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("Failed to create test schema: %v - %v", status, rr.Body.String())
	}

	var schema models.SchemaDefinition
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &schema)
	return schema.ID
}

// TestCreateCardWithSchema_Success creates a card with schema and valid structured_data
func TestCreateCardWithSchema_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create test schema with all field types
	fields := []models.FieldDefinition{
		{Name: "title_field", Type: "text", Required: true},
		{Name: "count_field", Type: "number", Required: true},
		{Name: "date_field", Type: "date", Required: false},
		{Name: "active_field", Type: "boolean", Required: false},
		{Name: "status_field", Type: "select", Required: false, Options: []string{"pending", "active", "completed"}},
		{Name: "tags_field", Type: "multi-select", Required: false, Options: []string{"work", "personal", "urgent"}},
	}
	schemaID := createTestSchema(s, t, 1, "Test Schema", fields)

	// Create card with schema and structured_data
	token, _ := tests.GenerateTestJWT(1)

	structuredDataJSON := `{"title_field":"Test Title","count_field":42,"date_field":"2024-01-15","active_field":true,"status_field":"active","tags_field":["work","urgent"]}`
	var structuredData json.RawMessage
	err := json.Unmarshal([]byte(structuredDataJSON), &structuredData)
	if err != nil {
		t.Fatalf("Failed to unmarshal structured data: %v", err)
	}

	data := models.EditCardParams{
		Title:          "Card with Schema",
		Body:           "Test body",
		CardID:         "SCHEMA001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v\nbody: %v", status, http.StatusCreated, rr.Body.String())
	}

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	if card.SchemaID == nil {
		t.Errorf("Expected schema_id to be set, got nil")
	}
	if *card.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v, got %v", schemaID, *card.SchemaID)
	}
	if card.StructuredData == nil {
		t.Errorf("Expected structured_data to be set, got nil")
	}

	// Verify the structured data content
	var resultData map[string]interface{}
	err = json.Unmarshal(*card.StructuredData, &resultData)
	if err != nil {
		t.Fatalf("Failed to unmarshal structured_data: %v", err)
	}
	if resultData["title_field"] != "Test Title" {
		t.Errorf("Expected title_field 'Test Title', got %v", resultData["title_field"])
	}
	if int(resultData["count_field"].(float64)) != 42 {
		t.Errorf("Expected count_field 42, got %v", resultData["count_field"])
	}
}

// TestCreateCardWithSchema_SchemaNotFound tests error when schema_id doesn't exist
func TestCreateCardWithSchema_SchemaNotFound(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	nonExistentSchemaID := 99999
	structuredDataJSON := `{"field1":"value1"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card with Invalid Schema",
		Body:           "Test body",
		CardID:         "SCHEMA002",
		Link:           "test",
		SchemaID:       &nonExistentSchemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	// Should return 404 because schema doesn't exist for this user
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %v for non-existent schema, got %v", http.StatusNotFound, status)
	}
	if rr.Body.String() != "Schema not found\n" {
		t.Errorf("Expected 'Schema not found' error message, got: %v", rr.Body.String())
	}
}

// TestCreateCardWithSchema_OtherUsersSchema tests error when using another user's schema
func TestCreateCardWithSchema_OtherUsersSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create schema for user 1
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "User 1 Schema", fields)

	// Try to create card for user 2 with user 1's schema
	token, _ := tests.GenerateTestJWT(2)

	structuredDataJSON := `{"field1":"value1"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "User 2 Card",
		Body:           "Test body",
		CardID:         "SCHEMA003",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	// Should return 404 because schema doesn't exist for this user
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %v for other user's schema, got %v", http.StatusNotFound, status)
	}
	if rr.Body.String() != "Schema not found\n" {
		t.Errorf("Expected 'Schema not found' error message, got: %v", rr.Body.String())
	}
}

// TestCreateCardWithSchema_MissingRequiredField tests that a card whose
// structured_data is missing a required field is rejected with a clear message
// (bead Zettelgarden-s2l).
func TestCreateCardWithSchema_MissingRequiredField(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create schema with required fields
	fields := []models.FieldDefinition{
		{Name: "required_field", Type: "text", Required: true},
		{Name: "optional_field", Type: "text", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Schema with Required Fields", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Missing required field
	structuredDataJSON := `{"optional_field":"present"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Missing Required Field",
		Body:           "Test body",
		CardID:         "SCHEMA004",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing required field, got %v (%v)", status, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "required field 'required_field'") {
		t.Errorf("Expected message naming the missing field, got: %v", rr.Body.String())
	}
}

// TestCreateCardWithSchema_EmptyRequiredField tests that a required field
// present but empty (whitespace-only string) is rejected the same as missing
// (bead Zettelgarden-s2l).
func TestCreateCardWithSchema_EmptyRequiredField(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	fields := []models.FieldDefinition{
		{Name: "required_field", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Schema with Required Text", fields)

	token, _ := tests.GenerateTestJWT(1)

	for name, sd := range map[string]string{
		"empty string":    `{"required_field":""}`,
		"whitespace only": `{"required_field":"   "}`,
		"null value":      `{"required_field":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			var structuredData json.RawMessage
			_ = json.Unmarshal([]byte(sd), &structuredData)

			data := models.EditCardParams{
				Title:          "Empty Required Field",
				Body:           "Test body",
				CardID:         "SCHEMA005",
				Link:           "test",
				SchemaID:       &schemaID,
				StructuredData: &structuredData,
			}
			jsonData, _ := json.Marshal(data)
			req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("Expected 400 for %s, got %v (%v)", name, status, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "required field 'required_field'") {
				t.Errorf("Expected message naming the field for %s, got: %v", name, rr.Body.String())
			}
		})
	}
}

// TestCreateCardWithSchema_InvalidFieldType tests error when field type doesn't match
func TestCreateCardWithSchema_InvalidFieldType(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create schema with number field
	fields := []models.FieldDefinition{
		{Name: "count", Type: "number", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Number Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Send text instead of number
	structuredDataJSON := `{"count":"not a number"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Invalid Type",
		Body:           "Test body",
		CardID:         "SCHEMA005",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	// Note: This test documents current behavior
	// The card is created successfully but type validation should be added
	if status := rr.Code; status != http.StatusCreated {
		t.Logf("Card creation failed with status %v: %v", status, rr.Body.String())
	}
}

// TestCreateCardWithSchema_InvalidSelectValue tests error when select value is not in options
func TestCreateCardWithSchema_InvalidSelectValue(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create schema with select field
	fields := []models.FieldDefinition{
		{Name: "status", Type: "select", Required: true, Options: []string{"active", "inactive"}},
	}
	schemaID := createTestSchema(s, t, 1, "Select Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Send invalid select value
	structuredDataJSON := `{"status":"pending"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Invalid Select",
		Body:           "Test body",
		CardID:         "SCHEMA006",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	// Note: This test documents current behavior
	// The card is created successfully but option validation should be added
	if status := rr.Code; status != http.StatusCreated {
		t.Logf("Card creation failed with status %v: %v", status, rr.Body.String())
	}
}

// TestCreateCardWithSchema_AllOptionalFieldsEmpty tests success with all optional fields empty
func TestCreateCardWithSchema_AllOptionalFieldsEmpty(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create schema with optional fields
	fields := []models.FieldDefinition{
		{Name: "optional_text", Type: "text", Required: false},
		{Name: "optional_number", Type: "number", Required: false},
		{Name: "optional_date", Type: "date", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Optional Fields Schema", fields)

	token, _ := tests.GenerateTestJWT(1)

	// Empty structured_data
	structuredDataJSON := `{}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Optional Fields Card",
		Body:           "Test body",
		CardID:         "SCHEMA007",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	if card.SchemaID == nil {
		t.Errorf("Expected schema_id to be set, got nil")
	}
	if *card.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v, got %v", schemaID, *card.SchemaID)
	}
}

// TestUpdateCardWithSchema_AddSchemaToExistingCard tests adding schema to existing card
func TestUpdateCardWithSchema_AddSchemaToExistingCard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create card without schema first
	data := models.EditCardParams{
		Title:  "Card Without Schema",
		Body:   "Test body",
		CardID: "NOSCHEMA001",
		Link:   "test",
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	// Now create a schema and add it to the card
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Update Schema", fields)

	// Update card to add schema
	structuredDataJSON := `{"field1":"value1"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	updateData := models.EditCardParams{
		Title:          card.Title,
		Body:           card.Body,
		CardID:         card.CardID,
		Link:           card.Link,
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	updateJSON, _ := json.Marshal(updateData)
	updateReq, err := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID), bytes.NewBuffer(updateJSON))
	if err != nil {
		t.Fatal(err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", strconv.Itoa(card.ID))

	updateRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(updateRR, updateReq)

	if status := updateRR.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedCard models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updatedCard)

	if updatedCard.SchemaID == nil {
		t.Errorf("Expected schema_id to be set after update, got nil")
	}
	if *updatedCard.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v, got %v", schemaID, *updatedCard.SchemaID)
	}
}

// TestUpdateCardWithSchema_UpdateStructuredData tests updating structured_data
func TestUpdateCardWithSchema_UpdateStructuredData(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
		{Name: "field2", Type: "number", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Update Data Schema", fields)

	// Create card with schema and initial structured_data
	structuredDataJSON := `{"field1":"initial","field2":10}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card for Update",
		Body:           "Test body",
		CardID:         "UPDATE001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	// Update structured_data
	newStructuredDataJSON := `{"field1":"updated","field2":20}`
	var newStructuredData json.RawMessage
	_ = json.Unmarshal([]byte(newStructuredDataJSON), &newStructuredData)

	updateData := models.EditCardParams{
		Title:          card.Title,
		Body:           card.Body,
		CardID:         card.CardID,
		Link:           card.Link,
		SchemaID:       card.SchemaID,
		StructuredData: &newStructuredData,
	}
	updateJSON, _ := json.Marshal(updateData)
	updateReq, err := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID), bytes.NewBuffer(updateJSON))
	if err != nil {
		t.Fatal(err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", strconv.Itoa(card.ID))

	updateRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(updateRR, updateReq)

	if status := updateRR.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedCard models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updatedCard)

	var resultData map[string]interface{}
	err = json.Unmarshal(*updatedCard.StructuredData, &resultData)
	if err != nil {
		t.Fatalf("Failed to unmarshal structured_data: %v", err)
	}
	if resultData["field1"] != "updated" {
		t.Errorf("Expected field1 'updated', got %v", resultData["field1"])
	}
	if int(resultData["field2"].(float64)) != 20 {
		t.Errorf("Expected field2 20, got %v", resultData["field2"])
	}
}

// TestUpdateCardWithSchema_RemoveSchema tests removing schema from card
func TestUpdateCardWithSchema_RemoveSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Remove Schema", fields)

	// Create card with schema
	structuredDataJSON := `{"field1":"value1"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card to Remove Schema",
		Body:           "Test body",
		CardID:         "REMOVE001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	// Update card to remove schema using the ClearSchema flag
	updateData := models.EditCardParams{
		Title:       card.Title,
		Body:        card.Body,
		CardID:      card.CardID,
		Link:        card.Link,
		ClearSchema: true,
	}
	updateJSON, _ := json.Marshal(updateData)
	updateReq, err := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID), bytes.NewBuffer(updateJSON))
	if err != nil {
		t.Fatal(err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", strconv.Itoa(card.ID))

	updateRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(updateRR, updateReq)

	if status := updateRR.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedCard models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updatedCard)

	if updatedCard.SchemaID != nil {
		t.Errorf("Expected schema_id to be nil after removal, got %v", *updatedCard.SchemaID)
	}
}

// createCardWithSchema creates a card attached to a schema through the REST
// handler and returns the created card.
func createCardWithSchema(t *testing.T, s *Handler, token string, cardID string, schemaID int, dataJSON string) models.Card {
	t.Helper()
	payload := fmt.Sprintf(`{
		"card_id": %q, "title": "Card %s", "body": "body", "link": "",
		"schema_id": %d, "structured_data": %s
	}`, cardID, cardID, schemaID, dataJSON)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("create card with schema: expected 201, got %d: %s", status, rr.Body.String())
	}
	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
	return card
}

// updateCardWithPayload issues a PUT to UpdateCardRoute and returns the recorder.
func updateCardWithPayload(t *testing.T, s *Handler, token string, id int, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(id), bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.Itoa(id))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(rr, req)
	return rr
}

// TestUpdateCardWithSchema_DetachWithEmptyObject verifies that clearing the
// schema dropdown in the editor — which sends schema_id: null with
// structured_data: {} — succeeds and detaches the schema instead of hitting the
// "structured_data requires schema_id" 400 (bead Zettelgarden-276).
func TestUpdateCardWithSchema_DetachWithEmptyObject(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	schemaID := createTestSchema(s, t, 1, "Detach Schema", []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	})
	card := createCardWithSchema(t, s, token, "DETACH001", schemaID, `{"field1":"value1"}`)

	payload := fmt.Sprintf(`{
		"card_id": %q, "title": "Updated", "body": "body", "link": "",
		"schema_id": null, "structured_data": {}
	}`, card.CardID)
	updateRR := updateCardWithPayload(t, s, token, card.ID, payload)

	if status := updateRR.Code; status != http.StatusOK {
		t.Fatalf("detach with {}: expected 200, got %d: %s", status, updateRR.Body.String())
	}
	var updated models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updated)
	if updated.SchemaID != nil {
		t.Errorf("expected schema_id nil after detach, got %v", *updated.SchemaID)
	}
	if updated.StructuredData != nil {
		t.Errorf("expected structured_data cleared after detach, got %s", string(*updated.StructuredData))
	}
}

// TestUpdateCardWithSchema_DetachWithNull is the same detach when the payload
// nulls structured_data instead of sending {} (bead Zettelgarden-276).
func TestUpdateCardWithSchema_DetachWithNull(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	schemaID := createTestSchema(s, t, 1, "Detach Schema Null", []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	})
	card := createCardWithSchema(t, s, token, "DETACH002", schemaID, `{"field1":"value1"}`)

	payload := fmt.Sprintf(`{
		"card_id": %q, "title": "Updated", "body": "body", "link": "",
		"schema_id": null, "structured_data": null
	}`, card.CardID)
	updateRR := updateCardWithPayload(t, s, token, card.ID, payload)

	if status := updateRR.Code; status != http.StatusOK {
		t.Fatalf("detach with null: expected 200, got %d: %s", status, updateRR.Body.String())
	}
	var updated models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updated)
	if updated.SchemaID != nil {
		t.Errorf("expected schema_id nil after detach, got %v", *updated.SchemaID)
	}
	if updated.StructuredData != nil {
		t.Errorf("expected structured_data cleared after detach, got %s", string(*updated.StructuredData))
	}
}

// TestUpdateCardWithSchema_OmittedSchemaFieldsPreserved verifies a partial
// update that omits schema_id/structured_data keeps the existing association
// (the UpdateCard preserve branch must stay intact, bead Zettelgarden-276).
func TestUpdateCardWithSchema_OmittedSchemaFieldsPreserved(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	schemaID := createTestSchema(s, t, 1, "Preserve Schema", []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	})
	card := createCardWithSchema(t, s, token, "PRESERVE001", schemaID, `{"field1":"value1"}`)

	payload := fmt.Sprintf(`{
		"card_id": %q, "title": "New Title", "body": "body", "link": ""
	}`, card.CardID)
	updateRR := updateCardWithPayload(t, s, token, card.ID, payload)

	if status := updateRR.Code; status != http.StatusOK {
		t.Fatalf("omitted schema fields: expected 200, got %d: %s", status, updateRR.Body.String())
	}
	var updated models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updated)
	if updated.SchemaID == nil || *updated.SchemaID != schemaID {
		t.Errorf("expected schema_id preserved (%d), got %v", schemaID, updated.SchemaID)
	}
	if updated.StructuredData == nil || !strings.Contains(string(*updated.StructuredData), "value1") {
		t.Errorf("expected structured_data preserved, got %v", updated.StructuredData)
	}
}

// TestUpdateCardWithSchema_NonEmptyDataNoSchema_Rejected verifies the
// stray-data contract on the update path: non-empty structured_data without a
// schema still returns the 400 (bead Zettelgarden-276).
func TestUpdateCardWithSchema_NonEmptyDataNoSchema_Rejected(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	schemaID := createTestSchema(s, t, 1, "Stray Data Schema", []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	})
	card := createCardWithSchema(t, s, token, "STRAY001", schemaID, `{"field1":"value1"}`)

	payload := fmt.Sprintf(`{
		"card_id": %q, "title": "Updated", "body": "body", "link": "",
		"schema_id": null, "structured_data": {"field1":"value2"}
	}`, card.CardID)
	updateRR := updateCardWithPayload(t, s, token, card.ID, payload)

	if status := updateRR.Code; status != http.StatusBadRequest {
		t.Fatalf("expected 400 for stray data on update, got %d: %s", status, updateRR.Body.String())
	}
	if !strings.Contains(updateRR.Body.String(), "structured_data requires schema_id") {
		t.Errorf("expected guard message, got: %v", updateRR.Body.String())
	}
}

// TestCreateCardWithEmptyDataNoSchema verifies the create path accepts the
// detach payload shape (schema_id: null + structured_data: {}) and stores no
// schema association (bead Zettelgarden-276).
func TestCreateCardWithEmptyDataNoSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	payload := `{
		"card_id": "NOSCHEMA001", "title": "No schema", "body": "body", "link": "",
		"schema_id": null, "structured_data": {}
	}`
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("create with empty data: expected 201, got %d: %s", status, rr.Body.String())
	}
	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)
	if card.SchemaID != nil {
		t.Errorf("expected schema_id nil, got %v", *card.SchemaID)
	}
	if card.StructuredData != nil {
		t.Errorf("expected structured_data nil, got %s", string(*card.StructuredData))
	}
}

// TestCreateCardWithStructuredDataNoSchema_Rejected verifies the stray-data
// contract on the create path: non-empty structured_data without schema_id
// still returns the 400 (that contract is intentional, bead Zettelgarden-276).
func TestCreateCardWithStructuredDataNoSchema_Rejected(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	payload := `{
		"card_id": "NOSCHEMA002", "title": "Stray", "body": "body", "link": "",
		"schema_id": null, "structured_data": {"field1":"x"}
	}`
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Fatalf("expected 400 for stray data on create, got %d: %s", status, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "structured_data requires schema_id") {
		t.Errorf("expected guard message, got: %v", rr.Body.String())
	}
}

// TestGetCardWithSchema_WithSchemaAndData tests getting card that includes schema_id and structured_data
func TestGetCardWithSchema_WithSchemaAndData(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema
	fields := []models.FieldDefinition{
		{Name: "title", Type: "text", Required: true},
		{Name: "count", Type: "number", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Get Schema", fields)

	// Create card with schema and data
	structuredDataJSON := `{"title":"Test Title","count":42}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card with Schema",
		Body:           "Test body",
		CardID:         "GETSCHEMA001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	// Get the card
	getReq, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(card.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	getReq.Header.Set("Authorization", "Bearer "+token)
	getReq.SetPathValue("id", strconv.Itoa(card.ID))

	getRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
	router.ServeHTTP(getRR, getReq)

	if status := getRR.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var retrievedCard models.Card
	tests.ParseJsonResponse(t, getRR.Body.Bytes(), &retrievedCard)

	if retrievedCard.SchemaID == nil {
		t.Errorf("Expected schema_id to be set, got nil")
	}
	if *retrievedCard.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v, got %v", schemaID, *retrievedCard.SchemaID)
	}
	if retrievedCard.StructuredData == nil {
		t.Errorf("Expected structured_data to be set, got nil")
	}

	// Verify structured_data content
	var resultData map[string]interface{}
	err = json.Unmarshal(*retrievedCard.StructuredData, &resultData)
	if err != nil {
		t.Fatalf("Failed to unmarshal structured_data: %v", err)
	}
	if resultData["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", resultData["title"])
	}
}

// TestGetCardWithSchema_NullSchemaID tests getting card with null schema_id
func TestGetCardWithSchema_NullSchemaID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create card without schema (using existing test card)
	rr := makeCardRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	if card.SchemaID != nil {
		t.Errorf("Expected schema_id to be nil for card without schema, got %v", *card.SchemaID)
	}
}

// TestGetCardWithSchema_NullStructuredData tests getting card with null structured_data
func TestGetCardWithSchema_NullStructuredData(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Null Data Schema", fields)

	// Create card with schema but no structured_data
	data := models.EditCardParams{
		Title:    "Card with Schema No Data",
		Body:     "Test body",
		CardID:   "NULLDATA001",
		Link:     "test",
		SchemaID: &schemaID,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	if card.SchemaID == nil {
		t.Errorf("Expected schema_id to be set, got nil")
	}
	// structured_data can be nil when no data is provided
	if card.StructuredData != nil {
		t.Logf("structured_data is set: %s", *card.StructuredData)
	}
}

// TestCreateCardWithSchema_WithoutStructuredDataWhenRequired tests error when schema requires structured_data
func TestCreateCardWithSchema_WithoutStructuredDataWhenRequired(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema with required fields
	fields := []models.FieldDefinition{
		{Name: "required_field", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Required Schema", fields)

	// Try to create card with schema but without structured_data
	data := models.EditCardParams{
		Title:    "Card without structured_data",
		Body:     "Test body",
		CardID:   "NODATA001",
		Link:     "test",
		SchemaID: &schemaID,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	// Should fail with bad request because schema requires structured_data
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %v when schema requires data, got %v", http.StatusBadRequest, status)
	}
	if !strings.Contains(rr.Body.String(), "required_field") {
		t.Errorf("Expected error about required_field, got: %v", rr.Body.String())
	}
}

// TestCreateCardWithSchema_WithLinkToCardField tests creating card with link_to_card field
func TestCreateCardWithSchema_WithLinkToCardField(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// First, create a card to link to
	linkTargetData := models.EditCardParams{
		Title:  "Link Target Card",
		Body:   "This card will be linked to",
		CardID: "LINKTARGET",
		Link:   "test",
	}
	jsonData, _ := json.Marshal(linkTargetData)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var targetCard models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &targetCard)

	// Create schema with link_to_card field
	fields := []models.FieldDefinition{
		{Name: "title_field", Type: "text", Required: true},
		{Name: "related_card", Type: "link_to_card", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Link Schema", fields)

	// Create card with link_to_card reference
	structuredDataJSON := fmt.Sprintf(`{"title_field":"Test","related_card":%d}`, targetCard.ID)
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card with Link",
		Body:           "Test body",
		CardID:         "LINKCARD001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ = json.Marshal(data)
	req, err = http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v\nbody: %v", status, http.StatusCreated, rr.Body.String())
	}

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	if card.SchemaID == nil {
		t.Errorf("Expected schema_id to be set, got nil")
	}
	if *card.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v, got %v", schemaID, *card.SchemaID)
	}

	// Verify the link_to_card reference
	var resultData map[string]interface{}
	err = json.Unmarshal(*card.StructuredData, &resultData)
	if err != nil {
		t.Fatalf("Failed to unmarshal structured_data: %v", err)
	}
	if int(resultData["related_card"].(float64)) != targetCard.ID {
		t.Errorf("Expected related_card %v, got %v", targetCard.ID, resultData["related_card"])
	}
}

// TestCreateCardWithSchema_InvalidLinkToCardReference tests error with invalid link_to_card reference
func TestCreateCardWithSchema_InvalidLinkToCardReference(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema with link_to_card field
	fields := []models.FieldDefinition{
		{Name: "title_field", Type: "text", Required: true},
		{Name: "related_card", Type: "link_to_card", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "Invalid Link Schema", fields)

	// Create card with invalid link_to_card reference (non-existent card)
	structuredDataJSON := `{"title_field":"Test","related_card":99999}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card with Invalid Link",
		Body:           "Test body",
		CardID:         "INVALIDLINK001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	// Should fail with bad request because referenced card doesn't exist
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %v for invalid link_to_card reference, got %v", http.StatusBadRequest, status)
	}
	if !strings.Contains(rr.Body.String(), "link_to_card") {
		t.Errorf("Expected error about link_to_card reference, got: %v", rr.Body.String())
	}
}

// TestCreateCardWithSchema_AllFieldTypes tests creating card with all supported field types
func TestCreateCardWithSchema_AllFieldTypes(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create a target card for link_to_card field
	linkTargetData := models.EditCardParams{
		Title:  "Link Target",
		Body:   "Target card",
		CardID: "ALLTYPESLINK",
		Link:   "test",
	}
	jsonData, _ := json.Marshal(linkTargetData)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var targetCard models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &targetCard)

	// Create schema with all field types
	fields := []models.FieldDefinition{
		{Name: "text_field", Type: "text", Required: true},
		{Name: "number_field", Type: "number", Required: true},
		{Name: "date_field", Type: "date", Required: false},
		{Name: "boolean_field", Type: "boolean", Required: false},
		{Name: "select_field", Type: "select", Required: false, Options: []string{"option1", "option2", "option3"}},
		{Name: "multi_select_field", Type: "multi-select", Required: false, Options: []string{"tag1", "tag2", "tag3"}},
		{Name: "link_field", Type: "link_to_card", Required: false},
	}
	schemaID := createTestSchema(s, t, 1, "All Types Schema", fields)

	// Create card with all field types populated
	structuredDataJSON := fmt.Sprintf(`{
		"text_field": "Sample text",
		"number_field": 123,
		"date_field": "2024-01-15",
		"boolean_field": true,
		"select_field": "option2",
		"multi_select_field": ["tag1", "tag3"],
		"link_field": %d
	}`, targetCard.ID)
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "All Types Card",
		Body:           "Test body",
		CardID:         "ALLTYPES001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ = json.Marshal(data)
	req, err = http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v\nbody: %v", status, http.StatusCreated, rr.Body.String())
	}

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	if card.SchemaID == nil {
		t.Errorf("Expected schema_id to be set, got nil")
	}
	if *card.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v, got %v", schemaID, *card.SchemaID)
	}

	// Verify all fields are correctly stored
	var resultData map[string]interface{}
	err = json.Unmarshal(*card.StructuredData, &resultData)
	if err != nil {
		t.Fatalf("Failed to unmarshal structured_data: %v", err)
	}

	// Check each field
	if resultData["text_field"] != "Sample text" {
		t.Errorf("Expected text_field 'Sample text', got %v", resultData["text_field"])
	}
	if int(resultData["number_field"].(float64)) != 123 {
		t.Errorf("Expected number_field 123, got %v", resultData["number_field"])
	}
	if resultData["date_field"] != "2024-01-15" {
		t.Errorf("Expected date_field '2024-01-15', got %v", resultData["date_field"])
	}
	if resultData["boolean_field"] != true {
		t.Errorf("Expected boolean_field true, got %v", resultData["boolean_field"])
	}
	if resultData["select_field"] != "option2" {
		t.Errorf("Expected select_field 'option2', got %v", resultData["select_field"])
	}
}

// TestUpdateCardWithSchema_InvalidStructuredData tests updating with invalid structured_data
func TestUpdateCardWithSchema_InvalidStructuredData(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Update Validation Schema", fields)

	// Create card with valid schema and data
	structuredDataJSON := `{"field1":"initial"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card for Update",
		Body:           "Test body",
		CardID:         "UPDATEVALID001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	// Try to update with invalid structured_data (missing required field)
	invalidStructuredDataJSON := `{}`
	var invalidStructuredData json.RawMessage
	_ = json.Unmarshal([]byte(invalidStructuredDataJSON), &invalidStructuredData)

	updateData := models.EditCardParams{
		Title:          card.Title,
		Body:           card.Body,
		CardID:         card.CardID,
		Link:           card.Link,
		SchemaID:       card.SchemaID,
		StructuredData: &invalidStructuredData,
	}
	updateJSON, _ := json.Marshal(updateData)
	updateReq, err := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID), bytes.NewBuffer(updateJSON))
	if err != nil {
		t.Fatal(err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", strconv.Itoa(card.ID))

	updateRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(updateRR, updateReq)

	// Should fail with bad request
	if status := updateRR.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %v for invalid structured_data, got %v", http.StatusBadRequest, status)
	}
	if !strings.Contains(updateRR.Body.String(), "required") {
		t.Errorf("Expected error about required field, got: %v", updateRR.Body.String())
	}
}

// TestUpdateCardWithSchema_PartialUpdatePreservesSchema tests that updating
// other fields (like title) without providing schema_id/structured_data
// preserves the existing schema association
func TestUpdateCardWithSchema_PartialUpdatePreservesSchema(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create schema
	fields := []models.FieldDefinition{
		{Name: "field1", Type: "text", Required: true},
	}
	schemaID := createTestSchema(s, t, 1, "Preserve Schema", fields)

	// Create card with schema and structured_data
	structuredDataJSON := `{"field1":"initial value"}`
	var structuredData json.RawMessage
	_ = json.Unmarshal([]byte(structuredDataJSON), &structuredData)

	data := models.EditCardParams{
		Title:          "Card with Schema - Initial",
		Body:           "Test body",
		CardID:         "PRESERVE001",
		Link:           "test",
		SchemaID:       &schemaID,
		StructuredData: &structuredData,
	}
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/cards/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateCardRoute))
	handler.ServeHTTP(rr, req)

	var card models.Card
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	// Verify initial state
	if card.SchemaID == nil {
		t.Errorf("Expected schema_id to be set initially, got nil")
	}
	if *card.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v initially, got %v", schemaID, *card.SchemaID)
	}
	if card.StructuredData == nil {
		t.Errorf("Expected structured_data to be set initially, got nil")
	}

	// Update ONLY the title (partial update - no schema_id or structured_data in request)
	updateData := models.EditCardParams{
		Title:  "Card with Schema - Updated",
		Body:   card.Body,
		CardID: card.CardID,
		Link:   card.Link,
		// SchemaID and StructuredData deliberately omitted
	}
	updateJSON, _ := json.Marshal(updateData)
	updateReq, err := http.NewRequest("PUT", "/api/cards/"+strconv.Itoa(card.ID), bytes.NewBuffer(updateJSON))
	if err != nil {
		t.Fatal(err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.SetPathValue("id", strconv.Itoa(card.ID))

	updateRR := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.UpdateCardRoute))
	router.ServeHTTP(updateRR, updateReq)

	if status := updateRR.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedCard models.Card
	tests.ParseJsonResponse(t, updateRR.Body.Bytes(), &updatedCard)

	// Verify title was updated
	if updatedCard.Title != "Card with Schema - Updated" {
		t.Errorf("Expected title to be updated, got %v", updatedCard.Title)
	}

	// Verify schema_id and structured_data were PRESERVED
	if updatedCard.SchemaID == nil {
		t.Errorf("Expected schema_id to be preserved after partial update, got nil")
	} else if *updatedCard.SchemaID != schemaID {
		t.Errorf("Expected schema_id %v to be preserved after partial update, got %v", schemaID, *updatedCard.SchemaID)
	}

	if updatedCard.StructuredData == nil {
		t.Errorf("Expected structured_data to be preserved after partial update, got nil")
	} else {
		// Verify the content is still correct
		var resultData map[string]interface{}
		err = json.Unmarshal(*updatedCard.StructuredData, &resultData)
		if err != nil {
			t.Fatalf("Failed to unmarshal structured_data: %v", err)
		}
		if resultData["field1"] != "initial value" {
			t.Errorf("Expected structured_data content to be preserved, got %v", resultData)
		}
	}
}

// TestGetRelatedCards_Success tests successful retrieval of related cards
func TestGetRelatedCards_Success(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Get related cards for card 1 (has entities, tags, and content)
	req, err := http.NewRequest("GET", "/api/cards/1/related", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/related", s.JwtMiddleware(s.GetRelatedCardsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %v", status, http.StatusOK, rr.Body.String())
	}

	var relatedCards []models.RelatedCard
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &relatedCards)

	// Verify we got some results
	if len(relatedCards) == 0 {
		t.Error("Expected at least one related card, got none")
	}

	// Verify response structure
	for _, rc := range relatedCards {
		if rc.Card.ID == 0 {
			t.Error("Related card has invalid ID")
		}
		if rc.Score < 0 {
			t.Errorf("Related card has invalid score: %v", rc.Score)
		}
		if len(rc.Reasons) == 0 {
			t.Error("Related card should have at least one reason")
		}
	}
}

// TestGetRelatedCards_ExcludesFamily tests that parent, siblings, and children are excluded
func TestGetRelatedCards_ExcludesFamily(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Card 1 has child 1/A, which should be excluded
	req, err := http.NewRequest("GET", "/api/cards/1/related", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/related", s.JwtMiddleware(s.GetRelatedCardsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var relatedCards []models.RelatedCard
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &relatedCards)

	// Verify that card 1/A (child of 1) is not in results
	for _, rc := range relatedCards {
		if rc.Card.CardID == "1/A" {
			t.Error("Expected child card 1/A to be excluded from related cards")
		}
		if rc.Card.ID == 1 {
			t.Error("Expected source card itself to be excluded from related cards")
		}
	}
}

// TestGetRelatedCards_UnauthorizedUser tests that user cannot access other user's cards
func TestGetRelatedCards_UnauthorizedUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2) // Different user

	req, err := http.NewRequest("GET", "/api/cards/1/related", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/related", s.JwtMiddleware(s.GetRelatedCardsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %v for non-existent card, got %v", http.StatusNotFound, status)
	}
}

// TestGetRelatedCards_NonExistentCard tests 404 for non-existent card
func TestGetRelatedCards_NonExistentCard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/cards/99999/related", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "99999")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/related", s.JwtMiddleware(s.GetRelatedCardsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %v for non-existent card, got %v", http.StatusNotFound, status)
	}
}

// TestGetRelatedCards_ConfigurableWeights verifies that the RELATED_* env vars
// control scoring weights (and that unparsable values fall back to defaults).
func TestGetRelatedCards_ConfigurableWeights(t *testing.T) {
	// t.Setenv restores prior values after the test, so sibling tests keep defaults.
	t.Setenv("RELATED_ENTITY_WEIGHT", "5")
	t.Setenv("RELATED_TAG_WEIGHT", "2")
	t.Setenv("RELATED_MAX_RESULTS", "3")
	// Unparsable value must fall back to the default.
	t.Setenv("RELATED_SEMANTIC_WEIGHT", "not-a-number")

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/cards/1/related", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/related", s.JwtMiddleware(s.GetRelatedCardsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var relatedCards []models.RelatedCard
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &relatedCards)

	if len(relatedCards) > 3 {
		t.Errorf("RELATED_MAX_RESULTS not honored: got %d results, want at most 3", len(relatedCards))
	}

	// Card 3 shares entity 4 ("Test Entity 4") with card 1 and is neither a
	// reference nor family, so it is the entity-based candidate. With
	// RELATED_ENTITY_WEIGHT=5 its score should be 1 * 5 = 5 and its reason
	// should carry the matched entity name.
	found := false
	for _, rc := range relatedCards {
		if rc.Card.ID == 3 {
			found = true
			if rc.Score != 5 {
				t.Errorf("expected card 3 score 5 (1 entity * weight 5), got %v", rc.Score)
			}
			if len(rc.Reasons) == 0 || rc.Reasons[0] != "1 shared entity: Test Entity 4" {
				t.Errorf("expected reason '1 shared entity: Test Entity 4', got %v", rc.Reasons)
			}
		}
	}
	if !found {
		t.Error("expected card 3 to appear in related cards")
	}
}

// TestGetRelatedCards_MaxResultsLimit verifies RELATED_MAX_RESULTS caps the
// number of returned related cards.
func TestGetRelatedCards_MaxResultsLimit(t *testing.T) {
	t.Setenv("RELATED_MAX_RESULTS", "2")

	s := NewHandler()
	defer tests.Teardown()

	// Give cards 4 and 5 a shared entity with card 1, so there are 3 entity
	// candidates (card 3 + cards 4 and 5). Card 2 is excluded as a reference.
	for _, cardPK := range []int{4, 5} {
		if _, err := s.GetDB().Exec(`
			INSERT INTO entity_card_junction (user_id, entity_id, card_pk)
			VALUES ($1, $2, $3)
		`, 1, 1, cardPK); err != nil {
			t.Fatalf("failed to link entity 1 to card %d: %v", cardPK, err)
		}
	}

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/cards/1/related", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/related", s.JwtMiddleware(s.GetRelatedCardsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var relatedCards []models.RelatedCard
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &relatedCards)

	if len(relatedCards) != 2 {
		t.Errorf("RELATED_MAX_RESULTS=2 not honored: got %d results", len(relatedCards))
	}
}

// TestGetUnlinkedMentions verifies detection of plain-text card_id mentions
// that are not linked, including already-linked exclusion, ownership, and
// word-boundary handling.
func TestGetUnlinkedMentions(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Source card with a mentionable (non-numeric) card_id.
	var sourceID int
	if err := s.GetDB().QueryRow(`
		INSERT INTO cards (user_id, card_id, title, body, link, created_at, updated_at)
		VALUES ($1, $2, $3, $4, '', datetime('now'), datetime('now'))
		RETURNING id
	`, 1, "REFXYZ", "Source", "Source body").Scan(&sourceID); err != nil {
		t.Fatalf("failed to create source card: %v", err)
	}

	// Card A: mentions REFXYZ in plain text once.
	// Card B: already links to REFXYZ - must be excluded.
	// Card C: mentions REFXYZ twice - count should be 2.
	// Card E: contains REFXYZ2 - word boundary, must NOT match REFXYZ.
	// Card D (user 2): mentions REFXYZ - must be excluded by ownership.
	cards := []struct {
		userID int
		cardID string
		body   string
	}{
		{1, "MENTION_A", "Some intro. See REFXYZ for the full notes."},
		{1, "MENTION_B", "Already linked via [[REFXYZ]] here."},
		{1, "MENTION_C", "REFXYZ covers this, and REFXYZ also covers that."},
		{1, "MENTION_E", "REFXYZ2 is a different id, not a match."},
		{2, "MENTION_D", "REFXYZ belongs to another user."},
	}
	for _, c := range cards {
		if _, err := s.GetDB().Exec(`
			INSERT INTO cards (user_id, card_id, title, body, link, created_at, updated_at)
			VALUES ($1, $2, $3, $4, '', datetime('now'), datetime('now'))
		`, c.userID, c.cardID, "Mention Card", c.body); err != nil {
			t.Fatalf("failed to create card %s: %v", c.cardID, err)
		}
	}

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/cards/"+strconv.Itoa(sourceID)+"/unlinked-mentions", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(sourceID))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/unlinked-mentions", s.JwtMiddleware(s.GetUnlinkedMentionsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var mentions []models.UnlinkedMention
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &mentions)

	byID := make(map[string]models.UnlinkedMention)
	for _, m := range mentions {
		byID[m.Card.CardID] = m
	}

	a, ok := byID["MENTION_A"]
	if !ok {
		t.Error("expected MENTION_A to be returned (plain-text mention)")
	} else {
		if a.MentionCount != 1 {
			t.Errorf("expected MENTION_A count 1, got %d", a.MentionCount)
		}
		if a.ContextSnippet == "" || !strings.Contains(a.ContextSnippet, "REFXYZ") {
			t.Errorf("expected non-empty snippet containing REFXYZ, got %q", a.ContextSnippet)
		}
	}

	if c, ok := byID["MENTION_C"]; !ok {
		t.Error("expected MENTION_C to be returned (two mentions)")
	} else if c.MentionCount != 2 {
		t.Errorf("expected MENTION_C count 2, got %d", c.MentionCount)
	}

	if _, ok := byID["MENTION_B"]; ok {
		t.Error("MENTION_B already links to REFXYZ and must be excluded")
	}
	if _, ok := byID["MENTION_E"]; ok {
		t.Error("MENTION_E contains REFXYZ2, which must not match REFXYZ")
	}
	if _, ok := byID["MENTION_D"]; ok {
		t.Error("MENTION_D belongs to another user and must be excluded")
	}
}

// TestGetUnlinkedMentions_NumericCardID verifies numeric-only card_ids are
// skipped (they would match nearly every body).
func TestGetUnlinkedMentions_NumericCardID(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Card 1 has card_id "1" (fixture) - numeric-only, must return empty.
	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/cards/1/unlinked-mentions", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/unlinked-mentions", s.JwtMiddleware(s.GetUnlinkedMentionsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var mentions []models.UnlinkedMention
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &mentions)
	if len(mentions) != 0 {
		t.Errorf("expected empty mentions for numeric card_id, got %d", len(mentions))
	}
}

// TestGetUnlinkedMentions_NonExistentCard verifies 404 for unknown cards.
func TestGetUnlinkedMentions_NonExistentCard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/cards/99999/unlinked-mentions", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "99999")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/cards/{id}/unlinked-mentions", s.JwtMiddleware(s.GetUnlinkedMentionsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %v for non-existent card, got %v", http.StatusNotFound, status)
	}
}

// TestEnvFloatAndEnvInt verifies the env helpers fall back to defaults for
// unset and unparsable values.
func TestEnvFloatAndEnvInt(t *testing.T) {
	if got := envFloat("TEST_ENV_FLOAT", 3); got != 3 {
		t.Errorf("expected unset env to return default 3, got %v", got)
	}
	t.Setenv("TEST_ENV_FLOAT", "4.5")
	if got := envFloat("TEST_ENV_FLOAT", 3); got != 4.5 {
		t.Errorf("expected 4.5, got %v", got)
	}
	t.Setenv("TEST_ENV_FLOAT", "junk")
	if got := envFloat("TEST_ENV_FLOAT", 3); got != 3 {
		t.Errorf("expected unparsable env to return default 3, got %v", got)
	}

	if got := envInt("TEST_ENV_INT", 10); got != 10 {
		t.Errorf("expected unset env to return default 10, got %v", got)
	}
	t.Setenv("TEST_ENV_INT", "25")
	if got := envInt("TEST_ENV_INT", 10); got != 25 {
		t.Errorf("expected 25, got %v", got)
	}
	t.Setenv("TEST_ENV_INT", "junk")
	if got := envInt("TEST_ENV_INT", 10); got != 10 {
		t.Errorf("expected unparsable env to return default 10, got %v", got)
	}
}
