package services

import (
	"archive/zip"
	"fmt"
	"io"
	"regexp"
	"strings"

	epublib "github.com/ArcadiaLin/go-epub"
	"go-backend/models"
)

// Constants for epub processing configuration
const (
	// MinChapterContent is the minimum character count for a chapter to be included
	MinChapterContent = 200
)

// Error definitions
var (
	// ErrNoValidChapters is returned when an epub contains no chapters with sufficient content
	ErrNoValidChapters = fmt.Errorf("no valid chapters found in epub")

	// yearRegex extracts 4-digit years (1900-2099) from date strings
	yearRegex = regexp.MustCompile(`\b(19|20)\d{2}\b`)

	// brRegex matches <br> tags (with optional attributes and whitespace)
	brRegex = regexp.MustCompile(`(?i)<br\s*/?>`)
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

	// Open the EPUB as a ZIP to read chapter HTML directly
	epubZip, err := zip.OpenReader(filePath)
	if err != nil {
		return metadata, chapters, fmt.Errorf("failed to open epub as zip: %w", err)
	}
	defer epubZip.Close()

	// Extract chapters with proper line break handling
	chapters, err = extractChaptersWithLineBreaks(book, epubZip)
	if err != nil {
		return metadata, chapters, err
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

// extractYear parses year from date string using pre-compiled regex
func extractYear(date string) string {
	if date == "" {
		return ""
	}
	return yearRegex.FindString(date)
}

// extractChaptersWithLineBreaks extracts chapters with proper <br> handling
func extractChaptersWithLineBreaks(book *epublib.Book, epubZip *zip.ReadCloser) ([]models.EpubChapter, error) {
	var chapters []models.EpubChapter

	// Build a map of chapter paths for quick lookup
	chapterMap := make(map[string]*epublib.Chapter)
	for i := 0; i < book.ChapterCount(); i++ {
		chapter, err := book.ChapterByIndex(i)
		if err != nil {
			continue
		}
		// Normalize the path (remove leading slash if present)
		path := strings.TrimPrefix(chapter.Path, "/")
		chapterMap[path] = chapter
	}

	// Process each chapter
	for i := 0; i < book.ChapterCount(); i++ {
		chapter, err := book.ChapterByIndex(i)
		if err != nil {
			continue
		}

		// Get chapter title
		title := getChapterTitle(chapter, i+1)

		// Try to read the chapter HTML directly for better line break handling
		body := ""
		normalizedPath := strings.TrimPrefix(chapter.Path, "/")

		if htmlContent, err := readChapterHTML(epubZip, normalizedPath); err == nil {
			// Parse HTML with proper <br> handling
			body = htmlToMarkdown(htmlContent)
		} else {
			// Fall back to library's paragraph extraction
			body = paragraphsToMarkdown(chapter.Paragraphs)
		}

		// Skip chapters with very little content (likely front matter)
		if len(strings.TrimSpace(body)) < MinChapterContent {
			continue
		}

		chapters = append(chapters, models.EpubChapter{
			Title: title,
			Body:  body,
		})
	}

	// If no chapters found, return error
	if len(chapters) == 0 {
		return nil, ErrNoValidChapters
	}

	return chapters, nil
}

// readChapterHTML reads the raw HTML content of a chapter from the EPUB ZIP
func readChapterHTML(epubZip *zip.ReadCloser, chapterPath string) (string, error) {
	// Try to find the file in the ZIP
	for _, file := range epubZip.File {
		// Normalize paths for comparison
		zipPath := strings.TrimPrefix(file.Name, "/")
		if zipPath == chapterPath {
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}
	return "", fmt.Errorf("chapter file not found in zip: %s", chapterPath)
}

// htmlToMarkdown converts HTML content to markdown with proper line breaks
func htmlToMarkdown(htmlContent string) string {
	// Replace <br> tags with newlines (before stripping other tags)
	htmlContent = brRegex.ReplaceAllString(htmlContent, "\n")

	// Replace closing </p> and </div> tags with double newlines
	htmlContent = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(htmlContent, "\n\n")
	htmlContent = regexp.MustCompile(`(?i)</div>`).ReplaceAllString(htmlContent, "\n\n")

	// Strip all remaining HTML tags
	htmlContent = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(htmlContent, "")

	// Decode common HTML entities
	htmlContent = strings.ReplaceAll(htmlContent, "&nbsp;", " ")
	htmlContent = strings.ReplaceAll(htmlContent, "&amp;", "&")
	htmlContent = strings.ReplaceAll(htmlContent, "&lt;", "<")
	htmlContent = strings.ReplaceAll(htmlContent, "&gt;", ">")
	htmlContent = strings.ReplaceAll(htmlContent, "&quot;", "\"")
	htmlContent = strings.ReplaceAll(htmlContent, "&#39;", "'")
	htmlContent = strings.ReplaceAll(htmlContent, "&apos;", "'")

	// Clean up whitespace
	// First normalize all whitespace sequences
	htmlContent = regexp.MustCompile(`[ \t]+`).ReplaceAllString(htmlContent, " ")
	// Then normalize multiple newlines to at most two
	htmlContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(htmlContent, "\n\n")
	// Remove leading/trailing whitespace from each line
	lines := strings.Split(htmlContent, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	htmlContent = strings.Join(lines, "\n")

	// Final trim
	return strings.TrimSpace(htmlContent)
}

// getChapterTitle returns the chapter title or generates a default one
func getChapterTitle(chapter *epublib.Chapter, index int) string {
	if chapter.Title != "" {
		return chapter.Title
	}
	return fmt.Sprintf("Chapter %d", index)
}

// paragraphsToMarkdown converts a slice of paragraphs to markdown format (fallback)
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
