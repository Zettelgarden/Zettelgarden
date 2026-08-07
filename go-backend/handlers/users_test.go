package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"go-backend/tests"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

func makeUserRequestSuccess(t *testing.T, id int) *httptest.ResponseRecorder {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/users/"+strconv.Itoa(id), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", strconv.Itoa(id))

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.GetUserRoute))
	router.ServeHTTP(rr, req)

	return rr
}

func TestUserGetAdmin(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetUserAdminRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}
func TestUserGetAdminFailure(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)

	req, err := http.NewRequest("GET", "/api/admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetUserAdminRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}
}

func TestGetUserSuccess(t *testing.T) {
	_ = NewHandler()
	defer tests.Teardown()

	rr := makeUserRequestSuccess(t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var user models.User
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &user)
	if user.ID != 1 {
		t.Errorf("handler returned wrong user id, got %v want %v", user.ID, 1)
	}

}
func TestGetUserUnauthorized(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/users/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.Admin(s.GetUserRoute)))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}

}
func TestGetUserBadInput(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/users/-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "-1")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.Admin(s.GetUserRoute)))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestGetCurrentUserSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(3)
	req, err := http.NewRequest("GET", "/api/current", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetCurrentUserRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var user models.User
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &user)
	if user.ID != 3 {
		t.Errorf("handler returned wrong user id, got %v want %v", user.ID, 3)
	}
}

func TestGetUserSubscriptionSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/users/1/subscription", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}/subscription", s.JwtMiddleware(s.GetUserSubscriptionRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var userSub models.UserSubscription
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &userSub)
	if userSub.ID != 1 {
		t.Errorf("handler returned wrong user id, got %v want %v", userSub.ID, 1)
	}
}
func TestGetUserSubscriptionUnauthorized(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(3)
	req, err := http.NewRequest("GET", "/api/users/1/subscription", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.Admin(s.GetUserSubscriptionRoute)))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestGetUsersRouteSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetUsersRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)
	users := response["users"].([]interface{})
	if len(users) != 3 {
		t.Errorf("handler returned wrong number of users, got %v want %v", len(users), 3)
	}
}

// func TestUpdateUserRouteSuccess(t *testing.T) {
// 	s := NewHandler()
// 	defer tests.Teardown()

// 	expected := "asdfasdf"

// 	rr := makeUserRequestSuccess(t, 1)
// 	var user models.User
// 	tests.ParseJsonResponse(t, rr.Body.Bytes(), &user)

// 	log.Printf("useraoaoe %v", user)
// 	token, _ := tests.GenerateTestJWT(1)
// 	newData := map[string]interface{}{
// 		"username": expected,
// 		"is_admin": true,
// 		"email":    expected,
// 	}
// 	jsonData, err := json.Marshal(newData)
// 	if err != nil {
// 		log.Fatalf("Error marshalling JSON: %v", err)
// 	}
// 	req, err := http.NewRequest("PUT", "/api/users/1", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.SetPathValue("id", "1")

// 	rr = httptest.NewRecorder()
// 	router := mux.NewRouter()
// 	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.UpdateUserRoute))
// 	router.ServeHTTP(rr, req)

// 	rr = makeUserRequestSuccess(t, 1)
// 	tests.ParseJsonResponse(t, rr.Body.Bytes(), &user)
// 	if status := rr.Code; status != http.StatusOK {
// 		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
// 	}

// 	if user.Username != expected {
// 		log.Printf("body %v", rr.Body)
// 		t.Errorf("handler returned wrong username, got %v want %v", user.Username, expected)
// 	}
// 	if user.EmailValidated {
// 		t.Errorf("handler returned wrong email validation, got %v want %v", user.EmailValidated, false)

// 	}

// }

func createUserWithParams(s *Handler, t *testing.T, params models.CreateUserParams) *httptest.ResponseRecorder {
	jsonData, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/users/", bytes.NewBuffer(jsonData))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.CreateUserRoute)
	handler.ServeHTTP(rr, req)

	return rr
}

func TestCreateUserSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	params := models.CreateUserParams{
		Username:        "asdfadf",
		Password:        "asdfasdfasdf",
		ConfirmPassword: "asdfasdfasdf",
		Email:           "asdf@asdf.com",
	}
	rr := createUserWithParams(s, t, params)

	var response models.CreateUserResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if response.NewID <= 0 {
		t.Errorf("handler returned unexpected result, got %v want ID > 0", response.NewID)
	}
}
func TestCreateUserDashboardCardsSuccess(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	params := models.CreateUserParams{
		Username:        "asdfadf",
		Password:        "asdfasdfasdf",
		ConfirmPassword: "asdfasdfasdf",
		Email:           "asdf@asdf.com",
	}
	rr := createUserWithParams(s, t, params)

	var response models.CreateUserResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if response.NewID <= 0 {
		t.Errorf("handler returned unexpected result, got %v want ID > 0", response.NewID)
	}

	var cardPK int
	err := s.Server.Tx.QueryRow("SELECT dashboard_card_pk FROM users where id = $1",
		response.NewID).Scan(&cardPK)
	if err != nil {
		t.Errorf("handler returned error %v", err)
	}
	if cardPK == 0 {
		t.Errorf("dashboard card not set, expected an id other than 0")
	}

	expectedTitle := "Welcome to Zettelgarden!"
	var title string
	var body string
	err = s.Server.Tx.QueryRow("SELECT title, body FROM cards where id = $1", cardPK).Scan(&title, &body)
	if err != nil {
		t.Errorf("handler returned error %v", err)
	}
	if title != expectedTitle {
		t.Errorf("incorrect card title returned, got %v want %v", title, expectedTitle)
	}
	if body == "" {
		t.Errorf("incorrect card body returned, want non-blank body")
	}
}

