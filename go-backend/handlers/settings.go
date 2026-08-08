package handlers

import (
	"encoding/json"
	"net/http"
)

// publicSettingsKeys are the settings exposed on the public GET /api/settings
// endpoint. admin_email is deliberately excluded: it is the operator's
// address (used server-side for notifications/admin grant) and the frontend
// has no need for it — no reason to leak it to unauthenticated visitors.
var publicSettingsKeys = []string{
	"site_name",
	"signups_enabled",
	"oidc_auto_provision",
	"mail_enabled",
	"email_auto_validate",
	"support_email",
}

// GET /api/settings
// Public, runtime source of truth for non-secret admin settings (site name,
// signups/mail toggles, support email). Values are hot-reloaded from the
// settings file (config.yaml), so admin UI edits apply without a restart.
// Secrets never appear here: they stay env-only by construction (see the
// settings package, Zettelgarden-6er.15).
func (s *Handler) SettingsRoute(w http.ResponseWriter, r *http.Request) {
	all := s.Settings.All()
	resp := make(map[string]string, len(publicSettingsKeys))
	for _, k := range publicSettingsKeys {
		resp[k] = all[k]
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
