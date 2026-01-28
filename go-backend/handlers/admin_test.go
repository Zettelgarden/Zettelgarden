package handlers

import (
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestAdminMiddleware_AllowsAdminAccess verifies that admin users can access admin routes
func TestAdminMiddleware_AllowsAdminAccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	// Create a simple test handler that returns 200 if called
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin access granted"))
	}

	// Apply admin middleware
	handler := s.JwtMiddleware(s.AdminMiddleware(testHandler))
	handler.ServeHTTP(rr, req)

	// Check that the request was allowed
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestAdminMiddleware_BlocksNonAdminAccess verifies that non-admin users cannot access admin routes
func TestAdminMiddleware_BlocksNonAdminAccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Ensure user 2 is NOT an admin (default)
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = false WHERE id = 2`)
	if err != nil {
		t.Fatalf("Failed to set user as non-admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	// Create a simple test handler that should never be called
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here"))
	}

	// Apply admin middleware
	handler := s.JwtMiddleware(s.AdminMiddleware(testHandler))
	handler.ServeHTTP(rr, req)

	// Check that the request was blocked
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
}

// TestAdminMiddleware_BlocksUnauthenticatedRequests verifies that unauthenticated requests are blocked
func TestAdminMiddleware_BlocksUnauthenticatedRequests(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	req, err := http.NewRequest("GET", "/api/admin/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	// No authorization header

	rr := httptest.NewRecorder()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here"))
	}

	// Apply admin middleware
	handler := s.AdminMiddleware(testHandler)
	handler.ServeHTTP(rr, req)

	// Check that the request was blocked
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

// TestAdminOrSelfMiddleware_AllowsAdminAnyUser verifies admins can access any user's resources
func TestAdminOrSelfMiddleware_AllowsAdminAnyUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Make user 1 an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = true WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/users/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	// Create a simple test handler
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin accessing user 2"))
	}

	// Apply admin-or-self middleware with "id" param
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.AdminOrSelfMiddleware("id")(testHandler)))
	router.ServeHTTP(rr, req)

	// Check that the request was allowed
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestAdminOrSelfMiddleware_AllowsUserOwnResources verifies users can access their own resources
func TestAdminOrSelfMiddleware_AllowsUserOwnResources(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Ensure user 2 is NOT an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = false WHERE id = 2`)
	if err != nil {
		t.Fatalf("Failed to set user as non-admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/users/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user accessing own resource"))
	}

	// Apply admin-or-self middleware with "id" param
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.AdminOrSelfMiddleware("id")(testHandler)))
	router.ServeHTTP(rr, req)

	// Check that the request was allowed
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestAdminOrSelfMiddleware_BlocksUserAccessingOtherUser verifies non-admins cannot access other users' resources
func TestAdminOrSelfMiddleware_BlocksUserAccessingOtherUser(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Ensure user 2 is NOT an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = false WHERE id = 2`)
	if err != nil {
		t.Fatalf("Failed to set user as non-admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/users/3", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here"))
	}

	// Apply admin-or-self middleware with "id" param
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.AdminOrSelfMiddleware("id")(testHandler)))
	router.ServeHTTP(rr, req)

	// Check that the request was blocked
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
}
