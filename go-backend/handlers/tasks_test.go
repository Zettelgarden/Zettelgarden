package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func makeTaskRequestSuccess(s *Handler, t *testing.T, id int) *httptest.ResponseRecorder {

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/tasks/"+strconv.Itoa(id), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(id))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{id}", s.JwtMiddleware(s.GetTaskRoute))
	router.ServeHTTP(rr, req)

	return rr
}

func makeTasksRequestSuccess(s *Handler, t *testing.T, params string) *httptest.ResponseRecorder {

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/tasks/?"+params, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetTasksRoute))
	handler.ServeHTTP(rr, req)

	return rr
}
func TestGetTaskSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	rr := makeTaskRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var task models.Task
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)
	if task.ID != 1 {
		t.Errorf("handler returned wrong task, got %v want %v", task.ID, 1)
	}
	if task.UserID != 1 {
		t.Errorf("handler returned task for wrong user, got %v want %v", task.UserID, 1)
	}
}

func TestGetTaskWrongUser(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)

	req, err := http.NewRequest("GET", "/api/tasks/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{id}", s.JwtMiddleware(s.GetTaskRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
	if rr.Body.String() != "unable to access task\n" {
		t.Errorf("handler returned wrong body, got %v want %v", rr.Body.String(), "unable to access task\n")
	}

}

func TestGetTasksSuccess(t *testing.T) {

	s := setup()
	defer tests.Teardown()

	rr := makeTasksRequestSuccess(s, t, "")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var response models.TasksResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	if len(response.Tasks) != 19 {
		t.Errorf("wrong number of tasks returned, got %v want %v", len(response.Tasks), 19)
	}
}

func TestUpdateTaskSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	expected := "ooga booga"

	rr := makeCardRequestSuccess(s, t, 1)
	var task models.Task
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)

	if task.Title == expected {
		t.Errorf("something is wrong, title has changed already")
	}
	task.Title = expected
	jsonData, err := json.Marshal(task)

	req, err := http.NewRequest("PUT", "/api/tasks/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{id}", s.JwtMiddleware(s.UpdateTaskRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	rr = makeTaskRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)
	if task.Title != expected {
		t.Errorf("handler return wrong title, got %v want %v", task.Title, expected)
	}
}

func TestUpdateTaskCompleteTask(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	rr := makeCardRequestSuccess(s, t, 1)
	var task models.Task
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)

	if task.IsComplete {
		t.Errorf("something is wrong, task already complete")
	}
	task.IsComplete = true
	jsonData, _ := json.Marshal(task)

	req, err := http.NewRequest("PUT", "/api/tasks/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{id}", s.JwtMiddleware(s.UpdateTaskRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	rr = makeTaskRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)
	if !task.IsComplete {
		t.Errorf("task should be complete, is not")
	}
	if task.CompletedAt == nil || task.CompletedAt.IsZero() {
		t.Errorf("completed_at not set properly upon task completion")
	}
}

func TestCreateTaskSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	var task models.Task
	var newTask models.Task
	token, _ := tests.GenerateTestJWT(1)

	expectedTitle := "Test Task"
	expectedScheduledDate := time.Date(2024, 7, 5, 10, 0, 0, 0, time.UTC)
	expectedDueDate := time.Date(2024, 7, 10, 10, 0, 0, 0, time.UTC)
	expected := models.Task{
		CardPK:        1,
		ScheduledDate: &expectedScheduledDate,
		DueDate:       &expectedDueDate,
		Title:         expectedTitle,
		IsComplete:    false,
	}
	jsonData, _ := json.Marshal(expected)
	req, err := http.NewRequest("POST", "/api/tasks/", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateTaskRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)

	rr = makeTaskRequestSuccess(s, t, task.ID)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &newTask)
	if newTask.Title != expectedTitle {
		t.Errorf("handler returned wrong task title: got %v want %v", newTask.Title, expectedTitle)
	}
	if newTask.ScheduledDate == nil || !newTask.ScheduledDate.Equal(expectedScheduledDate) {
		t.Errorf("handler returned wrong task scheduled date: got %v want %v", newTask.ScheduledDate, expectedScheduledDate)
	}
	if newTask.DueDate == nil || !newTask.DueDate.Equal(expectedDueDate) {
		t.Errorf("handler returned wrong task due date: got %v want %v", newTask.DueDate, expectedDueDate)
	}
}

func TestDeleteTaskSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Now, delete the created task
	deleteReq, err := http.NewRequest("DELETE", "/api/tasks/"+strconv.Itoa(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	deleteReq.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{id}", s.JwtMiddleware(s.DeleteTaskRoute))
	router.ServeHTTP(rr, deleteReq)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Verify the task is marked as deleted
	getReq, err := http.NewRequest("GET", "/api/tasks/"+strconv.Itoa(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	getReq.Header.Set("Authorization", "Bearer "+token)

	rr = makeTaskRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestParseRecurringTasks(t *testing.T) {
	_ = setup()
	defer tests.Teardown()

	testCases := []struct {
		name     string
		input    string
		expected models.RecurringTask
		found    bool
	}{
		{
			name:     "every day",
			input:    "hello world every day",
			expected: models.RecurringTask{Frequency: "daily", Interval: 1},
			found:    true,
		},
		{
			name:     "daily",
			input:    "hello world daily",
			expected: models.RecurringTask{Frequency: "daily", Interval: 1},
			found:    true,
		},
		{
			name:     "every 3 days",
			input:    "hello world every 3 days",
			expected: models.RecurringTask{Frequency: "daily", Interval: 3},
			found:    true,
		},
		{
			name:     "weekly",
			input:    "hello world every week",
			expected: models.RecurringTask{Frequency: "weekly", Interval: 7},
			found:    true,
		},
		{
			name:     "every 5 weeks",
			input:    "hello world every 5 weeks",
			expected: models.RecurringTask{Frequency: "weekly", Interval: 5},
			found:    true,
		},
		{
			name:     "monthly",
			input:    "hello world every month",
			expected: models.RecurringTask{Frequency: "monthly", Interval: 30},
			found:    true,
		},
		{
			name:     "every 6 months",
			input:    "hello world every 6 months",
			expected: models.RecurringTask{Frequency: "monthly", Interval: 6},
			found:    true,
		},
		{
			name:     "no recurring task",
			input:    "hello world",
			expected: models.RecurringTask{},
			found:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, found := services.ParseRecurringTasks(tc.input)
			if found != tc.found {
				t.Errorf("expected found to be %v, but got %v", tc.found, found)
			}
			if !reflect.DeepEqual(output, tc.expected) {
				t.Errorf("expected %+v, but got %+v", tc.expected, output)
			}
		})
	}
}
func TestUpdateTaskCompleteRecurringTask(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	var taskCount int
	_ = s.DB.QueryRow("SELECT count(*) FROM tasks").Scan(&taskCount)
	if taskCount <= 0 {
		t.Errorf("wrong task count, got %v", taskCount)
	}

	token, _ := tests.GenerateTestJWT(1)

	rr := makeCardRequestSuccess(s, t, 1)
	var task models.Task
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)

	if task.IsComplete {
		t.Errorf("something is wrong, task already complete")
	}
	task.IsComplete = true
	task.Title = task.Title + " every 2 days"
	jsonData, _ := json.Marshal(task)

	req, err := http.NewRequest("PUT", "/api/tasks/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/tasks/{id}", s.JwtMiddleware(s.UpdateTaskRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var newTaskCount int
	_ = s.DB.QueryRow("SELECT count(*) FROM tasks").Scan(&newTaskCount)
	if taskCount+1 != newTaskCount {
		t.Errorf("wrong task count, got %v want %v", taskCount+1, newTaskCount)
	}
}

func TestGetTasksForCardSuccess(t *testing.T) {

	s := setup()
	defer tests.Teardown()

	rr := makeTasksRequestSuccess(s, t, "")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	tasks, _ := s.QueryTasksByCard(1, 1)
	if len(tasks) != 1 {
		t.Errorf("wrong number of tasks returned, got %v want %v", len(tasks), 1)
	}
}

func TestTaskTimestampsAreTZAware(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Create a task with a specific scheduled date
	scheduledDate := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)

	token, _ := tests.GenerateTestJWT(1)

	taskData := map[string]interface{}{
		"title":          "Test TZ Task",
		"scheduled_date": scheduledDate.Format(time.RFC3339),
		"user_id":        1,
	}
	jsonData, _ := json.Marshal(taskData)

	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.CreateTaskRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	taskID := int(response["id"].(float64))

	// Retrieve the task and verify scheduled_date is preserved with TZ
	rr = makeTaskRequestSuccess(s, t, taskID)
	var task models.Task
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &task)

	if task.ScheduledDate == nil {
		t.Fatal("scheduled_date should not be nil")
	}

	// Verify the time is correct (allowing for minor precision differences)
	if task.ScheduledDate.UTC().Unix() != scheduledDate.Unix() {
		t.Errorf("scheduled_date not preserved correctly, got %v want %v",
			task.ScheduledDate.UTC(), scheduledDate)
	}
}

func TestTaskTimestampsJSONFormat(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	rr := makeTaskRequestSuccess(s, t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse raw JSON to verify timestamp format
	var rawJSON map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &rawJSON)

	// created_at should be present and in RFC3339 format with timezone
	if createdAt, ok := rawJSON["created_at"].(string); ok {
		_, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			t.Errorf("created_at should be in RFC3339 format with timezone, got %v, error: %v",
				createdAt, err)
		}
		// Verify it includes timezone information (ends with Z or +/-offset)
		if len(createdAt) < 20 {
			t.Errorf("created_at appears to be missing timezone information: %v", createdAt)
		}
	} else {
		t.Error("created_at should be a string in the JSON response")
	}
}
