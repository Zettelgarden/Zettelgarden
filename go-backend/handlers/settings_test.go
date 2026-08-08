package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-backend/tests"
)

func TestSettingsRouteReturnsPublicKeys(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rr := httptest.NewRecorder()
	s.SettingsRoute(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from settings route, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode settings response: %v", err)
	}

	// Every frontend-consumable key is present.
	for _, k := range []string{"site_name", "signups_enabled", "mail_enabled", "email_auto_validate", "support_email"} {
		if _, ok := resp[k]; !ok {
			t.Errorf("missing public key %q in %v", k, resp)
		}
	}

	// admin_email must NOT leak on the public endpoint.
	if _, ok := resp["admin_email"]; ok {
		t.Error("admin_email must not be exposed on the public settings endpoint")
	}

	// Values seeded from test env / registry defaults.
	if resp["site_name"] != "Zettelgarden" {
		t.Errorf("site_name = %q, want Zettelgarden", resp["site_name"])
	}
	if resp["signups_enabled"] != "true" {
		t.Errorf("signups_enabled = %q, want true", resp["signups_enabled"])
	}
	if resp["mail_enabled"] != "true" {
		t.Errorf("mail_enabled = %q, want true (MAIL_HOST set in tests)", resp["mail_enabled"])
	}
}
