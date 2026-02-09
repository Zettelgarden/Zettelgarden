package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// Test server for mock HTTP responses
type mockFeedServer struct {
	server *httptest.Server
	URL    string
}

func newMockFeedServer(handler http.HandlerFunc) *mockFeedServer {
	server := httptest.NewServer(handler)
	return &mockFeedServer{
		server: server,
		URL:    server.URL,
	}
}

func (m *mockFeedServer) Close() {
	m.server.Close()
}

// ============================================================================
// Request Validation Tests
// ============================================================================

func TestDiscoverFeedRoute_InvalidRequest(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Test with empty request body
	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 400 for empty URL
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty URL, got %v", status)
	}
}

func TestDiscoverFeedRoute_MalformedURL(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Test with malformed URL
	requestBody := map[string]string{"url": "not-a-valid-url"}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 400 for malformed URL
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed URL, got %v", status)
	}
}

func TestDiscoverFeedRoute_MissingURL(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Test with missing URL field
	requestBody := map[string]string{}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 400 for missing URL
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing URL, got %v", status)
	}

	// Check error message
	if body := rr.Body.String(); body == "" {
		t.Errorf("Expected error message in response body, got empty string")
	}
}

func TestDiscoverFeedRoute_InvalidScheme(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	// Test with invalid scheme (ftp://)
	requestBody := map[string]string{"url": "ftp://example.com/feed.xml"}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 400 for invalid scheme
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid scheme, got %v", status)
	}
}

// ============================================================================
// Successful Feed Discovery Tests
// ============================================================================

func TestDiscoverFeedRoute_ValidRSSFeed(t *testing.T) {
	// Create mock server with RSS feed in HTML head
	// Note: Handler prefers feed title over page title when link has title attribute
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>Test Blog</title>
	<link rel="alternate" type="application/rss+xml" href="/feed.xml" />
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	// Verify feed URL is correctly resolved
	expectedFeedURL := mockServer.URL + "/feed.xml"
	if response.FeedURL != expectedFeedURL {
		t.Errorf("Expected feed URL %s, got %s", expectedFeedURL, response.FeedURL)
	}

	// Verify title is extracted (page title when feed link has no title)
	if response.Title != "Test Blog" {
		t.Errorf("Expected title 'Test Blog', got '%s'", response.Title)
	}
}

func TestDiscoverFeedRoute_AtomFeedOnly(t *testing.T) {
	// Create mock server with Atom feed only
	// Note: Handler prefers feed title over page title when link has title attribute
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>Atom Blog</title>
	<link rel="alternate" type="application/atom+xml" href="/atom.xml" />
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	expectedFeedURL := mockServer.URL + "/atom.xml"
	if response.FeedURL != expectedFeedURL {
		t.Errorf("Expected feed URL %s, got %s", expectedFeedURL, response.FeedURL)
	}

	if response.Title != "Atom Blog" {
		t.Errorf("Expected title 'Atom Blog', got '%s'", response.Title)
	}
}

func TestDiscoverFeedRoute_MultipleFeeds(t *testing.T) {
	// Create mock server with multiple feeds
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>Multi Feed Blog</title>
	<link rel="alternate" type="application/atom+xml" title="Atom Feed" href="/atom.xml" />
	<link rel="alternate" type="application/rss+xml" title="RSS Feed" href="/rss.xml" />
	<link rel="alternate" type="application/rss+xml" title="RSS 2.0" href="/feed.xml" />
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	// Should prefer RSS over Atom
	if !strings.Contains(response.FeedURL, "rss.xml") && !strings.Contains(response.FeedURL, "feed.xml") {
		t.Errorf("Expected RSS feed URL, got %s", response.FeedURL)
	}
}

func TestDiscoverFeedRoute_DirectFeedURL(t *testing.T) {
	// Create mock server that returns RSS directly
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Direct Feed</title>
	<link>https://example.com</link>
	<description>A test feed</description>
</channel>
</rss>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	// Should return the URL itself as the feed URL
	if response.FeedURL != mockServer.URL {
		t.Errorf("Expected feed URL %s, got %s", mockServer.URL, response.FeedURL)
	}
}

// ============================================================================
// Feed Discovery with Fallback Paths Tests
// ============================================================================

