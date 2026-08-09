package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func uploadTestFile(s *Handler) {
	testFile, err := os.Open("../tests/test.txt")
	if err != nil {
		log.Fatal("unable to open test file")
		return
	}
	defer testFile.Close()
	uuidKey := uuid.New().String()

	// Real round-trip through the store (D8): the blob is actually written to
	// the tempdir-backed LocalStore, so TestDownloadFile can stream it back.
	if err := s.Server.Store.Upload(context.Background(), uuidKey, testFile); err != nil {
		log.Fatal("unable to upload test file:", err)
	}

	query := `UPDATE files SET path = $1, filename = $2 WHERE id = 1`
	_, err = s.Server.Tx.Exec(query, uuidKey, uuidKey)
	if err != nil {
		log.Fatal("unable to update file path:", err)
	}
}

func TestGetAllFiles(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/files", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetAllFilesRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Files   []models.File `json:"files"`
		Page    int           `json:"page"`
		PerPage int           `json:"per_page"`
		Total   int           `json:"total"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if len(response.Files) != 5 {
		t.Fatalf("wrong length of results, got %v want %v (after test data reduction)", len(response.Files), 5)
	}
}

// TestGetAllFilesEmptyResult guards against the nil-slice bug where a filter
// with zero matches serialized `files` as JSON null instead of []. The frontend
// expects an array (it reads .length), so null crashes the FileVault render.
func TestGetAllFilesEmptyResult(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/files?filetype=document", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetAllFilesRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if strings.Contains(rr.Body.String(), `"files":null`) {
		t.Errorf("response serialized files as null (nil slice bug): %s", rr.Body.String())
	}

	var response struct {
		Files []models.File `json:"files"`
		Page  int           `json:"page"`
		Total int           `json:"total"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if response.Total != 0 {
		t.Errorf("expected 0 total for filetype=document, got %d", response.Total)
	}
	if response.Files == nil {
		t.Errorf("files must be an empty array, got nil (JSON null)")
	}
	if len(response.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(response.Files))
	}
}
func TestGetAllFilesNoToken(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token := ""
	req, err := http.NewRequest("GET", "/api/files", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetAllFilesRoute))
	handler.ServeHTTP(rr, req)

	//	print("%v", rr.Code)
	if status := rr.Code; status == http.StatusOK {
		t.Errorf("handler returned wrong status code, got %v want %v", rr.Code, http.StatusBadRequest)
	}
	if rr.Body.String() != "Invalid token\n" {
		t.Errorf("handler returned wrong body, got %v want %v", rr.Body.String(), "Invalid token")
	}
}

func TestGetFileSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/files/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware((s.GetFileMetadataRoute)))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var file models.File
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &file)
	if file.ID != 1 {
		t.Errorf("handler returned wrong file, got %v want %v", file.ID, 1)
	}

}

func TestGetFileWrongUser(t *testing.T) {

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)

	req, err := http.NewRequest("GET", "/api/files/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware((s.GetFileMetadataRoute)))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
	if rr.Body.String() != "unable to access file\n" {
		t.Errorf("handler returned wrong body, got %v want %v", rr.Body.String(), "unable to access file\n")
	}
}

func TestEditFileSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	new_name := "new_name.txt"
	token, _ := tests.GenerateTestJWT(1)
	cardPK := 1
	fileData := models.EditFileMetadataParams{
		Name:   new_name,
		CardPK: &cardPK,
	}
	body, err := json.Marshal(fileData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/files/1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.EditFileMetadataRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var file models.File
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &file)
	if file.Name != new_name {
		t.Errorf("handler returned wrong file name, got %v want %v", file.Name, new_name)
	}
	if file.CardPK == nil || *file.CardPK != 1 {
		t.Errorf("handler returned wrong file, got id %v want %v", file.ID, 1)
	}
}

func TestEditFileSuccessChangeCard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	new_name := "new_name.txt"
	token, _ := tests.GenerateTestJWT(1)
	cardPK := 2
	fileData := models.EditFileMetadataParams{
		Name:   new_name,
		CardPK: &cardPK,
	}
	body, err := json.Marshal(fileData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/files/1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.EditFileMetadataRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var file models.File
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &file)
	if file.Name != new_name {
		t.Errorf("handler returned wrong file name, got %v want %v", file.Name, new_name)
	}
	if file.CardPK == nil || *file.CardPK != 2 {
		t.Errorf("handler returned wrong file, got id %v want %v", file.ID, 2)
	}
}

