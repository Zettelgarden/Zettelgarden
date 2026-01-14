package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	s := setup()
	defer tests.Teardown()

	var logCount int
	_ = s.DB.QueryRow("SELECT count(*) FROM card_views").Scan(&logCount)
	if logCount != 0 {
		t.Errorf("wrong log count, got %v want %v", logCount, 0)
	}
	rr := makeCardRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	_ = s.DB.QueryRow("SELECT count(*) FROM card_views").Scan(&logCount)
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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	s := setup()
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

	var refs CategorizedReferences
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &refs)

	// Card 1 references card 2 (outgoing), and card 22 (2/A) references card 1 (incoming)
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
	s := setup()
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

	var refs CategorizedReferences
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
	s := setup()
	defer tests.Teardown()

	var linkCount int
	_ = s.DB.QueryRow("SELECT count(*) FROM card_views").Scan(&linkCount)
	log.Printf("count %v", linkCount)
	if linkCount != 0 {
		t.Errorf("wrong log count, got %v want %v", linkCount, 0)
	}

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
	_ = s.DB.QueryRow("SELECT count(*) FROM card_views").Scan(&newLinkCount)
	log.Printf("new count %v", newLinkCount)
	if newLinkCount == linkCount {
		t.Errorf("wrong log count, got %v want %v", linkCount, 1)
	}
}

func TestUpdateCardUnauthorized(t *testing.T) {
	s := setup()
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
	s := setup()
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

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	rr = makeCardRequestSuccess(s, t, card.ID)

	tests.ParseJsonResponse(t, rr.Body.Bytes(), &newCard)
	if newCard.Title != expected {
		t.Errorf("handler returned wrong card: got %v want %v", newCard.Title, expected)
	}
}

func TestCreateCardDuplicateCardID(t *testing.T) {
	s := setup()
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

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
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
	s := setup()
	defer tests.Teardown()

	id := 3
	rr := makeCardDeleteRequestSuccess(s, t, id)

	if status := rr.Code; status != http.StatusNoContent {
		log.Printf(rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	rr = makeCardRequestSuccess(s, t, id)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
	rr = makeCardDeleteRequestSuccess(s, t, id)

	if status := rr.Code; status != http.StatusNotFound {
		log.Printf(rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestDeleteCardWrongUser(t *testing.T) {
	s := setup()
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
	s := setup()
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

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &card)

	rr = makeCardRequestSuccess(s, t, card.ID)
	log.Printf("%v", rr.Body.String())
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &newCard)
	if newCard.ParentID != parentCard.ID {
		t.Errorf("handler returned wrong parent: got %v want %v", newCard.ParentID, parentCard.ID)
	}
}

func TestGetNextRootCardID(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Test when no cards exist
	nextID := s.getNextRootCardID(1)
	if nextID != "21" {
		t.Errorf("Expected first ID to be 21 (after test data), got %v", nextID)
	}

	// Create a card with numeric ID
	data := models.EditCardParams{
		Title:  "Test Card",
		Body:   "Test Body",
		CardID: "5", // Using 5 to test non-sequential numbers
		Link:   "",
	}
	var err error
	_, err = services.CreateCard(s.DB, 2, data)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Test getting next ID after card exists (should still be 22 since 5 is lower)
	nextID = s.getNextRootCardID(1)
	if nextID != "21" {
		t.Errorf("Expected next ID to still be 21 (5 is lower), got %v", nextID)
	}

	// Test that non-numeric IDs are ignored
	data.CardID = "ABC123"
	_, err = services.CreateCard(s.DB, 1, data)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	nextID = s.getNextRootCardID(1)
	if nextID != "21" {
		t.Errorf("Expected next ID to still be 21 (ignoring non-numeric ID), got %v", nextID)
	}
}

func TestGetNextRootCardIDRoute(t *testing.T) {
	s := setup()
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
	if response.NextID != "21" {
		t.Errorf("Expected first ID to be 21 (after test data), got %v", response.NextID)
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
	s := setup()
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
	if !s.checkChunkLinkedOrRelated(userID, mainCard, models.CardChunk{
		ID:       testCard.ID,
		ParentID: mainCard.ID,
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

func TestGetNextChildCardID(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Create a test parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card Test",
		Body:   "Test Body",
		CardID: "999", // Use high number to avoid conflicts
		Link:   "",
	}
	parentCard, err := services.CreateCard(s.DB, 1, parentParams)
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
	childCard1, err := services.CreateCard(s.DB, 1, childParams)
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
	childCard2, err := services.CreateCard(s.DB, 1, childParams)
	if err != nil {
		t.Fatalf("Failed to create second child test card: %v", err)
	}

	nextID = s.getNextChildCardID(1, childCard1.ID) // Test with nested grandchild
	if nextID != "999.1.1" {                           // Should extend the existing child ID
		t.Errorf("Expected nested child ID to be 999.1.1, got %s", nextID)
	}

	// Test 4: Non-sequential children - add ".5" (skip 3 and 4)
	childParams.CardID = "999.5"
	_, err = services.CreateCard(s.DB, 1, childParams)
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
	_, err = services.CreateCard(s.DB, 1, childParams)
	if err != nil {
		t.Fatalf("Failed to create fourth child test card: %v", err)
	}

	nextID = s.getNextChildCardID(1, parentCard.ID)
	expected = "999.8" // Should handle mixed separators and find max is 7
	if nextID != expected {
		t.Errorf("Expected next child ID to be %s (handling mixed separators), got %s", expected, nextID)
	}

	// Clean up test cards
	for _, cardID := range []int{childCard1.ID, childCard2.ID, parentCard.ID} {
		_, err = s.DB.Exec("DELETE FROM cards WHERE id = $1", cardID)
		if err != nil {
			t.Logf("Failed to clean up test card %d: %v", cardID, err)
		}
	}
}

func TestGetNextChildCardIDRoute(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Create a test parent card
	parentParams := models.EditCardParams{
		Title:  "Parent Card Route Test",
		Body:   "Test Body",
		CardID: "888", // Use high number to avoid conflicts
		Link:   "",
	}
	parentCard, err := services.CreateCard(s.DB, 1, parentParams)
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
	_, err = s.DB.Exec("DELETE FROM cards WHERE id = $1", parentCard.ID)
	if err != nil {
		t.Logf("Failed to clean up test card %d: %v", parentCard.ID, err)
	}
}
