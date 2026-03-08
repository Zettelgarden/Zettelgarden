package handlers

import (
	"context"
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestHabitsHandlerCompile(t *testing.T) {
	// Compile check
	if true != false {
		t.Error("handler file should compile")
	}
}

// TestGetHabitLogsRoute tests the GET /api/habits/{id}/logs endpoint
// This test suite validates the handler before implementation (TDD approach)
func TestGetHabitLogsRoute(t *testing.T) {
	testCases := []struct {
		name           string
		habitID        string
		queryLimit     string
		queryOffset    string
		expectedStatus int
		setupFunc      func(*Handler) (int, error) // Returns habitID and error
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "returns empty logs array for habit with no check-ins",
			habitID:        "1",
			queryLimit:     "",
			queryOffset:    "",
			expectedStatus: http.StatusOK,
			setupFunc: func(s *Handler) (int, error) {
				// Create a habit with no logs
				params := models.CreateHabitParams{
					Title:     "Test Habit",
					Frequency: models.FrequencyDaily,
				}
				id, err := tests.CreateTestHabit(s.GetDB(), 1, params)
				return id, err
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response struct {
					Logs  []models.HabitLog `json:"logs"`
					Total int               `json:"total"`
				}
				tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

				if response.Total != 0 {
					t.Errorf("expected total 0, got %d", response.Total)
				}
				if len(response.Logs) != 0 {
					t.Errorf("expected 0 logs, got %d", len(response.Logs))
				}
			},
		},
		{
			name:           "returns habit logs with default pagination",
			habitID:        "1",
			queryLimit:     "",
			queryOffset:    "",
			expectedStatus: http.StatusOK,
			setupFunc: func(s *Handler) (int, error) {
				// Create a habit
				params := models.CreateHabitParams{
					Title:     "Test Habit",
					Frequency: models.FrequencyDaily,
				}
				habitID, err := tests.CreateTestHabit(s.GetDB(), 1, params)
				if err != nil {
					return 0, err
				}

				// Create a log entry
				notes := "Felt great today!"
				_, err = tests.CreateTestHabitLog(s.GetDB(), 1, habitID, &notes)
				return habitID, err
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response struct {
					Logs  []models.HabitLog `json:"logs"`
					Total int               `json:"total"`
				}
				tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

				if response.Total != 1 {
					t.Errorf("expected total 1, got %d", response.Total)
				}
				if len(response.Logs) != 1 {
					t.Errorf("expected 1 log, got %d", len(response.Logs))
				}

				log := response.Logs[0]
				if log.HabitID != 1 {
					t.Errorf("expected habit_id 1, got %d", log.HabitID)
				}
				if log.UserID != 1 {
					t.Errorf("expected user_id 1, got %d", log.UserID)
				}
				if log.Notes == nil || *log.Notes != "Felt great today!" {
					t.Errorf("expected notes 'Felt great today!', got %v", log.Notes)
				}
			},
		},
		{
			name:           "returns 404 for non-existent habit",
			habitID:        "999",
			queryLimit:     "",
			queryOffset:    "",
			expectedStatus: http.StatusNotFound,
			setupFunc: func(s *Handler) (int, error) {
				// No setup - habit doesn't exist
				return 999, nil
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Body.String() == "" {
					t.Error("expected error message in response body")
				}
			},
		},
		{
			name:           "returns 400 for invalid habit ID",
			habitID:        "invalid",
			queryLimit:     "",
			queryOffset:    "",
			expectedStatus: http.StatusBadRequest,
			setupFunc: func(s *Handler) (int, error) {
				// No setup needed for invalid ID test
				return 0, nil
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				if rr.Body.String() == "" {
					t.Error("expected error message in response body")
				}
			},
		},
		{
			name:           "respects limit query parameter",
			habitID:        "1",
			queryLimit:     "1",
			queryOffset:    "",
			expectedStatus: http.StatusOK,
			setupFunc: func(s *Handler) (int, error) {
				// Create a habit
				params := models.CreateHabitParams{
					Title:     "Test Habit",
					Frequency: models.FrequencyDaily,
				}
				habitID, err := tests.CreateTestHabit(s.GetDB(), 1, params)
				if err != nil {
					return 0, err
				}

				// Create multiple log entries
				baseTime := time.Date(2026, 3, 8, 14, 30, 0, 0, time.UTC)
				for i := 0; i < 3; i++ {
					offset := time.Duration(-i * 24)
					completedAt := baseTime.Add(offset * time.Hour)
					_, err = tests.CreateTestHabitLogWithTime(s.GetDB(), 1, habitID, nil, completedAt)
					if err != nil {
						return 0, err
					}
				}
				return habitID, nil
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response struct {
					Logs  []models.HabitLog `json:"logs"`
					Total int               `json:"total"`
				}
				tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

				if response.Total != 3 {
					t.Errorf("expected total 3, got %d", response.Total)
				}
				if len(response.Logs) != 1 {
					t.Errorf("expected 1 log due to limit, got %d", len(response.Logs))
				}
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHandler()
			defer tests.Teardown()

			// Setup test data
			habitID, err := tt.setupFunc(s)
			if err != nil {
				t.Fatalf("failed to setup test: %v", err)
			}

			// Override habitID if test uses dynamic ID
			testHabitID := tt.habitID
			if tt.habitID == "1" && habitID != 1 {
				testHabitID = strconv.Itoa(habitID)
			}

			// Create request
			url := "/api/habits/" + testHabitID + "/logs"
			if tt.queryLimit != "" {
				url += "?limit=" + tt.queryLimit
			}
			if tt.queryOffset != "" {
				if tt.queryLimit == "" {
					url += "?offset=" + tt.queryOffset
				} else {
					url += "&offset=" + tt.queryOffset
				}
			}

			token, _ := tests.GenerateTestJWT(1)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req = mux.SetURLVars(req, map[string]string{"id": testHabitID})

			// Create response recorder and execute request
			rr := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/api/habits/{id}/logs", s.JwtMiddleware(s.GetHabitLogsRoute))
			router.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
					status, tt.expectedStatus, rr.Body.String())
			}

			// Run validation function if provided
			if tt.validateFunc != nil && tt.expectedStatus == http.StatusOK {
				tt.validateFunc(t, rr)
			}
		})
	}
}

