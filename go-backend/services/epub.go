package services

import (
	"fmt"
	"regexp"
	"strings"

	epublib "github.com/ArcadiaLin/go-epub"
	"go-backend/models"
)

// ParseEpub reads an epub file and extracts metadata and chapters
func ParseEpub(filePath string) (models.EpubMetadata, []models.EpubChapter, error) {
	var metadata models.EpubMetadata
	var chapters []models.EpubChapter

	// Open the epub file
	book, err := epublib.ReadBook(filePath)
	if err != nil {
		return metadata, chapters, fmt.Errorf("failed to open epub: %w", err)
	}

	// Extract metadata
	metadata = extractMetadata(book)

	// Extract chapters
	chapters, err = extractChapters(book)
	if err != nil {
		return metadata, chapters, fmt.Errorf("failed to extract chapters: %w", err)
	}

	return metadata, chapters, nil
}

// extractMetadata pulls book metadata from the epub
func extractMetadata(book *epublib.Book) models.EpubMetadata {
	metadata := models.EpubMetadata{}

	// Extract title
	if title, err := book.Title(); err == nil && title != "" {
		metadata.Title = title
	}

	// Extract author/creator
	if creator, err := book.Creator(); err == nil && creator != "" {
		metadata.Author = creator
	}

	// Extract publisher
	if publisher, err := book.Publisher(); err == nil && publisher != "" {
		metadata.Publisher = publisher
	}

	// Extract date and parse year
	if date, err := book.Date(); err == nil && date != "" {
		metadata.Year = extractYear(date)
	}

	// Extract description
	if description, err := book.Description(); err == nil && description != "" {
		metadata.Description = description
	}

	return metadata
}

// extractYear parses year from date string
func extractYear(date string) string {
	if date == "" {
		return ""
	}
	// Try to extract 4-digit year
	re := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	return re.FindString(date)
}

// extractChapters gets all chapters from the epub
func extractChapters(book *epublib.Book) ([]models.EpubChapter, error) {
	var chapters []models.EpubChapter

	// Get all chapters from the book
	for i := 0; i < book.ChapterCount(); i++ {
		chapter, err := book.ChapterByIndex(i)
		if err != nil {
			// Log error but continue with other chapters
			continue
		}

		// Skip chapters with very little content (likely front matter)
		text := chapter.Text()
		if len(strings.TrimSpace(text)) < 200 {
			continue
		}

		// Create chapter with markdown-formatted body
		epubChapter := models.EpubChapter{
			Title: getChapterTitle(chapter, i+1),
			Body:  paragraphsToMarkdown(chapter.Paragraphs),
		}

		chapters = append(chapters, epubChapter)
	}

	// If no chapters found, return error
	if len(chapters) == 0 {
		return nil, fmt.Errorf("no valid chapters found in epub")
	}

	return chapters, nil
}

// getChapterTitle returns the chapter title or generates a default one
func getChapterTitle(chapter *epublib.Chapter, index int) string {
	if chapter.Title != "" {
		return chapter.Title
	}
	return fmt.Sprintf("Chapter %d", index)
}

// paragraphsToMarkdown converts a slice of paragraphs to markdown format
func paragraphsToMarkdown(paragraphs []string) string {
	if len(paragraphs) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		sb.WriteString(p)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}
