package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
