package routes

import (
	"go-backend/handlers"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestAddAdminRoute_AuthenticationBeforeAuthorization verifies that addAdminRoute
// wraps middleware in the correct order: authentication BEFORE authorization.
//
// This test would have caught the bug where AdminMiddleware ran before
// APIKeyOrJWTMiddleware, causing "current_user not found in context" errors.
func TestAddAdminRoute_AuthenticationBeforeAuthorization(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	h := &handlers.Handler{Server: s, DB: s.DB}

	// Make user 1 an admin
	_, err := s.DB.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	router := mux.NewRouter()

	// Register a route using the addAdminRoute helper
	// This is how actual admin routes are registered in the codebase
	testCalled := false
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		testCalled = true
		// Verify that current_user is set in context (this would fail if middleware order is wrong)
		userID, ok := r.Context().Value("current_user").(int)
		if !ok {
			t.Error("current_user not found in context - middleware may be in wrong order")
			return
		}
		if userID != 1 {
			t.Errorf("Expected userID 1, got %d", userID)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin success"))
	}

	addAdminRoute(router, h, "/api/test/admin", testHandler, "GET")

	// Test 1: Valid admin token should work
	token, _ := tests.GenerateTestJWT(1)
	req, _ := http.NewRequest("GET", "/api/test/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Valid admin token: expected status 200, got %d", rr.Code)
	}
	if !testCalled {
		t.Error("Handler was not called - request was blocked prematurely")
	}

	// Test 2: Non-admin user should be blocked (not unauthorized)
	// If middleware order is wrong, this would return 401 (unauthorized) instead of 403 (forbidden)
	testCalled = false
	token2, _ := tests.GenerateTestJWT(2)
	req2, _ := http.NewRequest("GET", "/api/test/admin", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)

	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	// Should get 403 (forbidden) not 401 (unauthorized)
	// 401 means authentication failed, 403 means authentication succeeded but authorization failed
	if rr2.Code != http.StatusForbidden {
		t.Errorf("Non-admin user: expected status 403, got %d (if 401, middleware order is wrong)", rr2.Code)
	}
	if testCalled {
		t.Error("Handler should not have been called for non-admin user")
	}

	// Test 3: No token should return 401 (unauthorized)
	testCalled = false
	req3, _ := http.NewRequest("GET", "/api/test/admin", nil)

	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusUnauthorized {
		t.Errorf("No token: expected status 401, got %d", rr3.Code)
	}
}

// TestAddAdminOrSelfRoute_AuthenticationBeforeAuthorization verifies that addAdminOrSelfRoute
// wraps middleware in the correct order.
func TestAddAdminOrSelfRoute_AuthenticationBeforeAuthorization(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	h := &handlers.Handler{Server: s, DB: s.DB}

	// Make user 1 an admin
	_, err := s.DB.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	router := mux.NewRouter()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value("current_user").(int)
		if !ok {
			t.Error("current_user not found in context - middleware may be in wrong order")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	// Register route with id parameter
	addAdminOrSelfRoute(router, h, "/api/test/users/{id}", testHandler, "GET", "id")

	// Test 1: Admin can access any user's resource
	token, _ := tests.GenerateTestJWT(1)
	req, _ := http.NewRequest("GET", "/api/test/users/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Admin accessing other user: expected status 200, got %d", rr.Code)
	}

	// Test 2: User can access their own resource
	token2, _ := tests.GenerateTestJWT(2)
	req2, _ := http.NewRequest("GET", "/api/test/users/2", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)

	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("User accessing own resource: expected status 200, got %d", rr2.Code)
	}

	// Test 3: User cannot access another user's resource
	req3, _ := http.NewRequest("GET", "/api/test/users/1", nil)
	req3.Header.Set("Authorization", "Bearer "+token2)

	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusForbidden {
		t.Errorf("User accessing other user: expected status 403, got %d", rr3.Code)
	}
}

// TestAddProtectedRoute_AuthenticationWorks verifies that addProtectedRoute
// correctly requires authentication.
func TestAddProtectedRoute_AuthenticationWorks(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	h := &handlers.Handler{Server: s, DB: s.DB}

	router := mux.NewRouter()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value("current_user").(int)
		if !ok {
			t.Error("current_user not found in context")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("protected"))
	}

	addProtectedRoute(router, h, "/api/test/protected", testHandler, "GET")

	// Test with valid token
	token, _ := tests.GenerateTestJWT(1)
	req, _ := http.NewRequest("GET", "/api/test/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("With token: expected status 200, got %d", rr.Code)
	}

	// Test without token
	req2, _ := http.NewRequest("GET", "/api/test/protected", nil)

	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("Without token: expected status 401, got %d", rr2.Code)
	}
}