func TestEditFileDescriptionSave(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	cardPK := 1
	description := "a useful note about this file"
	fileData := models.EditFileMetadataParams{
		Name:        "described.txt",
		CardPK:      &cardPK,
		Description: &description,
	}
	body, err := json.Marshal(fileData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/files/1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.EditFileMetadataRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var file models.File
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &file)
	if file.Description == nil || *file.Description != description {
		t.Errorf("response description = %v, want %q", file.Description, description)
	}

	// Verify the value was actually written to the DB
	var storedDescription sql.NullString
	err = s.GetDB().QueryRow("SELECT description FROM files WHERE id = 1").Scan(&storedDescription)
	if err != nil {
		t.Fatalf("failed to read description from DB: %v", err)
	}
	if !storedDescription.Valid || storedDescription.String != description {
		t.Errorf("DB description = %v, want %q", storedDescription, description)
	}
}

func TestEditFileDescriptionClear(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Seed an existing description, then clear it explicitly
	_, err := s.GetDB().Exec("UPDATE files SET description = 'existing note' WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}

	token, _ := tests.GenerateTestJWT(1)
	cardPK := 1
	description := ""
	fileData := models.EditFileMetadataParams{
		Name:        "cleared.txt",
		CardPK:      &cardPK,
		Description: &description,
	}
	body, err := json.Marshal(fileData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/files/1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.EditFileMetadataRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var file models.File
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &file)
	if file.Description != nil {
		t.Errorf("response description = %v, want nil (cleared)", *file.Description)
	}

	var storedDescription sql.NullString
	err = s.GetDB().QueryRow("SELECT description FROM files WHERE id = 1").Scan(&storedDescription)
	if err != nil {
		t.Fatalf("failed to read description from DB: %v", err)
	}
	if storedDescription.Valid {
		t.Errorf("DB description = %q, want NULL after clear", storedDescription.String)
	}
}

func TestEditFileDescriptionOmittedKeepsExisting(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Seed an existing description, then rename without sending description
	_, err := s.GetDB().Exec("UPDATE files SET description = 'keep me' WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}

	token, _ := tests.GenerateTestJWT(1)
	cardPK := 1
	fileData := models.EditFileMetadataParams{
		Name:   "renamed.txt",
		CardPK: &cardPK,
	}
	body, err := json.Marshal(fileData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/files/1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.EditFileMetadataRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var storedDescription sql.NullString
	err = s.GetDB().QueryRow("SELECT description FROM files WHERE id = 1").Scan(&storedDescription)
	if err != nil {
		t.Fatalf("failed to read description from DB: %v", err)
	}
	if !storedDescription.Valid || storedDescription.String != "keep me" {
		t.Errorf("DB description = %v, want %q (unchanged by rename)", storedDescription, "keep me")
	}
}
func TestEditFileWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	new_name := "new_name.txt"
	token, _ := tests.GenerateTestJWT(2)
	fileData := models.EditFileMetadataParams{
		Name: new_name,
	}
	body, err := json.Marshal(fileData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/files/1", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.EditFileMetadataRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		log.Printf("%v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
	if rr.Body.String() != "unable to access file\n" {
		t.Errorf("handler returned wrong body, got %v want %v", rr.Body.String(), "unable to access file\n")
	}
}

func createTestFile(t *testing.T, buffer bytes.Buffer, writer *multipart.Writer) {
	// Add file field
	fileWriter, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	// Open a test file to upload
	testFile, err := os.Open("../tests/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer testFile.Close()

	// Copy the file content to the form field
	_, err = io.Copy(fileWriter, testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Add card_pk field
	err = writer.WriteField("card_pk", "1")
	if err != nil {
		t.Fatal(err)
	}

	// Close the writer to finalize the multipart form
	writer.Close()

}

func TestUploadFileSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a buffer to write our multipart form data
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	createTestFile(t, buffer, writer)

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("POST", "/api/files/upload", &buffer)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UploadFileRoute))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		log.Printf("Response body: %s", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var response models.UploadFileResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.File.Name != "test.txt" {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), "File uploaded successfully")
	}
}

func TestUploadFileNoFile(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("POST", "/api/files/upload", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UploadFileRoute))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestUploadFileNotAllowed(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	var count int
	err := s.Server.Tx.QueryRow("SELECT count(*) FROM files").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	// Add file field
	fileWriter, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	// Open a test file to upload
	testFile, err := os.Open("../tests/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer testFile.Close()

	// Copy the file content to the form field
	_, err = io.Copy(fileWriter, testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Add card_pk field
	err = writer.WriteField("card_pk", "1")
	if err != nil {
		t.Fatal(err)
	}

	// Close the writer to finalize the multipart form
	writer.Close()

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("POST", "/api/files/upload", &buffer)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UploadFileRoute))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}
	var newCount int
	err = s.Server.Tx.QueryRow("SELECT count(*) FROM files").Scan(&newCount)
	if err != nil {
		log.Fatal(err)
	}
	if count != newCount {
		t.Errorf("function created a file when it shouldn't have. old count %v new count %v", count, newCount)
	}

}

