package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-backend/models"
	"go-backend/services"
	"go-backend/tests"

	"github.com/gorilla/mux"
)

// TestGetGraph verifies the knowledge graph endpoint returns owned nodes and
// edges with correct type filtering.
func TestGetGraph(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/graph", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/graph", s.JwtMiddleware(s.GetGraphRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var data models.GraphData
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &data)

	// Fixture summary for user 1: cards 1-12 + 14 (13 is user 2) = 13 cards,
	// entities 1,2,4 (3 is user 2) = 3, tags 1-3 = 3. Edges: backlinks 12->1
	// and 3->4 (self-link 1->1 excluded); parents 1->11, 2->12, 12->14;
	// entity junctions for cards 1,2 (entities 1,2), cards 1,3 (entity 4);
	// card 2 -> tag 1.
	if len(data.Nodes) != 19 {
		t.Errorf("expected 19 nodes (13 cards + 3 entities + 3 tags), got %d", len(data.Nodes))
	}
	if len(data.Edges) != 12 {
		t.Errorf("expected 12 edges (2 ref + 3 parent + 6 entity + 1 tag), got %d", len(data.Edges))
	}

	got := make(map[string]models.GraphNode)
	for _, n := range data.Nodes {
		got[n.ID] = n
	}

	if n, ok := got["card:1"]; !ok || n.CardID != "1" {
		t.Errorf("expected card:1 node with card_id '1', got %+v", n)
	}
	if n, ok := got["entity:1"]; !ok || n.Label != "Test Entity 1" {
		t.Errorf("expected entity:1 node 'Test Entity 1', got %+v", n)
	}
	if n, ok := got["tag:1"]; !ok || n.Label != "test" {
		t.Errorf("expected tag:1 node 'test', got %+v", n)
	}
	if _, ok := got["card:13"]; ok {
		t.Error("card:13 belongs to user 2 and must not appear")
	}
	if _, ok := got["entity:3"]; ok {
		t.Error("entity:3 belongs to user 2 and must not appear")
	}

	edgeTypes := make(map[string]int)
	for _, e := range data.Edges {
		edgeTypes[e.Type]++
	}
	if edgeTypes["reference"] != 2 || edgeTypes["parent"] != 3 ||
		edgeTypes["entity"] != 6 || edgeTypes["tag"] != 1 {
		t.Errorf("unexpected edge type counts: %v", edgeTypes)
	}
}

// TestGetGraph_TypesFilter verifies the ?types= filter narrows nodes and edges.
func TestGetGraph_TypesFilter(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)
	req, err := http.NewRequest("GET", "/api/graph?types=card", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/graph", s.JwtMiddleware(s.GetGraphRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var data models.GraphData
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &data)

	if len(data.Nodes) != 13 {
		t.Errorf("expected 13 card nodes only, got %d", len(data.Nodes))
	}
	// Only reference (2) + parent (3) edges when entities/tags are off.
	if len(data.Edges) != 5 {
		t.Errorf("expected 5 card edges, got %d", len(data.Edges))
	}
	for _, n := range data.Nodes {
		if n.Type != "card" {
			t.Errorf("unexpected node type %q with types=card filter", n.Type)
		}
	}
}

// TestParseGraphTypes verifies the types filter parser defaults and fallbacks.
func TestParseGraphTypes(t *testing.T) {
	all := services.ParseGraphTypes("")
	if len(all) != 3 {
		t.Errorf("expected default to include all 3 types, got %v", all)
	}
	subset := services.ParseGraphTypes("card, tag")
	if !subset["card"] || !subset["tag"] || subset["entity"] {
		t.Errorf("unexpected parse for 'card, tag': %v", subset)
	}
	// Unknown-only input falls back to the default set.
	fallback := services.ParseGraphTypes("bogus")
	if len(fallback) != 3 {
		t.Errorf("expected fallback to all types for unknown input, got %v", fallback)
	}
}