func TestDiscoverFeedRoute_FallbackToFeedPath(t *testing.T) {
	// Create a mock server that:
	// 1. Returns HTML without feed links for the main URL
	// 2. Returns RSS for /feed path
	feedRequested := false
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			feedRequested = true
			w.Header().Set("Content-Type", "application/rss+xml")
			fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Fallback Feed</title>
</channel>
</rss>`)
		} else {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>Blog with /feed</title>
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
		}
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	// Verify that the fallback /feed path was tried
	if !feedRequested {
		t.Error("Expected fallback /feed path to be requested")
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	expectedFeedURL := mockServer.URL + "/feed"
	if response.FeedURL != expectedFeedURL {
		t.Errorf("Expected feed URL %s, got %s", expectedFeedURL, response.FeedURL)
	}
}

// ============================================================================
// Error Cases Tests
// ============================================================================

func TestDiscoverFeedRoute_NoFeedFound(t *testing.T) {
	// Create mock server with no feed
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		// Return HTML for main request, 404 for all feed paths
		if r.URL.Path == "/feed" || r.URL.Path == "/rss" || r.URL.Path == "/atom.xml" || r.URL.Path == "/feed.xml" || r.URL.Path == "/rss.xml" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>No Feed Blog</title>
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
		}
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 404 when no feed is found (error message contains "no feed found")
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status 404, got %v. Body: %s", status, rr.Body.String())
	}

	// Check error message contains helpful information
	body := rr.Body.String()
	if !strings.Contains(body, "feed") {
		t.Errorf("Expected error message to mention 'feed', got: %s", body)
	}
}

func TestDiscoverFeedRoute_NetworkTimeout(t *testing.T) {
	// Note: This test verifies timeout handling behavior.
	// The actual timeout detection depends on the error type implementing Timeout() method.
	// HTTP timeout errors from the standard library do implement this interface.
	skipIfShort(t)

	// Create mock server that delays response
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the discovery timeout (10 seconds)
		time.Sleep(15 * time.Second)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<html><body>Slow response</body></html>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 504 Gateway Timeout or 400 for timeout
	// The actual status depends on whether the error implements Timeout() method
	if status := rr.Code; status != http.StatusGatewayTimeout && status != http.StatusBadRequest {
		t.Errorf("Expected status 504 or 400 for timeout, got %v. Body: %s", status, rr.Body.String())
	}

	// Check error message mentions timeout or deadline exceeded
	body := rr.Body.String()
	bodyLower := strings.ToLower(body)
	if !strings.Contains(bodyLower, "timeout") && !strings.Contains(bodyLower, "deadline") {
		t.Errorf("Expected error message to mention 'timeout' or 'deadline', got: %s", body)
	}
}

// Helper function to skip long-running tests in short mode
func skipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}
}

func TestDiscoverFeedRoute_NonHTMLResponse(t *testing.T) {
	// Create mock server that returns non-HTML, non-feed content
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"message": "This is JSON, not HTML"}`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	// Should return 404 for non-HTML response (error contains "non-HTML response")
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-HTML response, got %v. Body: %s", status, rr.Body.String())
	}
}

func TestDiscoverFeedRoute_RelativeFeedURL(t *testing.T) {
	// Create mock server with relative feed URL
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>Relative Feed Blog</title>
	<link rel="alternate" type="application/rss+xml" title="RSS" href="feeds/rss.xml" />
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL + "/blog/"}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	// Relative URL should be resolved against base URL
	if !strings.Contains(response.FeedURL, "feeds/rss.xml") {
		t.Errorf("Expected resolved feed URL to contain 'feeds/rss.xml', got %s", response.FeedURL)
	}
}

func TestDiscoverFeedRoute_HTMLSpecialCharsInTitle(t *testing.T) {
	// Create mock server with HTML entities in page title
	// Note: Handler uses page title when feed link has no title attribute
	mockServer := newMockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head>
	<title>Blog &amp; News &raquo; Tech</title>
	<link rel="alternate" type="application/rss+xml" href="/feed.xml" />
</head>
<body>
	<h1>Welcome</h1>
</body>
</html>`)
	})
	defer mockServer.Close()

	s := NewHandler()
	defer tests.Teardown()

	token, _ := tests.GenerateTestJWT(1)

	requestBody := map[string]string{"url": mockServer.URL}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/api/rss/discover", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/rss/discover", s.JwtMiddleware(s.DiscoverFeedRoute))
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v. Body: %s", status, rr.Body.String())
	}

	var response DiscoverFeedResponse
	tests.ParseJsonResponse(t, rr.Body.Bytes(), &response)

	// HTML entities should be unescaped
	expectedTitle := "Blog & News \u00bb Tech"
	if response.Title != expectedTitle && response.Title != "Blog & News » Tech" {
		t.Errorf("Expected unescaped title '%s' or 'Blog & News » Tech', got '%s'", expectedTitle, response.Title)
	}
}
