package services

import (
	"bytes"
	"io"
	"log"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/xuri/excelize/v2"
)

// ExtractText extracts text content from various file types
func ExtractText(contentType string, reader io.Reader) (string, error) {
	switch {
	case strings.Contains(contentType, "pdf"):
		return extractFromPDF(reader)
	case strings.Contains(contentType, "word") || strings.Contains(contentType, "document"):
		return extractFromDocx(reader)
	case strings.Contains(contentType, "excel") || strings.Contains(contentType, "spreadsheet"):
		return extractFromXlsx(reader)
	case strings.HasPrefix(contentType, "text/"):
		return extractFromPlainText(reader)
	default:
		// Unsupported type - return empty string
		return "", nil
	}
}

func extractFromPDF(reader io.Reader) (string, error) {
	// Convert io.Reader to []byte for pdf library
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Limit size to prevent memory issues
	if len(data) > 50*1024*1024 { // 50MB
		data = data[:50*1024*1024]
	}

	pdfReader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()

	// Extract text from each page
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			log.Printf("Error extracting text from page %d: %v", i, err)
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")

		// Limit extracted text to 100KB
		if textBuilder.Len() > 100*1024 {
			textBuilder.WriteString("\n[TRUNCATED]")
			break
		}
	}

	return textBuilder.String(), nil
}

func extractFromDocx(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Use ReadDocxFromMemory with bytes.Reader (implements io.ReaderAt)
	doc, err := docx.ReadDocxFromMemory(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	defer doc.Close()

	editable := doc.Editable()
	content := editable.GetContent()

	// Limit to 100KB
	if len(content) > 100*1024 {
		return content[:100*1024] + "\n[TRUNCATED]", nil
	}

	return content, nil
}

func extractFromXlsx(reader io.Reader) (string, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var textBuilder strings.Builder
	sheets := f.GetSheetList()

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		for _, row := range rows {
			for _, cell := range row {
				textBuilder.WriteString(cell)
				textBuilder.WriteString(" ")
			}
			textBuilder.WriteString("\n")

			if textBuilder.Len() > 100*1024 {
				textBuilder.WriteString("\n[TRUNCATED]")
				return textBuilder.String(), nil
			}
		}
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

func extractFromPlainText(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Limit to 100KB
	if len(data) > 100*1024 {
		return string(data[:100*1024]) + "\n[TRUNCATED]", nil
	}

	return string(data), nil
}
