package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

// get existing tag
func TestGetTag(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	tagName := "test"

	tag, err := services.GetTag(s.Server.Tx, userID, tagName)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if tag.Name != tagName {
		t.Errorf("handler returned wrong tag, got %v want %v", tag.Name, tagName)
	}
}

func TestGetTagNotFound(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	tagName := "no tag by this name"

	tag, err := services.GetTag(s.Server.Tx, userID, tagName)
	if err == nil {
		t.Errorf("handler did not return error when expecting error")
	}
	if tag.Name != "" {
		t.Errorf("handler returned a tag when not expecting one, got %v", tag.Name)
	}

}

func TestGetTagsRoute(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/tags", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetTagsRoute))
	handler.ServeHTTP(rr, req)

	var tags []models.Tag
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &tags)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if len(tags) != 3 {
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(tags), 3)
	}
}

func TestCreateTagRoute(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	tagData := models.EditTagParams{
		Name:  "new-test-tag",
		Color: "blue",
	}
	jsonData, _ := json.Marshal(tagData)

	req, err := http.NewRequest("POST", "/api/tags", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateTagRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var tag models.Tag
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &tag)

	if tag.Name != tagData.Name {
		t.Errorf("handler returned wrong tag name: got %v want %v", tag.Name, tagData.Name)
	}
	if tag.Color != tagData.Color {
		t.Errorf("handler returned wrong tag color: got %v want %v", tag.Color, tagData.Color)
	}
}

// update tag (set new colour)

func TestEditTag(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1

	tagName := "test"
	tag, err := services.GetTag(s.Server.Tx, userID, tagName)

	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if tag.Name != tagName {
		t.Errorf("handler returned wrong tag, got %v want %v", tag.Name, tagName)
	}
	tagData := models.EditTagParams{
		Name:  "hello-world",
		Color: "red",
	}

	tag, err = services.EditTag(s.Server.Tx, userID, tagName, tagData)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if tag.Name != tagData.Name {
		t.Errorf("handler returned wrong tag, got %v want %v", tag.Name, tagData.Name)
	}
	tag, err = services.GetTag(s.Server.Tx, userID, tagData.Name)
	if err != nil {
		t.Errorf("handler returned err %v", err)
	}
	if tag.Name != tagData.Name {
		t.Errorf("handler returned wrong tag, got %v want %v", tag.Name, tagData.Name)
	}
}

func TestCreateTagOverExisting(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1

	tagName := "test"
	tag, err := services.GetTag(s.Server.Tx, userID, tagName)
	oldID := tag.ID

	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if tag.Name != tagName {
		t.Errorf("handler returned wrong tag, got %v want %v", tag.Name, tagName)
	}
	tagData := models.EditTagParams{
		Name:  tagName,
		Color: "red",
	}
	tag, err = services.CreateTag(s.Server.Tx, userID, tagData)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if tag.ID != oldID {
		t.Errorf("handler returned wrong tag, got id %v want id %v", tag.ID, oldID)
	}
	tag, err = services.GetTag(s.Server.Tx, userID, tagData.Name)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if tag.ID != oldID {
		t.Errorf("handler returned wrong tag, got id %v want id %v", tag.ID, oldID)
	}
	if tag.Name != tagData.Name {
		t.Errorf("handler returned wrong tag, got %v want %v", tag.Name, tagData.Name)
	}

}

// add tags for a card

func TestAddTagToCard(t *testing.T) {

	s := setup()
	defer tests.Teardown()

	userID := 1
	cardPK := 1
	tagName := "test"

	var count int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_tags").Scan(&count)

	err := services.AddTagToCard(s.Server.Tx, userID, tagName, cardPK)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}

	var newCount int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_tags").Scan(&newCount)
	if newCount != (count + 1) {
		t.Errorf("handler returned wrong number of card_tags, got %v want %v", newCount, count+1)
	}

}

