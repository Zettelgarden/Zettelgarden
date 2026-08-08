package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"go-backend/handlers"
	"go-backend/settings"
	"go-backend/tests"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// setupSettingsTestHandler returns a Handler wired with a real settings
// manager (seeded from test env) and an admin user (id 1) for audit logging.
func setupSettingsTestHandler(t *testing.T) *handlers.Handler {
	s := tests.Setup()
	t.Cleanup(tests.Teardown)

	sm, err := settings.New(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("settings init: %v", err)
	}

	h := &handlers.Handler{
		DB:       s.DB,
		Server:   s,
		Settings: sm,
	}
	s.Settings = sm

	if _, err := h.DB.Exec(`UPDATE users SET is_admin = true WHERE id = 1`); err != nil {
		t.Fatalf("make user 1 admin: %v", err)
	}
	return h
}

func adminRequest(t *testing.T, h *handlers.Handler, method string, body []byte) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest(method, "/api/admin/settings", bytes.NewReader(body))
	// current_user in context for LogAdminAction audit.
	ctx := context.WithValue(req.Context(), "current_user", 1)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	return rr, req
}

func TestGetAdminSettingsRouteIncludesAdminEmail(t *testing.T) {
	h := setupSettingsTestHandler(t)

	rr, req := adminRequest(t, h, http.MethodGet, nil)
	GetAdminSettingsRoute(h, rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// admin_email present on the ADMIN endpoint (unlike the public one).
	if resp["admin_email"] != "admin@test.com" {
		t.Errorf("admin_email = %q, want admin@test.com (seeded from test env)", resp["admin_email"])
	}
	if resp["site_name"] != "Zettelgarden" {
		t.Errorf("site_name = %q, want Zettelgarden", resp["site_name"])
	}
}

func TestUpdateAdminSettingsRoute(t *testing.T) {
	h := setupSettingsTestHandler(t)

	body, _ := json.Marshal(map[string]string{
		"signups_enabled": "false",
		"site_name":       "My Notes",
	})
	rr, req := adminRequest(t, h, http.MethodPut, body)
	UpdateAdminSettingsRoute(h, rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Response returns the full new settings map.
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["signups_enabled"] != "false" || resp["site_name"] != "My Notes" {
		t.Errorf("response does not reflect update: %v", resp)
	}
	// Untouched keys survive.
	if resp["admin_email"] != "admin@test.com" {
		t.Errorf("admin_email clobbered: %q", resp["admin_email"])
	}

	// Persisted: a fresh read sees the update.
	if got := h.Settings.Get("signups_enabled"); got != "false" {
		t.Errorf("settings.Get(signups_enabled) = %q, want false", got)
	}

	// Audited. LogAdminAction writes through GetDB() (the test transaction),
	// so read the audit row back through the same Tx.
	var auditCount int
	if err := h.Server.Tx.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = 'settings.update'`).Scan(&auditCount); err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if auditCount != 1 {
		t.Errorf("expected 1 settings.update audit entry, got %d", auditCount)
	}
}

func TestUpdateAdminSettingsRouteRejectsInvalid(t *testing.T) {
	h := setupSettingsTestHandler(t)

	// Invalid bool in the batch fails the whole write.
	body, _ := json.Marshal(map[string]string{
		"signups_enabled": "not-a-bool",
		"site_name":       "should not land",
	})
	rr, req := adminRequest(t, h, http.MethodPut, body)
	UpdateAdminSettingsRoute(h, rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid bool, got %d", rr.Code)
	}
	if got := h.Settings.Get("site_name"); got != "Zettelgarden" {
		t.Errorf("site_name changed despite failed batch: %q", got)
	}

	// Unknown key.
	body, _ = json.Marshal(map[string]string{"nope": "x"})
	rr, req = adminRequest(t, h, http.MethodPut, body)
	UpdateAdminSettingsRoute(h, rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown key, got %d", rr.Code)
	}

	// Empty body.
	rr, req = adminRequest(t, h, http.MethodPut, []byte("{}"))
	UpdateAdminSettingsRoute(h, rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty update, got %d", rr.Code)
	}
}