func TestDownloadFile(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	uploadTestFile(s)

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("POST", "/api/files/download/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/download/{id}", s.JwtMiddleware((s.DownloadFileRoute)))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// D8: now that the store is real and the Server.Testing short-circuit is
	// gone, the route streams the actual bytes we uploaded in uploadTestFile
	// (../tests/test.txt == "hello world").
	if got := rr.Body.String(); got != "hello world" {
		t.Errorf("download body = %q, want %q", got, "hello world")
	}
}

func TestDeleteFile(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	uploadTestFile(s)

	// The fixture inserts file rows directly (bypassing the upload route's
	// IncrementUserFileCount), so seed the counter the way a real upload would
	// (Zettelgarden-y6s: file_count must decrement on soft-delete).
	services.IncrementUserFileCount(s.GetDB(), 1)
	var before int
	if err := s.GetDB().QueryRow(`SELECT file_count FROM user_stats WHERE user_id = 1`).Scan(&before); err != nil {
		t.Fatalf("read file_count before delete: %v", err)
	}
	if before != 1 {
		t.Fatalf("file_count before delete = %d, want 1", before)
	}

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("DELETE", "/api/files/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.DeleteFileRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	req, err = http.NewRequest("GET", "/api/files/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr = httptest.NewRecorder()
	router = mux.NewRouter()
	router.HandleFunc("/api/files/{id}", s.JwtMiddleware(s.GetFileMetadataRoute))
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)

	}

	// Zettelgarden-y6s: the soft-delete path must decrement file_count.
	var after int
	if err := s.GetDB().QueryRow(`SELECT file_count FROM user_stats WHERE user_id = 1`).Scan(&after); err != nil {
		t.Fatalf("read file_count after delete: %v", err)
	}
	if after != 0 {
		t.Errorf("file_count after delete = %d, want 0 (decrement on soft-delete)", after)
	}
}