func TestAddTagsFromCardQuery(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 2

	cardPK := 13 // Updated after test data reduction (was 23)
	tagName := "to-read"

	var count int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_tags").Scan(&count)

	err := services.AddTagsFromCard(s.Server.Tx, userID, cardPK)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}

	var newCount int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM card_tags").Scan(&newCount)
	if newCount != (count + 1) {
		t.Errorf("handler returned wrong number of card_tags, got %v want %v", newCount, count+1)
	}

	tags, err := services.QueryTagsForCard(s.Server.Tx, userID, cardPK)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if len(tags) != 1 {
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(tags), 1)
	}

	if tags[0].Name != tagName {
		t.Errorf("wrong tag attached to card, got %v want %v", tags[0].Name, tagName)
	}

}

func TestAddTagsFromTaskQuery(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	userID := 1
	taskPK := 3
	tagName := "to-read"

	var count int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM task_tags").Scan(&count)

	err := s.AddTagsFromTask(userID, taskPK)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}

	var newCount int
	_ = s.Server.Tx.QueryRow("SELECT count(*) FROM task_tags").Scan(&newCount)
	if newCount != (count + 1) {
		t.Errorf("handler returned wrong number of task_tags, got %v want %v", newCount, count+1)
	}

	tags, err := s.QueryTagsForTask(userID, taskPK)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if len(tags) != 1 {
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(tags), 1)
	}

	if tags[0].Name != tagName {
		t.Errorf("wrong tag attached to card, got %v want %v", tags[0].Name, tagName)
	}
}

func TestParseTagsFromCardBody(t *testing.T) {
	body := "hello world \n\n#to-read #hello#world"

	tags, err := services.ParseTagsFromCardBody(body)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if len(tags) != 2 {
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(tags), 2)
	}
	if tags[0] != "to-read" {
		t.Errorf("wrong tag returned, got %v want %v", tags[0], "to-read")
	}
	if tags[1] != "hello" {
		t.Errorf("wrong tag returned, got %v want %v", tags[1], "hello")
	}

	body = "hello world \n\nto-read#hello #world"
	tags, err = services.ParseTagsFromCardBody(body)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if len(tags) != 1 {
		log.Printf("tags %v", tags)
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(tags), 1)
	}
	if tags[0] != "world" {
		t.Errorf("wrong tag returned, got %v want %v", tags[0], "world")
	}

}

func TestDeleteTag(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	id := 2
	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("DELETE", "/api/tags/id/"+strconv.Itoa(id), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(id))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tags/id/{id}", s.JwtMiddleware(s.DeleteTagRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		log.Printf("body %v", rr.Body)
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	_, err = services.GetTag(s.Server.Tx, 2, "test")
	if err == nil {
		t.Error("handler returned tag after it should have been deleted")
	}
}

func TestIdentifyParentTags(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	tag, err := s.getTagByID(1, 1)
	if err != nil {
		t.Errorf("handler returned error finding card: %v", err.Error())
	}
	expectedName := tag.Name

	card, err := s.QueryPartialCardByID(1, 14) // Updated after test data reduction (was 24)
	if err != nil {
		t.Errorf("handler returned error finding card: %v", err.Error())
	}
	parent_tags, err := services.IdentifyParentTags(s.Server.Tx, 1, card)
	if err != nil {
		t.Errorf("handler returned error: %v", err.Error())
	}
	log.Printf("%v", parent_tags)
	if len(parent_tags) != 1 {
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(parent_tags), 1)
	}
	if len(parent_tags) > 0 && parent_tags[0].Name != expectedName {
		t.Errorf("handler returned wrong tag, got %v want %v", parent_tags[0].Name, expectedName)
	}
}

func TestCreateCardSuccessRecursiveTags(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	tag, err := s.getTagByID(1, 1)
	if err != nil {
		t.Errorf("handler returned error finding card: %v", err.Error())
	}
	expectedName := tag.Name

	var card models.Card
	token, _ := tests.GenerateTestJWT(1)

	expected := "asdfasdf"
	data := models.EditCardParams{
		Title:  expected,
		Body:   expected,
		CardID: "2/A.1/A",
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

	tags, err := services.QueryTagsForCard(s.Server.Tx, 1, card.ID)
	if err != nil {
		t.Errorf("handler returned error, %v", err.Error())
	}
	if len(tags) != 1 {
		t.Errorf("handler returned wrong number of tags, got %v want %v", len(tags), 1)
	}

	if tags[0].Name != expectedName {
		t.Errorf("wrong tag attached to card, got %v want %v", tags[0].Name, expectedName)
	}

}
