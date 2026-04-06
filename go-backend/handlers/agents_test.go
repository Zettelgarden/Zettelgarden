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

func TestCreateAgentHandler(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	reqBody := models.CreateAgentRequest{
		Name: "Test Agent",
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "/api/agents", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/agents", s.JwtMiddleware(s.CreateAgentHandler)).Methods("POST")
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response models.CreateAgentResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	if response.Name != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", response.Name)
	}

	if response.APIKey == "" {
		t.Error("API key should be returned")
	}

	if len(response.APIKey) < 20 {
		t.Errorf("API key seems too short: %s", response.APIKey)
	}
}

func TestListAgentsHandler(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// First create an agent
	token, _ := tests.GenerateTestJWT(1)
	reqBody := models.CreateAgentRequest{Name: "Agent 1"}
	body, _ := json.Marshal(reqBody)

	req1, _ := http.NewRequest("POST", "/api/agents", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)

	rr1 := httptest.NewRecorder()
	router1 := mux.NewRouter()
	router1.HandleFunc("/api/agents", s.JwtMiddleware(s.CreateAgentHandler)).Methods("POST")
	router1.ServeHTTP(rr1, req1)

	// Now list agents
	req2, _ := http.NewRequest("GET", "/api/agents", nil)
	req2.Header.Set("Authorization", "Bearer "+token)

	rr2 := httptest.NewRecorder()
	router2 := mux.NewRouter()
	router2.HandleFunc("/api/agents", s.JwtMiddleware(s.ListAgentsHandler)).Methods("GET")
	router2.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr2.Code, rr2.Body.String())
	}

	var response map[string][]models.Agent
	tests.ParseJsonResponse(t, rr2.Body.Bytes(), &response)

	agents := response["agents"]
	if len(agents) == 0 {
		t.Error("Expected at least one agent")
	}

	if agents[0].Name != "Agent 1" {
		t.Errorf("Expected agent name 'Agent 1', got '%s'", agents[0].Name)
	}
}

func TestRevokeAgentHandler(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create agent first
	token, _ := tests.GenerateTestJWT(1)
	reqBody := models.CreateAgentRequest{Name: "Agent to Revoke"}
	body, _ := json.Marshal(reqBody)

	req1, _ := http.NewRequest("POST", "/api/agents", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)

	rr1 := httptest.NewRecorder()
	router1 := mux.NewRouter()
	router1.HandleFunc("/api/agents", s.JwtMiddleware(s.CreateAgentHandler)).Methods("POST")
	router1.ServeHTTP(rr1, req1)

	var createResp models.CreateAgentResponse
	tests.ParseJsonResponse(t, rr1.Body.Bytes(), &createResp)
	agentID := createResp.ID

	// Revoke the agent
	req2, _ := http.NewRequest("DELETE", "/api/agents/"+strconv.Itoa(agentID), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.SetPathValue("id", strconv.Itoa(agentID))

	rr2 := httptest.NewRecorder()
	router2 := mux.NewRouter()
	router2.HandleFunc("/api/agents/{id}", s.JwtMiddleware(s.RevokeAgentHandler)).Methods("DELETE")
	router2.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rr2.Code, rr2.Body.String())
	}
}

func TestGetAgentActivityHandler(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Create agent first
	token, _ := tests.GenerateTestJWT(1)
	reqBody := models.CreateAgentRequest{Name: "Agent with Activity"}
	body, _ := json.Marshal(reqBody)

	req1, _ := http.NewRequest("POST", "/api/agents", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)

	rr1 := httptest.NewRecorder()
	router1 := mux.NewRouter()
	router1.HandleFunc("/api/agents", s.JwtMiddleware(s.CreateAgentHandler)).Methods("POST")
	router1.ServeHTTP(rr1, req1)

	var createResp models.CreateAgentResponse
	tests.ParseJsonResponse(t, rr1.Body.Bytes(), &createResp)
	agentID := createResp.ID

	// Get activity
	req2, _ := http.NewRequest("GET", "/api/agents/"+strconv.Itoa(agentID)+"/activity", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.SetPathValue("id", strconv.Itoa(agentID))

	rr2 := httptest.NewRecorder()
	router2 := mux.NewRouter()
	router2.HandleFunc("/api/agents/{id}/activity", s.JwtMiddleware(s.GetAgentActivityHandler)).Methods("GET")
	router2.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr2.Code, rr2.Body.String())
	}

	var response map[string]interface{}
	tests.ParseJsonResponse(t, rr2.Body.Bytes(), &response)

	if _, ok := response["logs"]; !ok {
		t.Error("Response should contain 'logs' field")
	}

	if _, ok := response["pagination"]; !ok {
		t.Error("Response should contain 'pagination' field")
	}
}
