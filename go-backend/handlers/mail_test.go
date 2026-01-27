package handlers

import (
	"go-backend/models"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMailingListSubscribersSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Add a test subscriber first
	_, err := s.Server.Tx.Exec(`
		INSERT INTO mailing_list (email, welcome_email_sent)
		VALUES ($1, $2)
	`, "test@example.com", true)
	if err != nil {
		t.Fatal(err)
	}

	token, _ := tests.GenerateTestJWT(1) // Admin user
	req, err := http.NewRequest("GET", "/api/mailing-list", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetMailingListSubscribersRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var subscribers []models.MailingList
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &subscribers)
	if len(subscribers) != 1 {
		t.Errorf("handler returned wrong number of subscribers: got %v want %v", len(subscribers), 1)
	}
	if subscribers[0].Email != "test@example.com" {
		t.Errorf("handler returned wrong email: got %v want %v", subscribers[0].Email, "test@example.com")
	}
}

func TestGetMailingListSubscribersUnauthorized(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Ensure user 2 is NOT an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = false WHERE id = 2`)
	if err != nil {
		t.Fatalf("Failed to set user as non-admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2) // Non-admin user
	req, err := http.NewRequest("GET", "/api/mailing-list", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	// Apply the full middleware chain as in addAdminRoute: AdminMiddleware -> JwtMiddleware
	testHandler := s.AdminMiddleware(s.GetMailingListSubscribersRoute)
	jwtHandler := s.JwtMiddleware(testHandler)
	handler := http.HandlerFunc(jwtHandler)
	handler.ServeHTTP(rr, req)

	// AdminMiddleware returns 403 Forbidden for non-admin users, not 401 Unauthorized
	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}
}

func TestUnsubscribeMailingListSuccess(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Add a test subscriber first
	_, err := s.Server.Tx.Exec(`
		INSERT INTO mailing_list (email, welcome_email_sent, subscribed)
		VALUES ($1, $2, $3)
	`, "test@example.com", true, true)
	if err != nil {
		t.Fatal(err)
	}

	token, _ := tests.GenerateTestJWT(1) // Admin user
	body := `{"email": "test@example.com"}`
	req, err := http.NewRequest("POST", "/api/mailing-list/unsubscribe", tests.StringToReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UnsubscribeMailingListRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify the subscriber was actually unsubscribed
	var subscribed bool
	err = s.Server.Tx.QueryRow("SELECT subscribed FROM mailing_list WHERE email = $1", "test@example.com").Scan(&subscribed)
	if err != nil {
		t.Fatal(err)
	}
	if subscribed {
		t.Error("subscriber was not unsubscribed")
	}
}

func TestUnsubscribeMailingListUnauthorized(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	// Ensure user 2 is NOT an admin
	_, err := s.Server.Tx.Exec(`UPDATE users SET is_admin = false WHERE id = 2`)
	if err != nil {
		t.Fatalf("Failed to set user as non-admin: %v", err)
	}

	token, _ := tests.GenerateTestJWT(2) // Non-admin user
	body := `{"email": "test@example.com"}`
	req, err := http.NewRequest("POST", "/api/mailing-list/unsubscribe", tests.StringToReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	// Apply the full middleware chain as in addAdminRoute: AdminMiddleware -> JwtMiddleware
	testHandler := s.AdminMiddleware(s.UnsubscribeMailingListRoute)
	jwtHandler := s.JwtMiddleware(testHandler)
	handler := http.HandlerFunc(jwtHandler)
	handler.ServeHTTP(rr, req)

	// AdminMiddleware returns 403 Forbidden for non-admin users, not 401 Unauthorized
	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}
}

func TestUnsubscribeMailingListInvalidEmail(t *testing.T) {
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1) // Admin user
	body := `{"email": "nonexistent@example.com"}`
	req, err := http.NewRequest("POST", "/api/mailing-list/unsubscribe", tests.StringToReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.UnsubscribeMailingListRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}
}
