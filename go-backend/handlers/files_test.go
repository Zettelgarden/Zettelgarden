package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/tests"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func uploadTestFile(s *Handler) {
	testFile, err := os.Open("../tests/test.txt")
	if err != nil {
		log.Fatal("unable to open test file")
		return
	}
	uuidKey := uuid.New().String()

	s.uploadObject(s.Server.S3, uuidKey, testFile.Name())

	query := `UPDATE files SET path = $1, filename = $2 WHERE id = 1`
	s.DB.QueryRow(query, uuidKey, uuidKey)
}

func TestGetAllFiles(t *testing.T) {
	s := setup()
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
	if len(response.Files) != 20 {
		t.Fatalf("wrong length of results, got %v want %v", len(response.Files), 20)
	}
}
func TestGetAllFilesNoToken(t *testing.T) {
	s := setup()
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
	s := setup()
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

	s := setup()
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
	s := setup()
	defer tests.Teardown()

	new_name := "new_name.txt"
	token, _ := tests.GenerateTestJWT(1)
	fileData := models.EditFileMetadataParams{
		Name:   new_name,
		CardPK: 1,
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
	if file.CardPK != 1 {
		t.Errorf("handler returned wrong file, got id %v want %v", file.ID, 1)
	}
}

func TestEditFileSuccessChangeCard(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	new_name := "new_name.txt"
	token, _ := tests.GenerateTestJWT(1)
	fileData := models.EditFileMetadataParams{
		Name:   new_name,
		CardPK: 2,
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
	if file.CardPK != 2 {
		t.Errorf("handler returned wrong file, got id %v want %v", file.ID, 2)
	}
}
func TestEditFileWrongUser(t *testing.T) {
	s := setup()
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
	s := setup()
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
		log.Printf(rr.Body.String())
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
	s := setup()
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
	s := setup()
	defer tests.Teardown()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	var count int
	err := s.DB.QueryRow("SELECT count(*) FROM files").Scan(&count)
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
	err = s.DB.QueryRow("SELECT count(*) FROM files").Scan(&newCount)
	if err != nil {
		log.Fatal(err)
	}
	if count != newCount {
		t.Errorf("function created a file when it shouldn't have. old count %v new count %v", count, newCount)
	}

}

func TestDownloadFile(t *testing.T) {
	s := setup()
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
}

func TestDeleteFile(t *testing.T) {
	s := setup()
	defer tests.Teardown()
	uploadTestFile(s)

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
}
