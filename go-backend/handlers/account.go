package handlers

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-backend/models"
	"go-backend/services"

	"github.com/gorilla/mux"
)

// ---------------------------------------------------------------------------
// Self-serve data export + account deletion (6er.9).
//
// A self-hosted instance must let users get their data out and delete their
// account without emailing a SaaS support address. Export streams a zip of the
// user's tables (as JSON), a markdown/CSV rendering of cards, and the raw file
// blobs; deletion cascades all user-owned rows (SQLite ON DELETE CASCADE) after
// explicitly cleaning the few tables that predate the FK.
// ---------------------------------------------------------------------------

// exportTable describes one table to dump into the export zip. where is the
// SQL WHERE clause (with $1 bound to the user id), or "" for a full dump.
type exportTable struct {
	name  string
	where string
}

// userScopedTables lists every user-owned table that should ship in an export,
// with the clause that scopes it to one user. Junction tables are scoped via a
// subquery on their owning parent table.
var userScopedTables = []exportTable{
	{"cards", "user_id = $1"},
	{"card_tags", "card_pk IN (SELECT id FROM cards WHERE user_id = $1)"},
	{"card_templates", "user_id = $1"},
	{"card_views", "user_id = $1"},
	{"card_chunks", "user_id = $1"},
	{"card_embeddings", "user_id = $1"},
	{"backlinks", "source_id_int IN (SELECT id FROM cards WHERE user_id = $1) OR target_id_int IN (SELECT id FROM cards WHERE user_id = $1)"},
	{"keywords", "user_id = $1"},
	{"tags", "user_id = $1"},
	{"file_tags", "user_id = $1"},
	{"files", "user_id = $1"},
	{"files_tags", "file_id IN (SELECT id FROM files WHERE user_id = $1)"},
	{"tasks", "user_id = $1"},
	{"task_tags", "task_pk IN (SELECT id FROM tasks WHERE user_id = $1)"},
	{"task_dependencies", "task_id IN (SELECT id FROM tasks WHERE user_id = $1) OR blocking_task_id IN (SELECT id FROM tasks WHERE user_id = $1)"},
	{"task_statuses", "user_id = $1"},
	{"task_saved_searches", "user_id = $1"},
	{"entities", "user_id = $1"},
	{"entity_card_junction", "user_id = $1"},
	{"entity_fact_junction", "user_id = $1"},
	{"facts", "user_id = $1"},
	{"fact_card_junction", "user_id = $1"},
	{"starred_cards", "user_id = $1"},
	{"starred_searches", "user_id = $1"},
	{"rss_feeds", "user_id = $1"},
	{"rss_folders", "user_id = $1"},
	{"rss_seen_articles", "feed_id IN (SELECT id FROM rss_feeds WHERE user_id = $1)"},
	{"chat_conversations", "user_id = $1"},
	{"chat_instructions", "user_id = $1"},
	{"chat_messages", "conversation_id IN (SELECT id FROM chat_conversations WHERE user_id = $1)"},
	{"chat_tool_calls", "user_id = $1"},
	{"chat_usage_quotas", "user_id = $1"},
	{"emails", "user_id = $1"},
	{"email_accounts", "user_id = $1"},
	{"email_attachments", "user_id = $1"},
	{"email_card_links", "email_id IN (SELECT id FROM emails WHERE user_id = $1)"},
	{"email_fact_junction", "user_id = $1"},
	{"email_triage_decisions", "email_id IN (SELECT id FROM emails WHERE user_id = $1)"},
	{"summarizations", "user_id = $1"},
	{"api_keys", "user_id = $1"},
	{"notifications", "user_id = $1"},
	{"notification_preferences", "user_id = $1"},
	{"inactive_cards", "user_id = $1"},
	{"schema_definitions", "owner_id = $1"},
	{"spreadsheets", "user_id = $1"},
	{"flashcard_reviews", "user_id = $1"},
	{"llm_jobs", "user_id = $1"},
	{"llm_query_log", "user_id = $1"},
	{"user_stats", "user_id = $1"},
	{"user_llm_configurations", "user_id = $1"},
}