// TestGetHabitLogsRouteUnauthorized tests that unauthorized requests are rejected
func TestGetHabitLogsRouteUnauthorized(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	req, err := http.NewRequest("GET", "/api/habits/1/logs", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/habits/{id}/logs", s.JwtMiddleware(s.GetHabitLogsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

// TestGetHabitLogsRouteWrongUser tests that users can't access other users' habit logs
func TestGetHabitLogsRouteWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create a habit for user 1
	params := models.CreateHabitParams{
		Title:     "User 1 Habit",
		Frequency: models.FrequencyDaily,
	}
	habitID, err := tests.CreateTestHabit(s.GetDB(), 1, params)
	if err != nil {
		t.Fatalf("failed to create habit: %v", err)
	}

	// Create a log entry
	notes := "User 1's log"
	_, err = tests.CreateTestHabitLog(s.GetDB(), 1, habitID, &notes)
	if err != nil {
		t.Fatalf("failed to create habit log: %v", err)
	}

	// Try to access as user 2
	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/habits/"+strconv.Itoa(habitID)+"/logs", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(habitID)})

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/habits/{id}/logs", s.JwtMiddleware(s.GetHabitLogsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

// TestGetHabitStatsRoute tests the GET /api/habits/{id}/stats endpoint
func TestGetHabitStatsRoute(t *testing.T) {
	testCases := []struct {
		name           string
		habitID        string
		expectedStatus int
		setupFunc      func(*Handler) (int, error) // Returns habitID and error
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "returns stats for habit with no check-ins",
			habitID:        "1",
			expectedStatus: http.StatusOK,
			setupFunc: func(s *Handler) (int, error) {
				params := models.CreateHabitParams{
					Title:     "Test Habit",
					Frequency: models.FrequencyDaily,
				}
				id, err := tests.CreateTestHabit(s.GetDB(), 1, params)
				return id, err
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var stats models.HabitStats
				tests.ParseJsonResponse(t, rr.Body.Bytes(), &stats)

				if stats.TotalCompletions != 0 {
					t.Errorf("expected total completions 0, got %d", stats.TotalCompletions)
				}
				if stats.CurrentStreak != 0 {
					t.Errorf("expected current streak 0, got %d", stats.CurrentStreak)
				}
				if stats.LongestStreak != 0 {
					t.Errorf("expected longest streak 0, got %d", stats.LongestStreak)
				}
			},
		},
		{
			name:           "returns stats with completions",
			habitID:        "1",
			expectedStatus: http.StatusOK,
			setupFunc: func(s *Handler) (int, error) {
				params := models.CreateHabitParams{
					Title:     "Test Habit",
					Frequency: models.FrequencyDaily,
				}
				habitID, err := tests.CreateTestHabit(s.GetDB(), 1, params)
				if err != nil {
					return 0, err
				}

				// Create multiple log entries (3 consecutive days)
				baseTime := time.Date(2026, 3, 8, 14, 30, 0, 0, time.UTC)
				for i := 0; i < 3; i++ {
					offset := time.Duration(-i * 24)
					completedAt := baseTime.Add(offset * time.Hour)
					_, err = tests.CreateTestHabitLogWithTime(s.GetDB(), 1, habitID, nil, completedAt)
					if err != nil {
						return 0, err
					}
				}
				return habitID, nil
			},
			validateFunc: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var stats models.HabitStats
				tests.ParseJsonResponse(t, rr.Body.Bytes(), &stats)

				if stats.TotalCompletions != 3 {
					t.Errorf("expected total completions 3, got %d", stats.TotalCompletions)
				}
				if stats.CurrentStreak != 3 {
					t.Errorf("expected current streak 3, got %d", stats.CurrentStreak)
				}
				if stats.LongestStreak != 3 {
					t.Errorf("expected longest streak 3, got %d", stats.LongestStreak)
				}
				if stats.LastCompletedAt == nil {
					t.Error("expected last_completed_at to be set")
				}
			},
		},
		{
			name:           "returns 404 for non-existent habit",
			habitID:        "999",
			expectedStatus: http.StatusNotFound,
			setupFunc: func(s *Handler) (int, error) {
				return 999, nil
			},
			validateFunc: nil,
		},
		{
			name:           "returns 400 for invalid habit ID",
			habitID:        "invalid",
			expectedStatus: http.StatusBadRequest,
			setupFunc: func(s *Handler) (int, error) {
				return 0, nil
			},
			validateFunc: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHandler()
			defer tests.Teardown()

			habitID, err := tt.setupFunc(s)
			if err != nil {
				t.Fatalf("failed to setup test: %v", err)
			}

			testHabitID := tt.habitID
			if tt.habitID == "1" && habitID != 1 {
				testHabitID = strconv.Itoa(habitID)
			}

			token, _ := tests.GenerateTestJWT(1)
			req, err := http.NewRequest("GET", "/api/habits/"+testHabitID+"/stats", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req = mux.SetURLVars(req, map[string]string{"id": testHabitID})

			rr := httptest.NewRecorder()
			router := mux.NewRouter()
			router.HandleFunc("/api/habits/{id}/stats", s.JwtMiddleware(s.GetHabitStatsRoute))
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
					status, tt.expectedStatus, rr.Body.String())
			}

			if tt.validateFunc != nil && tt.expectedStatus == http.StatusOK {
				tt.validateFunc(t, rr)
			}
		})
	}
}

// TestGetHabitStatsRouteUnauthorized tests that unauthorized requests are rejected
func TestGetHabitStatsRouteUnauthorized(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	req, err := http.NewRequest("GET", "/api/habits/1/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/habits/{id}/stats", s.JwtMiddleware(s.GetHabitStatsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

// TestGetHabitStatsRouteWrongUser tests that users can't access other users' habit stats
func TestGetHabitStatsRouteWrongUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	params := models.CreateHabitParams{
		Title:     "User 1 Habit",
		Frequency: models.FrequencyDaily,
	}
	habitID, err := tests.CreateTestHabit(s.GetDB(), 1, params)
	if err != nil {
		t.Fatalf("failed to create habit: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/habits/"+strconv.Itoa(habitID)+"/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(habitID)})

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/habits/{id}/stats", s.JwtMiddleware(s.GetHabitStatsRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}
