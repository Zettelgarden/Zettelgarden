package models

import "time"

// RSSFeed represents a configured RSS feed subscription
type RSSFeed struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	URL            string     `json:"url"`
	Name           string     `json:"name"`
	Folder         *string    `json:"folder,omitempty"`
	AutoTags       string     `json:"auto_tags"`
	FetchInterval  int        `json:"fetch_interval"`
	LastFetchedAt  *time.Time `json:"last_fetched_at,omitempty"`
	LastError      *string    `json:"last_error,omitempty"`
	Enabled        bool       `json:"enabled"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// RSSArticle represents an article fetched from an RSS feed
type RSSArticle struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	FeedID      int        `json:"feed_id"`
	Title       string     `json:"title"`
	Content     *string    `json:"content,omitempty"`
	Author      *string    `json:"author,omitempty"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	Read        bool       `json:"read"`
	CardID      *int       `json:"card_id,omitempty"`
}

// RSSFolder represents a folder for organizing RSS feeds
type RSSFolder struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	Name       string `json:"name"`
	OrderIndex int    `json:"order_index"`
}

// CreateRSSFeedParams represents parameters for creating or updating an RSS feed
type CreateRSSFeedParams struct {
	URL           string  `json:"url"`
	Name          string  `json:"name"`
	Folder        *string `json:"folder,omitempty"`
	AutoTags      string  `json:"auto_tags,omitempty"`
	FetchInterval *int    `json:"fetch_interval,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
}

// UpdateRSSFeedParams represents parameters for updating an RSS feed
type UpdateRSSFeedParams struct {
	Name          *string `json:"name,omitempty"`
	Folder        *string `json:"folder,omitempty"`
	AutoTags      *string `json:"auto_tags,omitempty"`
	FetchInterval *int    `json:"fetch_interval,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
}

// ConvertArticleParams represents parameters for converting an RSS article to a card
type ConvertArticleParams struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
	Tags  *string `json:"tags,omitempty"`
}

// CreateRSSFolderParams represents parameters for creating an RSS folder
type CreateRSSFolderParams struct {
	Name       string `json:"name"`
	OrderIndex *int   `json:"order_index,omitempty"`
}
