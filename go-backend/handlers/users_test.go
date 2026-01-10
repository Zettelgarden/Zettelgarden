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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	_ = setup()
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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	s := setup()
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
	var users []models.User
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &users)
	if len(users) != 10 {
		t.Errorf("handler returned wrong number of users, got %v want %v", len(users), 10)
	}
}

// func TestUpdateUserRouteSuccess(t *testing.T) {
// 	s := setup()
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
	s := setup()
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
	if response.NewID != 11 {
		t.Errorf("handler returned unexpected result, got %v want %v", response.NewID, 11)
	}
}
func TestCreateUserDashboardCardsSuccess(t *testing.T) {
	s := setup()
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
	if response.NewID != 11 {
		t.Errorf("handler returned unexpected result, got %v want %v", response.NewID, 11)
	}

	var cardPK int
	err := s.DB.QueryRow("SELECT dashboard_card_pk FROM users where id = $1",
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
	err = s.DB.QueryRow("SELECT title, body FROM cards where id = $1", cardPK).Scan(&title, &body)
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
	s := setup()
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
	s := setup()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
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
	s := setup()
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
	s := setup()
	defer tests.Teardown()

	token, _ := s.generateTempToken(1)

	_, err := s.DB.Exec(`UPDATE users SET email_validated = FALSE WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	user, err := s.QueryUser(1)
	if user.EmailValidated {
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

	user, err = s.QueryUser(1)
	if !user.EmailValidated {
		t.Fatal("something has gone wrong: email is not validated when it should be")
	}

}

func TestUserTimezoneDefaultsToUTC(t *testing.T) {
	s := setup()
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
	_ = setup()
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
	s := setup()
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

func TestGetCurrentUserIncludesTimezone(t *testing.T) {
	s := setup()
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
