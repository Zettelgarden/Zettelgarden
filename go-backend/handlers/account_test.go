package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"go-backend/models"
	"go-backend/tests"

	"github.com/gorilla/mux"
)

// clearUsers empties the users table inside the test transaction (rolled back
// by Teardown) to simulate a fresh install. keywords and entity_card_junction
// reference cards/entities with NO ACTION FKs, so clear them first or the
// user-delete cascade fails (same as TestCreateUserFirstUserBecomesAdmin).
func clearUsers(t *testing.T, s *Handler) {
	t.Helper()
	for _, table := range []string{"keywords", "entity_card_junction"} {
		if _, err := s.GetDB().Exec(`DELETE FROM ` + table); err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	if _, err := s.GetDB().Exec(`DELETE FROM users`); err != nil {
		t.Fatalf("clear users table: %v", err)
	}
}

// createTestUser creates a user with a known password and returns the id.
func createTestUser(t *testing.T, s *Handler, username, email string) int {
	t.Helper()
	id, err := s.CreateUser(models.CreateUserParams{
		Username:        username,
		Email:           email,
		Password:        "password123",
		ConfirmPassword: "password123",
	})
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return id
}

func userContext(req *http.Request, userID int) *http.Request {
	ctx := context.WithValue(req.Context(), "current_user", userID)
	return req.WithContext(ctx)
}

// zipEntries returns the named entry's bytes from a zip reader, or nil.
func zipEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open export zip: %v", err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", f.Name, err)
		}
		out[f.Name] = b
	}
	return out
}

// zipEntryNames returns the entry names for diagnostics.
func zipEntryNames(entries map[string][]byte) []string {
	names := make([]string, 0, len(entries))
	for k := range entries {
		names = append(names, k)
	}
	return names
}

// TestExportUserDataRoute verifies the export zip ships table JSON, raw file
// blobs, and the human-readable card renderings.
func TestExportUserDataRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	clearUsers(t, s)

	userID := createTestUser(t, s, "exporter", "exporter@example.com")

	// Seed a card, a tag, and an uploaded file (row + blob).
	if _, err := s.GetDB().Exec(
		`INSERT INTO cards (user_id, card_id, title, body, created_at, updated_at) VALUES ($1, 'card-1', 'Hello World', 'Body text', datetime('now'), datetime('now'))`,
		userID); err != nil {
		t.Fatalf("insert card: %v", err)
	}
	if _, err := s.GetDB().Exec(`INSERT INTO tags (user_id, name) VALUES ($1, 'mytag')`, userID); err != nil {
		t.Fatalf("insert tag: %v", err)
	}
	blob := "file-bytes-123"
	if err := s.Server.Store.Upload(context.Background(), "1/export-test.txt", strings.NewReader(blob)); err != nil {
		t.Fatalf("upload blob: %v", err)
	}
	if _, err := s.GetDB().Exec(
		`INSERT INTO files (user_id, name, type, path, filename, size) VALUES ($1, 'note.txt', 'text/plain', '1/export-test.txt', 'note.txt', $2)`,
		userID, len(blob)); err != nil {
		t.Fatalf("insert file row: %v", err)
	}
	var fileID int
	if err := s.GetDB().QueryRow(`SELECT id FROM files WHERE path = '1/export-test.txt'`).Scan(&fileID); err != nil {
		t.Fatalf("find file row: %v", err)
	}

	req := userContext(httptest.NewRequest(http.MethodGet, "/api/user/export", nil), userID)
	rr := httptest.NewRecorder()
	s.ExportUserDataRoute(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("export status: got %d want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/zip") {
		t.Errorf("export content-type %q, want application/zip", ct)
	}

	entries := zipEntries(t, rr.Body.Bytes())

	// Table JSON includes the card and tag.
	if b, ok := entries["tables/cards.json"]; !ok {
		t.Error("missing tables/cards.json in export")
	} else if !strings.Contains(string(b), "Hello World") {
		t.Errorf("cards.json missing card title: %s", b)
	}
	if b, ok := entries["tables/tags.json"]; !ok || !strings.Contains(string(b), "mytag") {
		t.Errorf("tags.json missing tag: %v", string(b))
	}

	// Raw file blob is included with the file bytes.
	wantFileEntry := "files/" + strconv.Itoa(fileID) + "_note.txt"
	if b, ok := entries[wantFileEntry]; !ok {
		t.Errorf("missing %s in export (have %v)", wantFileEntry, zipEntryNames(entries))
	} else if string(b) != blob {
		t.Errorf("exported file bytes %q, want %q", b, blob)
	}

	// Markdown + CSV card renderings exist.
	if b, ok := entries["cards.md"]; !ok || !strings.Contains(string(b), "Hello World") {
		t.Errorf("cards.md missing card: %v", string(b))
	}
	if b, ok := entries["cards.csv"]; !ok || !strings.Contains(string(b), "card-1") {
		t.Errorf("cards.csv missing card: %v", string(b))
	}

	// user.json has the profile but no password field.
	ub, ok := entries["user.json"]
	if !ok {
		t.Fatal("missing user.json in export")
	}
	var u map[string]interface{}
	if err := json.Unmarshal(ub, &u); err != nil {
		t.Fatalf("user.json not JSON: %v", err)
	}
	if u["username"] != "exporter" {
		t.Errorf("user.json username %v, want exporter", u["username"])
	}
	if _, has := u["password"]; has {
		t.Error("user.json must not contain the password hash")
	}
}