// dumpTable returns every row of table scoped by where (with arg bound to $1)
// as a slice of column maps. Values come back in driver-native types (int64,
// string, []byte, float64), which JSON-encodes losslessly.
func dumpTable(db models.Database, table, where string, arg int) ([]map[string]interface{}, error) {
	query := "SELECT * FROM " + table
	if where != "" {
		query += " WHERE " + where
	}
	rows, err := db.Query(query, arg)
	if err != nil {
		return nil, fmt.Errorf("dump %s: %w", table, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]interface{}{}
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for rows.Next() {
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("dump %s scan: %w", table, err)
		}
		row := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// writeJSONEntry writes one zip entry with the JSON encoding of v.
func writeJSONEntry(zw *zip.Writer, name string, v interface{}) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// sanitizeFilename makes a file name safe to use inside a zip path.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_")
	return replacer.Replace(name)
}

// GET /api/user/export
// Streams a zip archive containing the caller's data: one JSON file per table,
// a markdown + CSV rendering of cards, and the raw uploaded file blobs.
func (s *Handler) ExportUserDataRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	user, err := s.QueryUser(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="zettelgarden-export-%d-%s.zip"`, userID, time.Now().Format("2006-01-02")))

	zw := zip.NewWriter(w)
	defer zw.Close()

	// 1. user profile (sensitive fields stripped)
	if err := writeJSONEntry(zw, "user.json", sanitizeUserForExport(user)); err != nil {
		log.Printf("export: user.json: %v", err)
		return
	}

	// 2. every user-scoped table as JSON
	for _, t := range userScopedTables {
		rows, err := dumpTable(s.GetDB(), t.name, t.where, userID)
		if err != nil {
			log.Printf("export: %s: %v", t.name, err)
			continue
		}
		if err := writeJSONEntry(zw, "tables/"+t.name+".json", rows); err != nil {
			log.Printf("export: %s: %v", t.name, err)
			return
		}
	}

	// 3. human-readable card renderings
	if err := s.writeCardsMarkdown(zw, userID); err != nil {
		log.Printf("export: cards.md: %v", err)
	}
	if err := s.writeCardsCSV(zw, userID); err != nil {
		log.Printf("export: cards.csv: %v", err)
	}

	// 4. raw file blobs
	s.writeFileBlobs(zw, r.Context(), userID)

	if err := writeJSONEntry(zw, "export.json", map[string]interface{}{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"user_id":     userID,
		"format":      "zettelgarden-data-export-v1",
	}); err != nil {
		log.Printf("export: export.json: %v", err)
	}
}

// sanitizeUserForExport returns the user profile without credentials.
func sanitizeUserForExport(u models.User) map[string]interface{} {
	// Re-marshal through the struct's json tags minus the secret fields.
	secrets := map[string]bool{
		"password": true, "caldav_token": true,
	}
	raw, err := json.Marshal(u)
	if err != nil {
		return map[string]interface{}{"id": u.ID, "username": u.Username, "email": u.Email}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]interface{}{"id": u.ID, "username": u.Username, "email": u.Email}
	}
	for k := range secrets {
		delete(m, k)
	}
	return m
}

// writeCardsMarkdown writes cards.md: one markdown section per non-deleted card.
func (s *Handler) writeCardsMarkdown(zw *zip.Writer, userID int) error {
	rows, err := dumpTable(s.GetDB(), "cards", "user_id = $1 AND is_deleted = 0", userID)
	if err != nil {
		return err
	}
	w, err := zw.Create("cards.md")
	if err != nil {
		return err
	}
	var b strings.Builder
	for _, c := range rows {
		title := str(c["title"])
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(&b, "## %s\n\n", title)
		if body := str(c["body"]); body != "" {
			fmt.Fprintf(&b, "%s\n\n", body)
		}
		if link := str(c["link"]); link != "" {
			fmt.Fprintf(&b, "Link: %s\n\n", link)
		}
		fmt.Fprintf(&b, "Card ID: %v\n", c["card_id"])
		fmt.Fprintf(&b, "Created: %v\n", c["created_at"])
		b.WriteString("\n---\n\n")
	}
	_, err = io.WriteString(w, b.String())
	return err
}

// writeCardsCSV writes cards.csv with the core card columns.
func (s *Handler) writeCardsCSV(zw *zip.Writer, userID int) error {
	rows, err := dumpTable(s.GetDB(), "cards", "user_id = $1", userID)
	if err != nil {
		return err
	}
	w, err := zw.Create("cards.csv")
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cw.Write([]string{"id", "card_id", "title", "link", "created_at", "updated_at", "is_deleted"})
	for _, c := range rows {
		cw.Write([]string{
			str(c["id"]), str(c["card_id"]), str(c["title"]), str(c["link"]),
			str(c["created_at"]), str(c["updated_at"]), str(c["is_deleted"]),
		})
	}
	return cw.Error()
}

// str renders a scanned cell as a string (nil-safe).
func str(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// writeFileBlobs streams each uploaded file's bytes into files/<id>_<name>.
// Blob read failures are logged and skipped (metadata still ships in
// tables/files.json).
func (s *Handler) writeFileBlobs(zw *zip.Writer, ctx context.Context, userID int) {
	rows, err := s.GetDB().Query(`SELECT id, filename, path FROM files WHERE user_id = $1`, userID)
	if err != nil {
		log.Printf("export: file list: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var filename, path string
		if err := rows.Scan(&id, &filename, &path); err != nil {
			log.Printf("export: file scan: %v", err)
			continue
		}
		if path == "" {
			continue
		}
		rc, err := s.Server.Store.Download(ctx, path)
		if err != nil {
			log.Printf("export: download %s: %v", path, err)
			continue
		}
		name := fmt.Sprintf("files/%d_%s", id, sanitizeFilename(filename))
		w, err := zw.Create(name)
		if err != nil {
			rc.Close()
			log.Printf("export: zip create %s: %v", name, err)
			return
		}
		if _, err := io.Copy(w, rc); err != nil {
			log.Printf("export: copy %s: %v", name, err)
		}
		rc.Close()
	}
}

// ---------------------------------------------------------------------------
// Account deletion
// ---------------------------------------------------------------------------

// deleteUserData runs the cascade delete in a transaction (the test tx during
// tests, per ShouldCommitTx) and returns the storage keys of the user's file
// blobs so the caller can remove the bytes after commit.
func (s *Handler) deleteUserData(userID int) ([]string, error) {
	var keys []string
	err := func() error {
		tx, err := s.BeginTx()
		if err != nil {
			return err
		}
		keys, err = services.DeleteUserData(tx, userID)
		if err != nil {
			return err
		}
		if s.ShouldCommitTx() {
			return tx.Commit()
		}
		return nil
	}()
	return keys, err
}

// removeFileBlobs best-effort deletes the given storage keys from disk.
func (s *Handler) removeFileBlobs(ctx context.Context, keys []string) {
	for _, k := range keys {
		if err := s.Server.Store.Delete(ctx, k); err != nil {
			log.Printf("delete account: failed to remove blob %q: %v", k, err)
		}
	}
}

// lastAdminGuard rejects deleting the final admin account.
func (s *Handler) lastAdminGuard(target models.User) error {
	if !target.IsAdmin {
		return nil
	}
	admins, err := services.CountAdmins(s.GetDB())
	if err != nil {
		return err
	}
	if admins <= 1 {
		return fmt.Errorf("cannot delete the last admin account")
	}
	return nil
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
}

// DELETE /api/user
// Self-serve account deletion. Local users must confirm their password; users
// provisioned via GitHub/OIDC (no local password) are authenticated by their
// session token alone. Admins cannot delete the last remaining admin account.
func (s *Handler) DeleteAccountRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	user, err := s.QueryUser(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var req DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user.Password != "" && !checkPasswordHash(req.Password, user.Password) {
		http.Error(w, "Invalid password", http.StatusForbidden)
		return
	}

	if err := s.lastAdminGuard(user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	keys, err := s.deleteUserData(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("delete account %d: %v", userID, err)
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}
	s.removeFileBlobs(r.Context(), keys)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Account deleted",
		"user_id": userID,
	})
}

// DELETE /api/users/{id}
// Admin-only account deletion for any user.
func (s *Handler) DeleteUserRoute(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		return
	}
	target, err := s.QueryUser(targetID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := s.lastAdminGuard(target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	keys, err := s.deleteUserData(targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("admin delete user %d: %v", targetID, err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	s.removeFileBlobs(r.Context(), keys)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User deleted",
		"user_id": targetID,
	})
}
