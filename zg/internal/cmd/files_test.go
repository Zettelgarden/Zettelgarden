package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFileList(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]any{
				{"id": 1, "name": "notes.md", "size": 1234},
			},
			"page": 1, "per_page": 20, "total": 1, "total_pages": 1,
			"storage_used": 0, "max_storage": 0,
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runFileList(fileListCmd, nil); err != nil {
			t.Fatalf("runFileList: %v", err)
		}
	})
	if !strings.Contains(gotPath, "/api/files?page=1&per_page=20") {
		t.Errorf("unexpected request %q", gotPath)
	}
	if !strings.Contains(out, `"name":"notes.md"`) {
		t.Errorf("expected file in output, got %q", out)
	}
}

func TestRunFileGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/5" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": 5, "name": "paper.pdf", "filetype": "application/pdf"})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runFileGet(fileGetCmd, []string{"5"}); err != nil {
			t.Fatalf("runFileGet: %v", err)
		}
	})
	if !strings.Contains(out, `"name":"paper.pdf"`) {
		t.Errorf("expected file metadata, got %q", out)
	}
}

func TestRunFileDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/files/5" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runFileDelete(fileDeleteCmd, []string{"5"}); err != nil {
			t.Fatalf("runFileDelete: %v", err)
		}
	})
	if !strings.Contains(out, "File 5 deleted") {
		t.Errorf("expected delete message, got %q", out)
	}
}

func TestRunFileEdit(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/files/5" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": 5, "name": "renamed.md"})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	fileEditName = "renamed.md"
	fileEditCardPK = 7
	fileEditDesc = "hello"
	t.Cleanup(func() { fileEditName, fileEditCardPK, fileEditDesc = "", 0, "" })

	cmd := fileEditCmd
	cmd.Flags().Set("card-pk", "7")
	out := captureStdout(t, func() {
		if err := runFileEdit(cmd, []string{"5"}); err != nil {
			t.Fatalf("runFileEdit: %v", err)
		}
	})
	if gotBody["name"] != "renamed.md" || gotBody["card_pk"] != float64(7) || gotBody["description"] != "hello" {
		t.Errorf("edit body = %v", gotBody)
	}
	if !strings.Contains(out, `"name":"renamed.md"`) {
		t.Errorf("expected updated file, got %q", out)
	}
}

func TestRunFileTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/tags" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "receipts", "file_count": 3}})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runFileTags(fileTagsCmd, nil); err != nil {
			t.Fatalf("runFileTags: %v", err)
		}
	})
	if !strings.Contains(out, `"name":"receipts"`) {
		t.Errorf("expected file tag in output, got %q", out)
	}
}

func TestRunFileTag(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/files/5/tags" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runFileTag(fileTagCmd, []string{"5", "receipt", "travel"}); err != nil {
			t.Fatalf("runFileTag: %v", err)
		}
	})
	tags, _ := gotBody["tag_names"].([]any)
	if len(tags) != 2 || tags[0] != "receipt" || tags[1] != "travel" {
		t.Errorf("tag_names = %v", gotBody)
	}
	if !strings.Contains(out, "File 5 tagged receipt, travel") {
		t.Errorf("expected tag message, got %q", out)
	}
}

func TestRunFileUntag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/files/5/tags/receipt" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runFileUntag(fileUntagCmd, []string{"5", "#receipt"}); err != nil {
			t.Fatalf("runFileUntag: %v", err)
		}
	})
	if !strings.Contains(out, "Tag #receipt removed from file 5") {
		t.Errorf("expected untag message, got %q", out)
	}
}

func TestRunFileUpload(t *testing.T) {
	var gotContentType string
	var gotCardPK string
	var gotFileContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/files/upload" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		gotContentType = r.Header.Get("Content-Type")
		mr, err := r.MultipartReader()
		if err != nil {
			t.Errorf("multipart: %v", err)
		}
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			buf := make([]byte, 4096)
			n, _ := part.Read(buf)
			value := string(buf[:n])
			if part.FormName() == "file" {
				gotFileContent = value
			}
			if part.FormName() == "card_pk" {
				gotCardPK = value
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "File successfully uploaded",
			"file":    map[string]any{"id": 11, "name": "hello.txt"},
		})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	src := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(src, []byte("file contents here"), 0644); err != nil {
		t.Fatal(err)
	}
	fileUploadCard = 3
	t.Cleanup(func() { fileUploadCard = 0 })

	out := captureStdout(t, func() {
		if err := runFileUpload(fileUploadCmd, []string{src}); err != nil {
			t.Fatalf("runFileUpload: %v", err)
		}
	})
	if !strings.Contains(gotContentType, "multipart/form-data") {
		t.Errorf("expected multipart content type, got %q", gotContentType)
	}
	if gotFileContent != "file contents here" {
		t.Errorf("uploaded file content = %q", gotFileContent)
	}
	if gotCardPK != "3" {
		t.Errorf("card_pk field = %q, want 3", gotCardPK)
	}
	if !strings.Contains(out, `"name":"hello.txt"`) {
		t.Errorf("expected uploaded file in output, got %q", out)
	}
}

func TestRunFileDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/download/5" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="book.pdf"`)
		w.Write([]byte("pdf-bytes"))
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	dir := t.TempDir()
	fileDownloadTo = filepath.Join(dir, "saved.pdf")
	t.Cleanup(func() { fileDownloadTo = "" })

	out := captureStdout(t, func() {
		if err := runFileDownload(fileDownloadCmd, []string{"5"}); err != nil {
			t.Fatalf("runFileDownload: %v", err)
		}
	})
	data, err := os.ReadFile(fileDownloadTo)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != "pdf-bytes" {
		t.Errorf("downloaded content = %q, want pdf-bytes", data)
	}
	if !strings.Contains(out, "Downloaded file 5") {
		t.Errorf("expected download message, got %q", out)
	}
}

func TestContentDispositionFilename(t *testing.T) {
	cases := []struct{ header, want string }{
		{`attachment; filename="book.pdf"`, "book.pdf"},
		{`attachment; filename=book.pdf`, "book.pdf"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := contentDispositionFilename(tc.header); got != tc.want {
			t.Errorf("contentDispositionFilename(%q) = %q, want %q", tc.header, got, tc.want)
		}
	}
}

func TestRunFileImportEpub(t *testing.T) {
	var gotBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/files/9/import-epub" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "title": "Chapter One"}})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	fileEpubCardID = "3a"
	t.Cleanup(func() { fileEpubCardID = "" })

	out := captureStdout(t, func() {
		if err := runFileImportEpub(fileImportEpubCmd, []string{"9"}); err != nil {
			t.Fatalf("runFileImportEpub: %v", err)
		}
	})
	if gotBody["card_id"] != "3a" {
		t.Errorf("card_id sent = %q", gotBody["card_id"])
	}
	if !strings.Contains(out, `"title":"Chapter One"`) {
		t.Errorf("expected epub cards in output, got %q", out)
	}
}

func TestRunCardFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cards/3/files" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "name": "attachment.png"}})
	}))
	defer server.Close()
	writeTestConfig(t, server.URL)

	out := captureStdout(t, func() {
		if err := runCardFiles(cardFilesCmd, []string{"3"}); err != nil {
			t.Fatalf("runCardFiles: %v", err)
		}
	})
	if !strings.Contains(out, `"name":"attachment.png"`) {
		t.Errorf("expected card file in output, got %q", out)
	}
}

// guard against unused import if multipart helper changes
