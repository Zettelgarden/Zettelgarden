package handlers

import (
	"io"
	"strings"
)

// snippetRadius is the number of characters kept on each side of the match.
const snippetRadius = 60

// buildFileSnippet finds the first occurrence of query in the file's name,
// description, tags, or extracted text and returns a short snippet around the
// match plus the field it matched. It returns ("", "") when there is no match.
// The field values let the UI label the result (e.g. "matched in content").
func buildFileSnippet(query, name, description, extractedText string, tags []string) (string, string) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return "", ""
	}

	if idx := strings.Index(strings.ToLower(name), q); idx >= 0 {
		return snippetAround(name, idx, len(q)), "name"
	}
	if idx := strings.Index(strings.ToLower(description), q); idx >= 0 {
		return snippetAround(description, idx, len(q)), "description"
	}
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return tag, "tag"
		}
	}
	if idx := strings.Index(strings.ToLower(extractedText), q); idx >= 0 {
		return snippetAround(extractedText, idx, len(q)), "content"
	}
	return "", ""
}

// snippetAround returns up to (2*snippetRadius + matchLen) characters around
// the match at text[start:start+matchLen], padded with ellipses when truncated.
func snippetAround(text string, start, matchLen int) string {
	end := start + matchLen
	from := start - snippetRadius
	if from < 0 {
		from = 0
	}
	to := end + snippetRadius
	if to > len(text) {
		to = len(text)
	}
	snippet := text[from:to]
	if from > 0 {
		snippet = "…" + snippet
	}
	if to < len(text) {
		snippet = snippet + "…"
	}
	return snippet
}

// maxExtractedText caps how many bytes of text are kept per file. Large enough
// for meaningful search + snippets, small enough to keep payloads and the
// Typesense document reasonable.
const maxExtractedText = 1 << 20 // 1 MiB

// isTextContentType reports whether a MIME type should be extracted as text.
func isTextContentType(contentType string) bool {
	ctype := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(ctype, "text/") {
		return true
	}
	switch ctype {
	case "application/json", "application/xml", "application/javascript",
		"application/x-javascript", "application/x-yaml", "application/yaml",
		"application/x-sh", "application/x-sql", "application/sql",
		"application/csv", "application/x-httpd-php", "application/x-httpd-php-source",
		"application/x-python", "application/x-perl", "application/x-ruby",
		"application/toml", "application/x-www-form-urlencoded":
		return true
	}
	return false
}

// extractTextFromFile reads up to maxExtractedText bytes from r and returns them
// as extracted text when the content type is text-like; otherwise "". The reader
// must be positioned at the start of the content.
func extractTextFromFile(contentType string, r io.Reader) string {
	if !isTextContentType(contentType) {
		return ""
	}
	buf := make([]byte, maxExtractedText+1)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return ""
	}
	if n > maxExtractedText {
		n = maxExtractedText
	}
	return string(buf[:n])
}
