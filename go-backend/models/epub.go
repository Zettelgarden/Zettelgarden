package models

// EpubMetadata contains extracted metadata from an epub file
type EpubMetadata struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Publisher   string `json:"publisher"`
	Year        string `json:"year"`
	Description string `json:"description"`
}

// EpubChapter represents a single chapter from an epub
type EpubChapter struct {
	Title string `json:"title"`
	Body  string `json:"body"` // Markdown content
}

// ImportEpubResponse is the API response for epub import
type ImportEpubResponse struct {
	ParentCardID int          `json:"parent_card_id"`
	ChildCardIDs []int        `json:"child_card_ids"`
	Metadata     EpubMetadata `json:"metadata"`
}