// TestDeleteAccountRoute verifies self-serve deletion: wrong password is
// rejected, correct password cascades the user's data, and file blobs are
// removed from storage.
func TestDeleteAccountRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	clearUsers(t, s)

	adminID := createTestUser(t, s, "admin", "admin@example.com") // first user -> admin
	userID := createTestUser(t, s, "deletee", "deletee@example.com")

	// Seed user data: a card (cascade), a task_saved_search (non-cascade),
	// and a file row + blob.
	if _, err := s.GetDB().Exec(
		`INSERT INTO cards (user_id, card_id, title, created_at, updated_at) VALUES ($1, 'card-2', 'To Delete', datetime('now'), datetime('now'))`,
		userID); err != nil {
		t.Fatalf("insert card: %v", err)
	}
	if _, err := s.GetDB().Exec(`INSERT INTO task_saved_searches (user_id, name) VALUES ($1, 'mysearch')`, userID); err != nil {
		t.Fatalf("insert saved search: %v", err)
	}
	if err := s.Server.Store.Upload(context.Background(), "2/del-test.txt", strings.NewReader("delete-me")); err != nil {
		t.Fatalf("upload blob: %v", err)
	}
	if _, err := s.GetDB().Exec(
		`INSERT INTO files (user_id, name, type, path, filename, size) VALUES ($1, 'x.txt', 'text/plain', '2/del-test.txt', 'x.txt', 9)`,
		userID); err != nil {
		t.Fatalf("insert file row: %v", err)
	}

	// Wrong password -> 403.
	req := userContext(httptest.NewRequest(http.MethodDelete, "/api/user", strings.NewReader(`{"password":"wrong"}`)), userID)
	rr := httptest.NewRecorder()
	s.DeleteAccountRoute(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("wrong password: got %d want 403", rr.Code)
	}

	// Correct password -> 200.
	req = userContext(httptest.NewRequest(http.MethodDelete, "/api/user", strings.NewReader(`{"password":"password123"}`)), userID)
	rr = httptest.NewRecorder()
	s.DeleteAccountRoute(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete account: got %d want 200 (body: %s)", rr.Code, rr.Body.String())
	}

	// User row gone.
	var n int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if n != 0 {
		t.Errorf("user %d still exists after delete", userID)
	}

	// Cascaded data gone (card), non-cascade table cleaned (saved search).
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM cards WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if n != 0 {
		t.Errorf("cards for user %d not cascaded away", userID)
	}
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM task_saved_searches WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count saved searches: %v", err)
	}
	if n != 0 {
		t.Errorf("task_saved_searches for user %d not cleaned", userID)
	}

	// File blob removed from storage.
	exists, err := s.Server.Store.Exists(context.Background(), "2/del-test.txt")
	if err != nil {
		t.Fatalf("store exists: %v", err)
	}
	if exists {
		t.Error("file blob still on disk after account deletion")
	}

	// The admin user is untouched.
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE id = $1`, adminID).Scan(&n); err != nil {
		t.Fatalf("count admin: %v", err)
	}
	if n != 1 {
		t.Errorf("admin user %d should survive", adminID)
	}
}

// TestDeleteAccountLastAdminGuard verifies the final admin cannot delete their
// own account (would lock the instance out of admin settings).
func TestDeleteAccountLastAdminGuard(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	clearUsers(t, s)

	adminID := createTestUser(t, s, "only-admin", "only@example.com") // first user -> admin

	req := userContext(httptest.NewRequest(http.MethodDelete, "/api/user", strings.NewReader(`{"password":"password123"}`)), adminID)
	rr := httptest.NewRecorder()
	s.DeleteAccountRoute(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("last admin delete: got %d want 400 (body: %s)", rr.Code, rr.Body.String())
	}

	var n int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE id = $1`, adminID).Scan(&n); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if n != 1 {
		t.Errorf("last admin was deleted: users=%d", n)
	}
}

// TestDeleteUserRoute verifies admin-only account deletion.
func TestDeleteUserRoute(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	clearUsers(t, s)

	adminID := createTestUser(t, s, "admin2", "admin2@example.com") // first -> admin
	targetID := createTestUser(t, s, "target", "target@example.com")

	if _, err := s.GetDB().Exec(
		`INSERT INTO cards (user_id, card_id, title, created_at, updated_at) VALUES ($1, 'card-3', 'Admin Deleted', datetime('now'), datetime('now'))`,
		targetID); err != nil {
		t.Fatalf("insert card: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+strconv.Itoa(targetID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(targetID)})
	req = userContext(req, adminID)
	rr := httptest.NewRecorder()
	s.DeleteUserRoute(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin delete: got %d want 200 (body: %s)", rr.Code, rr.Body.String())
	}

	var n int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE id = $1`, targetID).Scan(&n); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if n != 0 {
		t.Errorf("target user %d still exists after admin delete", targetID)
	}
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM cards WHERE user_id = $1`, targetID).Scan(&n); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if n != 0 {
		t.Errorf("cards for target user %d not cascaded away", targetID)
	}
}
