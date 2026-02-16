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

// TestStarRSSArticleRoute tests starring an RSS article
func TestStarRSSArticleRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create a test RSS feed first
	feedReq, _ := http.NewRequest("POST", "/api/rss/feeds", tests.StringToReader(`{"url":"https://example.com/feed.xml","name":"Test Feed"}`))
	feedReq.Header.Set("Authorization", "Bearer "+token)
	feedReq.Header.Set("Content-Type", "application/json")

	feedRR := httptest.NewRecorder()
	feedRouter := mux.NewRouter()
	feedRouter.HandleFunc("/api/rss/feeds", s.JwtMiddleware(s.CreateRSSFeedRoute))
	feedRouter.ServeHTTP(feedRR, feedReq)

	if feedRR.Code != http.StatusCreated && feedRR.Code != http.StatusOK {
		t.Fatalf("Failed to create test feed: status %d", feedRR.Code)
	}

	var feed models.RSSFeed
	json.Unmarshal(feedRR.Body.Bytes(), &feed)

	// Create a test article directly in the database
	var articleID int
	err := s.GetDB().QueryRow(`
		INSERT INTO rss_articles (user_id, feed_id, title, url, content, read, is_starred)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, 1, feed.ID, "Test Article", "https://example.com/article", "Test content", false, false).Scan(&articleID)
	if err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	// Test starring the article
	starReq, _ := http.NewRequest("POST", "/api/rss/articles/"+string(rune(articleID))+"/star", nil)
	starReq.Header.Set("Authorization", "Bearer "+token)

	starRR := httptest.NewRecorder()
	starRouter := mux.NewRouter()
	starRouter.HandleFunc("/api/rss/articles/{id}/star", s.JwtMiddleware(s.StarRSSArticleRoute))
	starRouter.ServeHTTP(starRR, starReq)

	if status := starRR.Code; status != http.StatusNoContent {
		t.Errorf("Star handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Verify the article is starred
	var article models.RSSArticle
	getReq, _ := http.NewRequest("GET", "/api/rss/articles/"+string(rune(articleID)), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	getRR := httptest.NewRecorder()
	getRouter := mux.NewRouter()
	getRouter.HandleFunc("/api/rss/articles/{id}", s.JwtMiddleware(s.GetRSSArticleRoute))
	getRouter.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("Failed to get article: status %d", getRR.Code)
	}

	json.Unmarshal(getRR.Body.Bytes(), &article)
	if !article.IsStarred {
		t.Errorf("Expected article to be starred, but IsStarred is false")
	}
}

// TestUnstarRSSArticleRoute tests unstarring an RSS article
func TestUnstarRSSArticleRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create a test RSS feed first
	feedReq, _ := http.NewRequest("POST", "/api/rss/feeds", tests.StringToReader(`{"url":"https://example.com/feed.xml","name":"Test Feed"}`))
	feedReq.Header.Set("Authorization", "Bearer "+token)
	feedReq.Header.Set("Content-Type", "application/json")

	feedRR := httptest.NewRecorder()
	feedRouter := mux.NewRouter()
	feedRouter.HandleFunc("/api/rss/feeds", s.JwtMiddleware(s.CreateRSSFeedRoute))
	feedRouter.ServeHTTP(feedRR, feedReq)

	if feedRR.Code != http.StatusCreated && feedRR.Code != http.StatusOK {
		t.Fatalf("Failed to create test feed: status %d", feedRR.Code)
	}

	var feed models.RSSFeed
	json.Unmarshal(feedRR.Body.Bytes(), &feed)

	// Create a starred test article directly in the database
	var articleID int
	err := s.GetDB().QueryRow(`
		INSERT INTO rss_articles (user_id, feed_id, title, url, content, read, is_starred)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, 1, feed.ID, "Test Article", "https://example.com/article", "Test content", false, true).Scan(&articleID)
	if err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	// Test unstarring the article
	unstarReq, _ := http.NewRequest("DELETE", "/api/rss/articles/"+string(rune(articleID))+"/star", nil)
	unstarReq.Header.Set("Authorization", "Bearer "+token)

	unstarRR := httptest.NewRecorder()
	unstarRouter := mux.NewRouter()
	unstarRouter.HandleFunc("/api/rss/articles/{id}/star", s.JwtMiddleware(s.UnstarRSSArticleRoute))
	unstarRouter.ServeHTTP(unstarRR, unstarReq)

	if status := unstarRR.Code; status != http.StatusNoContent {
		t.Errorf("Unstar handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Verify the article is unstarred
	var article models.RSSArticle
	getReq, _ := http.NewRequest("GET", "/api/rss/articles/"+string(rune(articleID)), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	getRR := httptest.NewRecorder()
	getRouter := mux.NewRouter()
	getRouter.HandleFunc("/api/rss/articles/{id}", s.JwtMiddleware(s.GetRSSArticleRoute))
	getRouter.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("Failed to get article: status %d", getRR.Code)
	}

	json.Unmarshal(getRR.Body.Bytes(), &article)
	if article.IsStarred {
		t.Errorf("Expected article to be unstarred, but IsStarred is true")
	}
}

// TestListStarredRSSArticlesRoute tests filtering articles by starred status
func TestListStarredRSSArticlesRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Create a test RSS feed first
	feedReq, _ := http.NewRequest("POST", "/api/rss/feeds", tests.StringToReader(`{"url":"https://example.com/feed.xml","name":"Test Feed"}`))
	feedReq.Header.Set("Authorization", "Bearer "+token)
	feedReq.Header.Set("Content-Type", "application/json")

	feedRR := httptest.NewRecorder()
	feedRouter := mux.NewRouter()
	feedRouter.HandleFunc("/api/rss/feeds", s.JwtMiddleware(s.CreateRSSFeedRoute))
	feedRouter.ServeHTTP(feedRR, feedReq)

	if feedRR.Code != http.StatusCreated && feedRR.Code != http.StatusOK {
		t.Fatalf("Failed to create test feed: status %d", feedRR.Code)
	}

	var feed models.RSSFeed
	json.Unmarshal(feedRR.Body.Bytes(), &feed)

	// Create two articles - one starred, one not
	_, err := s.GetDB().Exec(`
		INSERT INTO rss_articles (user_id, feed_id, title, url, content, read, is_starred)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, 1, feed.ID, "Unstarred Article", "https://example.com/article1", "Test content 1", false, false)
	if err != nil {
		t.Fatalf("Failed to create unstarred article: %v", err)
	}

	_, err = s.GetDB().Exec(`
		INSERT INTO rss_articles (user_id, feed_id, title, url, content, read, is_starred)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, 1, feed.ID, "Starred Article", "https://example.com/article2", "Test content 2", false, true)
	if err != nil {
		t.Fatalf("Failed to create starred article: %v", err)
	}

	// Test getting only starred articles
	starredReq, _ := http.NewRequest("GET", "/api/rss/articles?starred=true", nil)
	starredReq.Header.Set("Authorization", "Bearer "+token)

	starredRR := httptest.NewRecorder()
	starredRouter := mux.NewRouter()
	starredRouter.HandleFunc("/api/rss/articles", s.JwtMiddleware(s.ListRSSArticlesRoute))
	starredRouter.ServeHTTP(starredRR, starredReq)

	if status := starredRR.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var result map[string]interface{}
	json.Unmarshal(starredRR.Body.Bytes(), &result)

	articles := result["articles"].([]interface{})
	if len(articles) != 1 {
		t.Errorf("Expected 1 starred article, got %d", len(articles))
	}

	// Verify it's the starred article
	if len(articles) > 0 {
		article := articles[0].(map[string]interface{})
		if article["title"] != "Starred Article" {
			t.Errorf("Expected 'Starred Article', got %v", article["title"])
		}
		if article["is_starred"] != true {
			t.Errorf("Expected is_starred to be true, got %v", article["is_starred"])
		}
	}
}
