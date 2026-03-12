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

	// MaxChapterHTMLSize is the maximum size of a chapter HTML file (1MB)
	// This prevents memory exhaustion from malformed or malicious EPUBs
	MaxChapterHTMLSize = 1 * 1024 * 1024
)

// Error definitions
var (
	// ErrNoValidChapters is returned when an epub contains no chapters with sufficient content
	ErrNoValidChapters = fmt.Errorf("no valid chapters found in epub")

	// ErrChapterTooLarge is returned when a chapter exceeds the size limit
	ErrChapterTooLarge = fmt.Errorf("chapter file exceeds maximum size")

	// ErrInvalidChapterPath is returned when a chapter path is invalid or potentially malicious
	ErrInvalidChapterPath = fmt.Errorf("invalid chapter path")
)

// Pre-compiled regex patterns for performance and security
var (
	// yearRegex extracts 4-digit years (1900-2099) from date strings
	yearRegex = regexp.MustCompile(`\b(19|20)\d{2}\b`)

	// brRegex matches <br> tags (with optional attributes and whitespace)
	brRegex = regexp.MustCompile(`(?i)<br\s*/?>`)

	// pathTraversalRegex matches path traversal sequences
	pathTraversalRegex = regexp.MustCompile(`\.\.`)

	// paragraphCloseRegex matches closing paragraph tags
	paragraphCloseRegex = regexp.MustCompile(`(?i)</p>`)

	// divCloseRegex matches closing div tags
	divCloseRegex = regexp.MustCompile(`(?i)</div>`)

	// htmlTagRegex matches any HTML tag
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

	// whitespaceRegex matches multiple spaces/tabs
	whitespaceRegex = regexp.MustCompile(`[ \t]+`)

	// multiNewlineRegex matches 3+ consecutive newlines
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
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

		// Validate and sanitize the chapter path before reading
		if err := validateChapterPath(chapter.Path); err == nil {
			if htmlContent, err := readChapterHTML(epubZip, chapter.Path); err == nil {
				// Parse HTML with proper <br> handling
				body = htmlToMarkdown(htmlContent)
			}
		}

		// Fall back to library's paragraph extraction if direct read fails
		if body == "" {
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

// validateChapterPath checks that a chapter path is safe to read
// It prevents path traversal attacks and validates the path format
func validateChapterPath(path string) error {
	if path == "" {
		return ErrInvalidChapterPath
	}

	// Normalize the path
	normalized := strings.TrimPrefix(path, "/")

	// Check for path traversal attempts
	if pathTraversalRegex.MatchString(normalized) {
		return ErrInvalidChapterPath
	}

	// Check for absolute paths (shouldn't happen but be safe)
	if strings.HasPrefix(path, "/") && strings.Contains(path[1:], ":") {
		return ErrInvalidChapterPath
	}

	return nil
}

// readChapterHTML reads the raw HTML content of a chapter from the EPUB ZIP
// with security constraints to prevent memory exhaustion and path traversal
func readChapterHTML(epubZip *zip.ReadCloser, chapterPath string) (string, error) {
	// Normalize paths for comparison
	normalizedPath := strings.TrimPrefix(chapterPath, "/")

	// Try to find the file in the ZIP
	for _, file := range epubZip.File {
		// Normalize ZIP path for comparison
		zipPath := strings.TrimPrefix(file.Name, "/")

		if zipPath != normalizedPath {
			continue
		}

		// Check file size before opening (prevent ZIP bombs)
		if file.UncompressedSize64 > MaxChapterHTMLSize {
			return "", ErrChapterTooLarge
		}

		rc, err := file.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()

		// Use LimitReader to prevent reading more than expected
		// This is a defense-in-depth measure even though we checked UncompressedSize64
		limitedReader := io.LimitReader(rc, MaxChapterHTMLSize+1)

		content, err := io.ReadAll(limitedReader)
		if err != nil {
			return "", err
		}

		// Double-check we didn't exceed the limit
		if len(content) > MaxChapterHTMLSize {
			return "", ErrChapterTooLarge
		}

		return string(content), nil
	}

	return "", fmt.Errorf("chapter file not found in zip: %s", chapterPath)
}

// htmlToMarkdown converts HTML content to markdown with proper line breaks
func htmlToMarkdown(htmlContent string) string {
	// Replace <br> tags with newlines (before stripping other tags)
	htmlContent = brRegex.ReplaceAllString(htmlContent, "\n")

	// Replace closing </p> and </div> tags with double newlines
	htmlContent = paragraphCloseRegex.ReplaceAllString(htmlContent, "\n\n")
	htmlContent = divCloseRegex.ReplaceAllString(htmlContent, "\n\n")

	// Strip all remaining HTML tags
	htmlContent = htmlTagRegex.ReplaceAllString(htmlContent, "")

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
	htmlContent = whitespaceRegex.ReplaceAllString(htmlContent, " ")
	// Then normalize multiple newlines to at most two
	htmlContent = multiNewlineRegex.ReplaceAllString(htmlContent, "\n\n")
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
