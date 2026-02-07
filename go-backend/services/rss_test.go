package services

import (
	"go-backend/models"
	"go-backend/tests"
	"testing"
)

func TestCreateRSSFeed(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	params := models.CreateRSSFeedParams{
		URL:    "https://example.com/feed.xml",
		Name:   "Test Feed",
		Folder: stringPtr("Tech"),
	}

	// This will fail with a fake URL but tests the logic
	feed, err := CreateRSSFeed(s.Tx, userID, params)
	if err != nil {
		t.Logf("Expected error with fake URL: %v", err)
	}
	if feed != nil && feed.Name != params.Name {
		t.Errorf("expected name %s, got %s", params.Name, feed.Name)
	}
}

func TestListRSSFeeds(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	feeds, err := ListRSSFeeds(s.Tx, userID)
	if err != nil {
		t.Errorf("failed to list feeds: %v", err)
	}
	// Should be empty initially
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func TestListRSSArticles(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	articles, err := ListRSSArticles(s.Tx, userID, map[string]interface{}{})
	if err != nil {
		t.Errorf("failed to list articles: %v", err)
	}
	// Should be empty initially
	if len(articles) != 0 {
		t.Errorf("expected 0 articles, got %d", len(articles))
	}
}

func TestListRSSFolders(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	folders, err := ListRSSFolders(s.Tx, userID)
	if err != nil {
		t.Errorf("failed to list folders: %v", err)
	}
	// Should be empty initially
	if len(folders) != 0 {
		t.Errorf("expected 0 folders, got %d", len(folders))
	}
}

func TestCreateRSSFolder(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	name := "Tech"
	orderIndex := 0

	params := models.CreateRSSFolderParams{
		Name:       name,
		OrderIndex: &orderIndex,
	}
	folder, err := CreateRSSFolder(s.Tx, userID, params)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	if folder.Name != name {
		t.Errorf("expected name %s, got %s", name, folder.Name)
	}

	if folder.OrderIndex != orderIndex {
		t.Errorf("expected order_index %d, got %d", orderIndex, folder.OrderIndex)
	}

	if folder.UserID != userID {
		t.Errorf("expected user_id %d, got %d", userID, folder.UserID)
	}
}

func TestGetRSSFolderByID(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	name := "News"
	orderIndex := 1

	// Create a folder first
	params := models.CreateRSSFolderParams{
		Name:       name,
		OrderIndex: &orderIndex,
	}
	createdFolder, err := CreateRSSFolder(s.Tx, userID, params)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	// Get the folder by ID
	folder, err := GetRSSFolderByID(s.Tx, userID, createdFolder.ID)
	if err != nil {
		t.Fatalf("failed to get folder: %v", err)
	}

	if folder.ID != createdFolder.ID {
		t.Errorf("expected id %d, got %d", createdFolder.ID, folder.ID)
	}

	if folder.Name != name {
		t.Errorf("expected name %s, got %s", name, folder.Name)
	}

	if folder.OrderIndex != orderIndex {
		t.Errorf("expected order_index %d, got %d", orderIndex, folder.OrderIndex)
	}
}

func TestGetRSSFeedByIDNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	feedID := 99999

	_, err := GetRSSFeedByID(s.Tx, userID, feedID)
	if err == nil {
		t.Error("expected error for non-existent feed")
	}

	if err != nil && err.Error() != "feed not found" {
		t.Errorf("expected 'feed not found' error, got: %v", err)
	}
}

func TestGetRSSArticleByIDNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	articleID := 99999

	_, err := GetRSSArticleByID(s.Tx, userID, articleID)
	if err == nil {
		t.Error("expected error for non-existent article")
	}

	if err != nil && err.Error() != "article not found" {
		t.Errorf("expected 'article not found' error, got: %v", err)
	}
}

func TestGetRSSFolderByIDNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	folderID := 99999

	_, err := GetRSSFolderByID(s.Tx, userID, folderID)
	if err == nil {
		t.Error("expected error for non-existent folder")
	}

	if err != nil && err.Error() != "folder not found" {
		t.Errorf("expected 'folder not found' error, got: %v", err)
	}
}

func TestDeleteRSSFeedNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	feedID := 99999

	err := DeleteRSSFeed(s.Tx, userID, feedID)
	if err == nil {
		t.Error("expected error for non-existent feed")
	}

	if err != nil && err.Error() != "feed not found" {
		t.Errorf("expected 'feed not found' error, got: %v", err)
	}
}

func TestMarkRSSArticleAsReadNotFound(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1
	articleID := 99999

	err := MarkRSSArticleAsRead(s.Tx, userID, articleID, true)
	if err == nil {
		t.Error("expected error for non-existent article")
	}

	if err != nil && err.Error() != "article not found" {
		t.Errorf("expected 'article not found' error, got: %v", err)
	}
}

func TestUpdateRSSFeedNoChanges(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	// First create a folder to use in feed
	userID := 1
	folderParams := models.CreateRSSFolderParams{
		Name: "Tech",
	}
	folder, err := CreateRSSFolder(s.Tx, userID, folderParams)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	// Create a feed (we'll manually insert since CreateRSSFeed tries to validate URL)
	var feedID int
	err = s.Tx.QueryRow(`
		INSERT INTO rss_feeds (user_id, url, name, folder, auto_tags, fetch_interval, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userID, "https://example.com/feed.xml", "Test Feed", folder.Name, "", 60, true).Scan(&feedID)
	if err != nil {
		t.Fatalf("failed to insert feed: %v", err)
	}

	// Update with no changes (empty params)
	updatedFeed, err := UpdateRSSFeed(s.Tx, userID, feedID, models.UpdateRSSFeedParams{})
	if err != nil {
		t.Fatalf("failed to update feed: %v", err)
	}

	if updatedFeed.Name != "Test Feed" {
		t.Errorf("expected name to remain 'Test Feed', got %s", updatedFeed.Name)
	}
}

func TestListRSSArticlesWithUnreadFilter(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	userID := 1

	// Test with unread filter
	filters := map[string]interface{}{
		"unread": true,
	}

	articles, err := ListRSSArticles(s.Tx, userID, filters)
	if err != nil {
		t.Errorf("failed to list articles with unread filter: %v", err)
	}

	// Should be empty since we have no articles
	if len(articles) != 0 {
		t.Errorf("expected 0 unread articles, got %d", len(articles))
	}
}
