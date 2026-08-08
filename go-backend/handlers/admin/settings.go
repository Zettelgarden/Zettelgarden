package admin

import (
	"encoding/json"
	"net/http"

	"go-backend/handlers"
)

// GetAdminSettingsRoute returns ALL admin-managed settings — including
// admin_email, which the public /api/settings endpoint deliberately excludes
// (operator address, no frontend need). Values are hot-reloaded from the
// settings file (config.yaml), so edits made here apply without a restart.
// Admin protected (via middleware).
func GetAdminSettingsRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	resp := h.Settings.All()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateAdminSettingsRoute accepts a partial update (subset of registry keys)
// as {"key": "value"}, validates the whole batch, persists atomically (one
// file write), and audits the change via the admin audit log with before/after
// values. Returns the full new settings map so the UI can refresh in place.
// Admin protected (via middleware).
func UpdateAdminSettingsRoute(h *handlers.Handler, w http.ResponseWriter, r *http.Request) {
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(updates) == 0 {
		http.Error(w, "no settings provided", http.StatusBadRequest)
		return
	}

	before := h.Settings.All()
	if err := h.Settings.SetMany(updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	details := make(map[string]interface{}, len(updates))
	for key, value := range updates {
		details[key] = map[string]string{"from": before[key], "to": value}
	}
	h.LogAdminAction(r, "settings.update", "settings", 0, details)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Settings.All())
}
