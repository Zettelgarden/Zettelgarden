package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go-backend/pkg/config"

	"github.com/typesense/typesense-go/typesense"
)

// mockTypesense is a minimal Typesense API stub used to exercise the retry
// loop without a real server. While "down" every endpoint returns 503; once
// up, the collection starts missing (404) and is created via POST /collections
// (200), after which GET /collections/{name} reports it as existing.
type mockTypesense struct {
	mu       sync.Mutex
	requests int
	downFor  int // fail the first `downFor` requests
	created  bool
}

func (m *mockTypesense) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	down := m.requests < m.downFor
	m.requests++
	created := m.created
	m.mu.Unlock()

	if down {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "service unavailable"})
		return
	}

	collectionName := strings.TrimPrefix(r.URL.Path, "/collections/")
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/collections/"):
		// Real Typesense returns 404 for a missing collection, which is what
		// InitTypesense uses to decide to attempt creation.
		if !created {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "collection not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":   collectionName,
			"fields": []map[string]interface{}{},
		})
	case r.Method == http.MethodPost && r.URL.Path == "/collections":
		m.mu.Lock()
		m.created = true
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":   collectionName,
			"fields": []map[string]interface{}{},
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *mockTypesense) requestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests
}

func TestRetryInitTypesenseConnectsWhenTypesenseComesUp(t *testing.T) {
	const collectionName = "cards"
	// Fail the first three requests (attempt 1: GET+POST, attempt 2: GET) so
	// the loop is forced to retry before Typesense "comes up" and the
	// collection gets created.
	mock := &mockTypesense{downFor: 3}
	ts := httptest.NewServer(mock)
	defer ts.Close()

	cfg := config.SearchConfig{
		Host:       ts.URL,
		Password:   "test-key",
		Collection: collectionName,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan *typesense.Client, 1)
	RetryInitTypesense(ctx, cfg, 10*time.Millisecond, 50*time.Millisecond, func(c *typesense.Client) {
		ready <- c
	})

	var client *typesense.Client
	select {
	case client = <-ready:
	case <-time.After(5 * time.Second):
		t.Fatalf("onReady was not called after Typesense came back up (requests=%d)", mock.requestCount())
	}
	if client == nil {
		t.Fatal("onReady received a nil client")
	}

	// The client should be usable against the (now up) mock: the collection
	// was created, so Retrieve succeeds with the expected name.
	resp, err := client.Collection(collectionName).Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve failed with the retried client: %v", err)
	}
	if resp.Name != collectionName {
		t.Errorf("Retrieve returned collection %q, want %q", resp.Name, collectionName)
	}

	// Sanity: the retry loop really did hit the down phase before succeeding.
	if n := mock.requestCount(); n < mock.downFor {
		t.Errorf("mock saw %d requests, expected at least %d (initial failures)", n, mock.downFor)
	}
}

func TestRetryInitTypesenseDoesNotFireWhileDownAndStopsOnCancel(t *testing.T) {
	const collectionName = "cards"
	mock := &mockTypesense{downFor: int(^uint(0) >> 1)} // always down
	ts := httptest.NewServer(mock)
	defer ts.Close()

	cfg := config.SearchConfig{
		Host:       ts.URL,
		Password:   "test-key",
		Collection: collectionName,
	}

	ctx, cancel := context.WithCancel(context.Background())

	ready := make(chan *typesense.Client, 1)
	RetryInitTypesense(ctx, cfg, 5*time.Millisecond, 10*time.Millisecond, func(c *typesense.Client) {
		ready <- c
	})

	// While Typesense stays down, onReady must never fire.
	select {
	case c := <-ready:
		t.Fatalf("onReady fired while Typesense was down (client=%v)", c)
	case <-time.After(150 * time.Millisecond):
	}

	// Cancelling the context stops the retry loop; it must not fire later.
	cancel()
	select {
	case c := <-ready:
		t.Fatalf("onReady fired after context cancellation (client=%v)", c)
	case <-time.After(100 * time.Millisecond):
	}
}
