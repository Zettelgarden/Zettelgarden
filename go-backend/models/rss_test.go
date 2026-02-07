package models

import "testing"

func TestRSSFeedModel(t *testing.T) {
	folder := "Tech"
	feed := RSSFeed{
		ID:     1,
		UserID: 1,
		URL:    "https://example.com/feed.xml",
		Name:   "Example Feed",
		Folder: &folder,
	}

	if feed.URL != "https://example.com/feed.xml" {
		t.Errorf("expected URL to be https://example.com/feed.xml, got %s", feed.URL)
	}

	if feed.Name != "Example Feed" {
		t.Errorf("expected Name to be Example Feed, got %s", feed.Name)
	}

	if feed.Folder == nil || *feed.Folder != "Tech" {
		t.Errorf("expected Folder to be Tech, got %v", feed.Folder)
	}
}

func TestRSSArticleModel(t *testing.T) {
	article := RSSArticle{
		ID:     1,
		UserID: 1,
		FeedID: 1,
		Title:  "Test Article",
		URL:    "https://example.com/article",
	}

	if article.Title != "Test Article" {
		t.Errorf("expected Title to be Test Article, got %s", article.Title)
	}

	if article.URL != "https://example.com/article" {
		t.Errorf("expected URL to be https://example.com/article, got %s", article.URL)
	}
}

func TestRSSFolderModel(t *testing.T) {
	folder := RSSFolder{
		ID:         1,
		UserID:     1,
		Name:       "Tech",
		OrderIndex: 0,
	}

	if folder.Name != "Tech" {
		t.Errorf("expected Name to be Tech, got %s", folder.Name)
	}

	if folder.OrderIndex != 0 {
		t.Errorf("expected OrderIndex to be 0, got %d", folder.OrderIndex)
	}
}

func TestCreateRSSFeedParams(t *testing.T) {
	autoTags := "tech,news"
	folder := "Tech"
	params := CreateRSSFeedParams{
		URL:      "https://example.com/feed.xml",
		Name:     "Example Feed",
		Folder:   &folder,
		AutoTags: autoTags,
	}

	if params.URL != "https://example.com/feed.xml" {
		t.Errorf("expected URL to be https://example.com/feed.xml, got %s", params.URL)
	}

	if params.AutoTags != "tech,news" {
		t.Errorf("expected AutoTags to be tech,news, got %s", params.AutoTags)
	}
}

func TestUpdateRSSFeedParams(t *testing.T) {
	name := "Updated Feed"
	folder := "News"
	autoTags := "updated"
	enabled := true

	params := UpdateRSSFeedParams{
		Name:     &name,
		Folder:   &folder,
		AutoTags: &autoTags,
		Enabled:  &enabled,
	}

	if params.Name == nil || *params.Name != "Updated Feed" {
		t.Errorf("expected Name to be Updated Feed, got %v", params.Name)
	}

	if params.Enabled == nil || *params.Enabled != true {
		t.Errorf("expected Enabled to be true, got %v", params.Enabled)
	}
}

func TestConvertArticleParams(t *testing.T) {
	title := "Custom Title"
	body := "Custom body content"
	tags := "#custom #tags"

	params := ConvertArticleParams{
		Title: &title,
		Body:  &body,
		Tags:  &tags,
	}

	if params.Title == nil || *params.Title != "Custom Title" {
		t.Errorf("expected Title to be Custom Title, got %v", params.Title)
	}

	if params.Body == nil || *params.Body != "Custom body content" {
		t.Errorf("expected Body to be Custom body content, got %v", params.Body)
	}

	if params.Tags == nil || *params.Tags != "#custom #tags" {
		t.Errorf("expected Tags to be #custom #tags, got %v", params.Tags)
	}
}
