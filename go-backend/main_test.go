package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPortConflictDetection(t *testing.T) {
	// Choose a test port that should be available
	testPort := "8081"

	// First, start a dummy server on the test port to simulate a conflict
	dummyListener, err := net.Listen("tcp", ":"+testPort)
	if err != nil {
		t.Skipf("Cannot create test listener on port %s: %v", testPort, err)
	}

	// Start a minimal HTTP server on the test port
	testServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	go testServer.Serve(dummyListener)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Now try to start our application server on the same port
	// We need to test the ListenAndServe call directly since main() uses log.Fatal
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "test")
	})

	// This should fail with "bind: address already in use" or similar
	err = http.ListenAndServe(":"+testPort, testHandler)

	// Verify that we got an error (port conflict)
	if err == nil {
		t.Error("Expected ListenAndServe to fail with port conflict, but it succeeded")
	}

	// Check that the error contains information about address already in use
	// The exact error message may vary by platform, but should indicate port conflict
	errStr := err.Error()
	if !containsPortConflictError(errStr) {
		t.Errorf("Expected port conflict error, got: %s", errStr)
	}

	// Clean up
	testServer.Close()
	dummyListener.Close()
}

func containsPortConflictError(errStr string) bool {
	// Common port conflict error patterns across different operating systems
	portConflictIndicators := []string{
		"address already in use",
		"bind: address already in use",
		"address in use",
		"bind: address in use",
	}

	for _, indicator := range portConflictIndicators {
		if contains(errStr, indicator) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (hasPrefix(s, substr) || hasSuffix(s, substr) || containsInner(s, substr)))
}

func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func hasSuffix(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	start := len(s) - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if s[start+i] != suffix[i] {
			return false
		}
	}
	return true
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if hasPrefix(s[i:], substr) {
			return true
		}
	}
	return false
}

func TestServerStartsSuccessfully(t *testing.T) {
	// Find an available port for testing
	listener, err := net.Listen("tcp", ":0") // :0 means auto-assign a free port
	if err != nil {
		t.Skipf("Cannot find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close() // Free the port immediately

	portStr := fmt.Sprintf("%d", port)

	// Create a simple handler for testing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Start server in a goroutine - it should start successfully on the free port
	go func() {
		err := http.ListenAndServe(":"+portStr, testHandler)
		if err != nil {
			// In a real server, this would be logged and cause exit
			// For testing, we just accept that it might fail during shutdown
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Try to make a test request to verify server is responding
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/test", portStr))

	if err != nil {
		t.Errorf("Could not connect to test server on port %s: %v", portStr, err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %d", resp.StatusCode)
		}
	}
}

// TestListenAndServeErrorHandling tests that our error handling works properly
// This simulates the logic from main() but in a testable way
func TestListenAndServeErrorHandling(t *testing.T) {
	// Create a dummy handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Test with an invalid port (should fail)
	err := http.ListenAndServe(":99999", handler)
	if err == nil {
		t.Error("Expected ListenAndServe to fail with invalid port, but it succeeded")
	}

	// Test with a port that's already in use
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Skip("Cannot create listener for port conflict test")
	}
	port := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)

	// Start a server on that port
	go http.Serve(listener, handler)
	time.Sleep(50 * time.Millisecond) // Give it time to start

	// Now try to start another server on the same port
	err = http.ListenAndServe(":"+port, handler)
	if err == nil {
		t.Error("Expected ListenAndServe to fail with port conflict, but it succeeded")
	}

	// Verify the error indicates a port conflict
	if !containsPortConflictError(err.Error()) {
		t.Errorf("Expected port conflict error, got: %v", err)
	}
}

func TestConfigureLoggingFlushesAndClosesFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "backend.log")
	// Anything other than "true" should be treated as non-dev.
	t.Setenv("ZETTEL_DEV", "false")
	t.Setenv("ZETTEL_BACKEND_LOG_LOCATION", logPath)

	file, cleanup, err := configureLogging()
	if err != nil {
		t.Fatalf("configureLogging() error: %v", err)
	}

	log.Printf("flush-test-line")
	cleanup()

	if _, err := file.WriteString("should-fail"); err == nil {
		t.Fatalf("expected write to closed log file to fail")
	}

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", logPath, err)
	}
	if !strings.Contains(string(b), "flush-test-line") {
		t.Fatalf("expected log output to contain flush-test-line; got: %q", string(b))
	}
}
