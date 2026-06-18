package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

// taskSavedSearchRouter mounts the saved-search route handlers behind JWT
// middleware on a fresh router so tests can hit them by path.
func taskSavedSearchRouter(s *Handler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/task-saved-searches", s.JwtMiddleware(s.GetTaskSavedSearchesRoute)).Methods("GET")
	r.HandleFunc("/api/task-saved-searches", s.JwtMiddleware(s.CreateTaskSavedSearchRoute)).Methods("POST")
	r.HandleFunc("/api/task-saved-searches/{id}", s.JwtMiddleware(s.GetTaskSavedSearchRoute)).Methods("GET")
	r.HandleFunc("/api/task-saved-searches/{id}", s.JwtMiddleware(s.UpdateTaskSavedSearchRoute)).Methods("PUT")
	r.HandleFunc("/api/task-saved-searches/{id}", s.JwtMiddleware(s.DeleteTaskSavedSearchRoute)).Methods("DELETE")
	return r
}

func doSavedSearchRequest(t *testing.T, router *mux.Router, method, path string, body interface{}, userID int) *httptest.ResponseRecorder {
	t.Helper()
	token, _ := tests.GenerateTestJWT(userID)
	var req *http.Request
	if body != nil {
		jsonData, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(jsonData))
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestTaskSavedSearchCRUD(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := taskSavedSearchRouter(s)

	// Start with no saved searches.
	rr := doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches", nil, 1)
	if rr.Code != http.StatusOK {
		t.Fatalf("list (empty) expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var initial []models.TaskSavedSearch
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &initial)
	if len(initial) != 0 {
		t.Fatalf("expected 0 saved searches, got %d", len(initial))
	}

	// Create a saved search.
	createBody := models.CreateTaskSavedSearchParams{
		Name: "Overdue", FilterString: "date:overdue", SortField: "priority", SortDirection: "desc", ViewMode: "list",
	}
	rr = doSavedSearchRequest(t, router, "POST", "/api/task-saved-searches", createBody, 1)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		ID int `json:"id"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected non-zero id")
	}

	// List should now contain it.
	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches", nil, 1)
	var list []models.TaskSavedSearch
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &list)
	if len(list) != 1 || list[0].Name != "Overdue" {
		t.Fatalf("unexpected list: %+v", list)
	}

	// Get by id.
	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches/"+itoa(created.ID), nil, 1)
	if rr.Code != http.StatusOK {
		t.Fatalf("get expected 200, got %d", rr.Code)
	}
	var fetched models.TaskSavedSearch
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &fetched)
	if fetched.FilterString != "date:overdue" || fetched.SortDirection != "desc" {
		t.Fatalf("unexpected fetched search: %+v", fetched)
	}

	// Update the name + filter.
	newName := "Overdue work"
	newFilter := "date:overdue #work"
	updateBody := models.UpdateTaskSavedSearchParams{
		Name:         &newName,
		FilterString: &newFilter,
	}
	rr = doSavedSearchRequest(t, router, "PUT", "/api/task-saved-searches/"+itoa(created.ID), updateBody, 1)
	if rr.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches/"+itoa(created.ID), nil, 1)
	var updated models.TaskSavedSearch
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &updated)
	if updated.Name != newName || updated.FilterString != newFilter || updated.SortDirection != "desc" {
		t.Fatalf("update not applied: %+v", updated)
	}

	// Delete.
	rr = doSavedSearchRequest(t, router, "DELETE", "/api/task-saved-searches/"+itoa(created.ID), nil, 1)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete expected 204, got %d", rr.Code)
	}

	// Get should now 404.
	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches/"+itoa(created.ID), nil, 1)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get after delete expected 404, got %d", rr.Code)
	}
}

func TestTaskSavedSearchValidation(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := taskSavedSearchRouter(s)

	// Missing name -> 400.
	rr := doSavedSearchRequest(t, router, "POST", "/api/task-saved-searches",
		models.CreateTaskSavedSearchParams{FilterString: "date:today"}, 1)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing name expected 400, got %d", rr.Code)
	}

	// Invalid sort_field -> 400.
	rr = doSavedSearchRequest(t, router, "POST", "/api/task-saved-searches",
		models.CreateTaskSavedSearchParams{Name: "Bad", SortField: "bogus"}, 1)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid sort_field expected 400, got %d", rr.Code)
	}

	// Defaults applied when enum fields omitted.
	rr = doSavedSearchRequest(t, router, "POST", "/api/task-saved-searches",
		models.CreateTaskSavedSearchParams{Name: "JustName", FilterString: "status:todo"}, 1)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create with defaults expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		ID int `json:"id"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &created)

	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches/"+itoa(created.ID), nil, 1)
	var fetched models.TaskSavedSearch
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &fetched)
	if fetched.SortField != "priority" || fetched.SortDirection != "asc" || fetched.ViewMode != "list" {
		t.Fatalf("defaults not applied: %+v", fetched)
	}
}

func TestTaskSavedSearchUserIsolation(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	router := taskSavedSearchRouter(s)

	// User 1 creates a search.
	rr := doSavedSearchRequest(t, router, "POST", "/api/task-saved-searches",
		models.CreateTaskSavedSearchParams{Name: "Mine", FilterString: "date:today"}, 1)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d", rr.Code)
	}
	var created struct {
		ID int `json:"id"`
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &created)

	// User 2 cannot see it in their list.
	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches", nil, 2)
	var list []models.TaskSavedSearch
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("user 2 should see no searches, got %d", len(list))
	}

	// User 2 cannot fetch/update/delete it by id.
	rr = doSavedSearchRequest(t, router, "GET", "/api/task-saved-searches/"+itoa(created.ID), nil, 2)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("cross-user get expected 404, got %d", rr.Code)
	}
	rr = doSavedSearchRequest(t, router, "DELETE", "/api/task-saved-searches/"+itoa(created.ID), nil, 2)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("cross-user delete expected 404, got %d", rr.Code)
	}
}

// itoa is a tiny strconv wrapper to keep call sites concise.
func itoa(i int) string {
	return strconv.Itoa(i)
}