func TestCreateUserMismatchedPass(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	params := models.CreateUserParams{
		Username:        "asdfadf",
		Password:        "asdfasdfasdf",
		ConfirmPassword: "a",
		Email:           "asdf@asdf.com",
	}
	rr := createUserWithParams(s, t, params)

	var response models.CreateUserResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
	if response.NewID != 0 {
		t.Errorf("handler returned unexpected result, got %v want %v", response.NewID, 0)
	}

}

func TestResendValidateEmailAlreadyValidated(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(2)
	req, err := http.NewRequest("GET", "/api/email-validate", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ResendEmailValidationRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestResendValidateEmailNotValidated(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	expected := "asdfasdf"
	newData := map[string]interface{}{
		"username": expected,
		"is_admin": true,
		"email":    expected,
	}
	jsonData, err := json.Marshal(newData)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}
	req, err := http.NewRequest("PUT", "/api/users/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.UpdateUserRoute))
	router.ServeHTTP(rr, req)

	req, err = http.NewRequest("GET", "/api/email-validate", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ResendEmailValidationRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestValidateEmail(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := s.generateTempToken(1)

	// Verify initial state - user 1's email should not be validated (from test data)
	var emailValidated bool
	err := s.Server.Tx.QueryRow(`SELECT email_validated FROM users WHERE id = 1`).Scan(&emailValidated)
	if err != nil {
		t.Fatal(err)
	}
	if emailValidated {
		t.Fatal("something has gone wrong: email is validated when it shouldn't be")
	}

	data := map[string]interface{}{
		"token": token,
	}
	jsonData, err := json.Marshal(data)
	req, err := http.NewRequest("POST", "/api/email-validate", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ValidateEmailRoute))
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify email is now validated - use transaction to see changes made by handler
	err = s.Server.Tx.QueryRow(`SELECT email_validated FROM users WHERE id = 1`).Scan(&emailValidated)
	if err != nil {
		t.Fatal(err)
	}
	if !emailValidated {
		t.Fatal("something has gone wrong: email is not validated when it should be")
	}

}

func TestUserTimezoneDefaultsToUTC(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Get existing user
	user, err := s.QueryUser(1)
	if err != nil {
		t.Fatal(err)
	}

	// Verify timezone defaults to UTC
	if user.Timezone != "UTC" {
		t.Errorf("user timezone should default to UTC, got %v", user.Timezone)
	}
}

func TestUserTimezoneInAPIResponse(t *testing.T) {
	_ = NewHandler()
	defer tests.Teardown()

	rr := makeUserRequestSuccess(t, 1)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var user models.User
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &user)

	// Verify timezone is in response
	if user.Timezone == "" {
		t.Error("user timezone should be included in API response")
	}
	if user.Timezone != "UTC" {
		t.Errorf("user timezone should be UTC, got %v", user.Timezone)
	}
}

func TestUpdateUserTimezone(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	expectedTimezone := "America/New_York"

	newData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"is_admin": true,
		"timezone": expectedTimezone,
	}
	jsonData, err := json.Marshal(newData)
	if err != nil {
		t.Fatalf("Error marshalling JSON: %v", err)
	}

	req, err := http.NewRequest("PUT", "/api/users/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.UpdateUserRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify timezone was updated
	user, err := s.QueryUser(1)
	if err != nil {
		t.Fatal(err)
	}

	if user.Timezone != expectedTimezone {
		t.Errorf("user timezone not updated correctly, got %v want %v", user.Timezone, expectedTimezone)
	}
}

func TestUpdateUserShowTasksAndRss(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	newData := map[string]interface{}{
		"username":   "testuser",
		"email":      "test@example.com",
		"is_admin":   true,
		"show_tasks": false,
		"show_rss":   false,
	}
	jsonData, err := json.Marshal(newData)
	if err != nil {
		t.Fatalf("Error marshalling JSON: %v", err)
	}

	req, err := http.NewRequest("PUT", "/api/users/1", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/users/{id}", s.JwtMiddleware(s.UpdateUserRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		log.Printf("err %v", rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	user, err := s.QueryUser(1)
	if err != nil {
		t.Fatal(err)
	}

	if user.ShowTasks {
		t.Errorf("show_tasks should be false after update, got %v", user.ShowTasks)
	}
	if user.ShowRss {
		t.Errorf("show_rss should be false after update, got %v", user.ShowRss)
	}

	// Verify defaults are true for a fresh user
	user2, err := s.QueryUser(2)
	if err != nil {
		t.Fatal(err)
	}
	if !user2.ShowTasks {
		t.Error("show_tasks should default to true")
	}
	if !user2.ShowRss {
		t.Error("show_rss should default to true")
	}
}

func TestGetCurrentUserIncludesTimezone(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/current", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.GetCurrentUserRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var user models.User
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &user)

	if user.Timezone == "" {
		t.Error("current user endpoint should include timezone")
	}
	if user.Timezone != "UTC" {
		t.Errorf("user timezone should default to UTC, got %v", user.Timezone)
	}
}