// TestRunFileListQueryWithCandidateIDs covers the Typesense-filtered path
// (Zettelgarden-72f.3): a candidate ID set from Typesense is narrowed by the
// shared SQL query without needing a live Typesense instance.
func TestRunFileListQueryWithCandidateIDs(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	files, total, storageUsed, maxStorage, err := s.runFileListQuery(fileListQuery{
		UserID:    1,
		FileIDs:   []int{2, 1},
		SortBy:    "name",
		SortOrder: "asc",
		Page:      1,
		PerPage:   20,
	})
	if err != nil {
		t.Fatalf("runFileListQuery returned error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	seen := map[int]bool{}
	for _, f := range files {
		seen[f.ID] = true
		// Description + card are populated like the plain SQL path
		if f.Card.ID == 0 {
			t.Errorf("file %d card not populated", f.ID)
		}
	}
	if !seen[1] || !seen[2] {
		t.Errorf("expected files 1 and 2, got %v", seen)
	}
	if storageUsed <= 0 {
		t.Errorf("storage_used = %d, want > 0", storageUsed)
	}
	if maxStorage <= 0 {
		t.Errorf("max_storage = %d, want > 0", maxStorage)
	}
}

// TestRunFileListQueryEmptyCandidates guards the empty-set short circuit: a
// Typesense search with no hits must return an empty array (not null) with a
// computed quota.
func TestRunFileListQueryEmptyCandidates(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	files, total, storageUsed, maxStorage, err := s.runFileListQuery(fileListQuery{
		UserID:    1,
		FileIDs:   []int{},
		SortBy:    "date",
		SortOrder: "desc",
		Page:      1,
		PerPage:   20,
	})
	if err != nil {
		t.Fatalf("runFileListQuery returned error: %v", err)
	}
	if files == nil {
		t.Errorf("files must be an empty array, got nil")
	}
	if len(files) != 0 || total != 0 {
		t.Errorf("expected 0 files/0 total, got %d/%d", len(files), total)
	}
	if maxStorage <= 0 {
		t.Errorf("max_storage = %d, want > 0 (quota must survive an empty search)", maxStorage)
	}
	if storageUsed <= 0 {
		t.Errorf("storage_used = %d, want > 0", storageUsed)
	}
}

// TestGetAllFilesUnlinkedFilter checks the SQL path honors the unlinked param.
func TestGetAllFilesUnlinkedFilter(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Fixture files all have card_pk 1..5; unlink file 5.
	_, err := s.GetDB().Exec("UPDATE files SET card_pk = -1 WHERE id = 5")
	if err != nil {
		t.Fatal(err)
	}

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/files?unlinked=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetAllFilesRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var response struct {
		Files []models.File `json:"files"`
		Total int           `json:"total"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if response.Total != 1 {
		t.Errorf("total = %d, want 1", response.Total)
	}
	if len(response.Files) != 1 || response.Files[0].ID != 5 {
		t.Errorf("expected only file 5, got %+v", response.Files)
	}
}

// TestGetAllFilesFiletypeFilter checks the SQL path honors the filetype param.
func TestGetAllFilesFiletypeFilter(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	_, err := s.GetDB().Exec("UPDATE files SET type = 'application/pdf' WHERE id = 3")
	if err != nil {
		t.Fatal(err)
	}

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/files?filetype=pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetAllFilesRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var response struct {
		Files []models.File `json:"files"`
		Total int           `json:"total"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if response.Total != 1 {
		t.Errorf("total = %d, want 1", response.Total)
	}
	if len(response.Files) != 1 || response.Files[0].ID != 3 {
		t.Errorf("expected only file 3, got %+v", response.Files)
	}
}

// TestUploadFileExtractsText verifies inline text extraction on upload: a
// text/plain file gets its content stored in extracted_text (Zettelgarden-72f.10).
func TestUploadFileExtractsText(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// Build the file part with an explicit text/plain content type.
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="notes.txt"`)
	header.Set("Content-Type", "text/plain")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("the quick brown fox\njumps over the lazy dog")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("card_pk", "-1"); err != nil {
		t.Fatal(err)
	}
	writer.Close()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("POST", "/api/files/upload", &buffer)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UploadFileRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("handler returned wrong status code: got %v want %v: %s", status, http.StatusCreated, rr.Body.String())
	}
	var response models.UploadFileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.File.ExtractedText == nil || !strings.Contains(*response.File.ExtractedText, "quick brown fox") {
		t.Errorf("extracted_text = %v, want content containing %q", response.File.ExtractedText, "quick brown fox")
	}
}

// TestRunFileListQuerySnippet verifies search results carry a server-computed
// snippet + field around the match (Zettelgarden-72f.10).
func TestRunFileListQuerySnippet(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Seed a name, description and extracted text on file 3. The name contains
	// "budget" (so the SQL LIKE path returns it); the extracted text contains
	// "quarterly" (so a Typesense content match would return it via FileIDs).
	_, err := s.GetDB().Exec("UPDATE files SET name = 'budget-notes.md', description = 'a review doc', extracted_text = 'expense report for the quarterly planning meeting' WHERE id = 3")
	if err != nil {
		t.Fatal(err)
	}

	// SQL path: match against the name.
	files, _, _, _, err := s.runFileListQuery(fileListQuery{
		UserID:        1,
		SearchPattern: "%budget%",
		SearchTerm:    "budget",
		SortBy:        "date",
		SortOrder:     "desc",
		Page:          1,
		PerPage:       20,
	})
	if err != nil {
		t.Fatalf("runFileListQuery returned error: %v", err)
	}
	var matched *models.File
	for i := range files {
		if files[i].ID == 3 {
			matched = &files[i]
		}
	}
	if matched == nil {
		t.Fatalf("file 3 not in results")
	}
	if matched.Snippet == nil || matched.SnippetField == nil {
		t.Fatalf("file 3 snippet = %v / %v, want a match", matched.Snippet, matched.SnippetField)
	}
	if *matched.SnippetField != "name" {
		t.Errorf("snippet_field = %q, want %q", *matched.SnippetField, "name")
	}
	if !strings.Contains(*matched.Snippet, "budget") {
		t.Errorf("snippet %q should contain the match", *matched.Snippet)
	}
	if matched.ExtractedText == nil {
		t.Errorf("extracted_text should be included on search results")
	}

	// Typesense-filtered path: the candidate ID came from a content match, so
	// the snippet should come from extracted_text.
	files, _, _, _, err = s.runFileListQuery(fileListQuery{
		UserID:     1,
		FileIDs:    []int{3},
		SearchTerm: "quarterly",
		SortBy:     "date",
		SortOrder:  "desc",
		Page:       1,
		PerPage:    20,
	})
	if err != nil {
		t.Fatalf("runFileListQuery returned error: %v", err)
	}
	if len(files) != 1 || files[0].ID != 3 {
		t.Fatalf("expected file 3 only, got %+v", files)
	}
	if files[0].SnippetField == nil || *files[0].SnippetField != "content" {
		t.Errorf("snippet_field = %v, want %q", files[0].SnippetField, "content")
	}
	if files[0].Snippet == nil || !strings.Contains(*files[0].Snippet, "quarterly") {
		t.Errorf("snippet = %v, want content containing %q", files[0].Snippet, "quarterly")
	}

	// No match -> no snippet on any result.
	files, _, _, _, err = s.runFileListQuery(fileListQuery{
		UserID:        1,
		SearchPattern: "%zzzznomatch%",
		SearchTerm:    "zzzznomatch",
		SortBy:        "date",
		SortOrder:     "desc",
		Page:          1,
		PerPage:       20,
	})
	if err != nil {
		t.Fatalf("runFileListQuery returned error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 results for non-matching search, got %d", len(files))
	}
}
