package handlers

import (
	"encoding/json"
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestListRSSFeedsRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/rss/feeds", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/feeds", s.JwtMiddleware(s.ListRSSFeedsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feeds []models.RSSFeed
	json.Unmarshal(rr.Body.Bytes(), &feeds)

	// Should be empty initially
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func TestListRSSArticlesRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/rss/articles", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/articles", s.JwtMiddleware(s.ListRSSArticlesRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var articles []models.RSSArticle
	json.Unmarshal(rr.Body.Bytes(), &articles)

	// Should be empty initially
	if len(articles) != 0 {
		t.Errorf("expected 0 articles, got %d", len(articles))
	}
}

func TestListRSSFoldersRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/rss/folders", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/folders", s.JwtMiddleware(s.ListRSSFoldersRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var folders []models.RSSFolder
	json.Unmarshal(rr.Body.Bytes(), &folders)

	// Should be empty initially
	if len(folders) != 0 {
		t.Errorf("expected 0 folders, got %d", len(folders))
	}
}
