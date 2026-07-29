package main

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go-backend/server"
	"go-backend/services/storage"
)

// newFakeB2 returns an httptest server that emulates a B2 S3 endpoint for a
// fixed bucket: it verifies the incoming SigV4 Authorization header (re-derived
// via buildAuthorization from the same creds/region — a self-consistency check
// that the wire request matches what was signed) and serves object bytes from
// the objects map keyed by "{bucket}/{key}".
func newFakeB2(t *testing.T, bucket, accessKey, secretKey, region string, objects map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		amzDate := r.Header.Get("x-amz-date")
		wantAuth := buildAuthorization(http.MethodGet, r.URL.EscapedPath(), "", r.Host,
			region, "s3", accessKey, secretKey, amzDate, nil)
		if r.Header.Get("Authorization") != wantAuth {
			http.Error(w, "bad signature", http.StatusForbidden)
			return
		}
		body, ok := objects[r.URL.Path[1:]] // strip leading '/'
		if !ok {
			http.Error(w, "no such key", http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, body)
	})
	return httptest.NewServer(mux)
}

// setupFilesDB creates a temp SQLite DB with a minimal files table and the
// given rows (path, thumbnail_path, is_deleted).
func setupFilesDB(t *testing.T, rows [][3]any) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := server.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT,
		thumbnail_path TEXT,
		is_deleted BOOLEAN DEFAULT false
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO files (path, thumbnail_path, is_deleted) VALUES (?, ?, ?)`,
			r[0], r[1], r[2]); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	return db
}

func TestRunMigration(t *testing.T) {
	const (
		bucket    = "zettelgarden-test"
		accessKey = "AKIDTEST"
		secretKey = "testsecret"
		region    = "us-east-005"
	)
	// The objects B2 holds. Includes one object that exists in B2 but is
	// referenced by a deleted row (it must NOT be copied).
	objects := map[string]string{
		bucket + "/42/aaa":           "AAA-CONTENT",
		bucket + "/42/aaa_thumb.jpg": "THUMB-AAA",
		bucket + "/42/bbb":           "BBB",
		bucket + "/42/ccc":           "CCC-DELETED-SHOULD-NOT-COPY",
		bucket + "/42/ccc_thumb.jpg": "CCC-THUMB-SHOULD-NOT-COPY",
	}
	srv := newFakeB2(t, bucket, accessKey, secretKey, region, objects)
	t.Cleanup(srv.Close)

	db := setupFilesDB(t, [][3]any{
		{"42/aaa", "42/aaa_thumb.jpg", false}, // active, with thumbnail
		{"42/bbb", "", false},                 // active, no thumbnail
		{"42/ccc", "42/ccc_thumb.jpg", true},  // DELETED → skipped entirely
		{"", "", false},                       // empty path → filtered by query
	})

	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	src := sourceConfig{endpoint: srv.URL, region: region, bucket: bucket, accessKey: accessKey, secretKey: secretKey}
	opts := migrateOptions{timeout: 10 * time.Second}

	// First run: two active primary objects + one thumbnail copied.
	sum, err := runMigration(context.Background(), db, store, http.DefaultClient, src, opts)
	if err != nil {
		t.Fatalf("runMigration: %v", err)
	}
	if sum.filesSeen != 2 {
		t.Errorf("filesSeen = %d, want 2 (deleted + empty-path rows excluded)", sum.filesSeen)
	}
	if sum.expected != 3 { // 2 primaries + 1 thumbnail
		t.Errorf("expected = %d, want 3", sum.expected)
	}
	if sum.primaryDownloaded != 2 || sum.thumbDownloaded != 1 || sum.failed() != 0 {
		t.Errorf("first run counts = %+v, want {primaryDownloaded:2, thumbDownloaded:1, failed:0}", sum)
	}

	// Content checks: copied objects match B2 bytes; deleted row's object absent.
	wantBytes := map[string]string{
		"42/aaa":           "AAA-CONTENT",
		"42/aaa_thumb.jpg": "THUMB-AAA",
		"42/bbb":           "BBB",
	}
	ctx := context.Background()
	for key, want := range wantBytes {
		rc, err := store.Download(ctx, key)
		if err != nil {
			t.Errorf("Download %q: %v", key, err)
			continue
		}
		got, _ := io.ReadAll(rc)
		rc.Close()
		if string(got) != want {
			t.Errorf("content %q = %q, want %q", key, got, want)
		}
	}
	if ok, _ := store.Exists(ctx, "42/ccc"); ok {
		t.Errorf("deleted row's object 42/ccc was copied; it must be skipped")
	}

	// Second run: idempotent — everything already exists, all skipped.
	sum2, err := runMigration(context.Background(), db, store, http.DefaultClient, src, opts)
	if err != nil {
		t.Fatalf("runMigration (2nd): %v", err)
	}
	if sum2.primaryDownloaded != 0 || sum2.thumbDownloaded != 0 || sum2.failed() != 0 {
		t.Errorf("second run copied objects = %+v, want all skipped", sum2)
	}
	if sum2.primarySkipped != 2 || sum2.thumbSkipped != 1 {
		t.Errorf("second run skips = %+v, want primarySkipped:2, thumbSkipped:1", sum2)
	}
}

// TestRunMigrationDryRun confirms --dry-run fetches nothing and writes nothing.
func TestRunMigrationDryRun(t *testing.T) {
	const bucket, accessKey, secretKey, region = "b", "a", "s", "us-east-005"
	objects := map[string]string{bucket + "/9/zzz": "ZZZ"}
	srv := newFakeB2(t, bucket, accessKey, secretKey, region, objects)
	t.Cleanup(srv.Close)

	db := setupFilesDB(t, [][3]any{{"9/zzz", "", false}})
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	src := sourceConfig{endpoint: srv.URL, region: region, bucket: bucket, accessKey: accessKey, secretKey: secretKey}

	sum, err := runMigration(context.Background(), db, store, http.DefaultClient, src, migrateOptions{dryRun: true, timeout: time.Second})
	if err != nil {
		t.Fatalf("runMigration: %v", err)
	}
	if sum.primaryDownloaded != 1 { // dry-run reports would-download as downloaded in the tally
		t.Errorf("dry-run primaryDownloaded = %d, want 1", sum.primaryDownloaded)
	}
	if ok, _ := store.Exists(context.Background(), "9/zzz"); ok {
		t.Errorf("dry-run wrote an object; it must not")
	}
}

// TestRunMigrationServerFailure confirms a B2 404 is classified as not-in-source
// (an orphaned row), NOT a failure — so the run completes and can be re-run.
func TestRunMigrationServerFailure(t *testing.T) {
	const bucket, accessKey, secretKey, region = "b", "a", "s", "us-east-005"
	// Empty object map → every GET 404s.
	srv := newFakeB2(t, bucket, accessKey, secretKey, region, map[string]string{})
	t.Cleanup(srv.Close)

	db := setupFilesDB(t, [][3]any{{"5/missing", "", false}})
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	src := sourceConfig{endpoint: srv.URL, region: region, bucket: bucket, accessKey: accessKey, secretKey: secretKey}

	sum, err := runMigration(context.Background(), db, store, http.DefaultClient, src, migrateOptions{timeout: time.Second})
	if err != nil {
		t.Fatalf("runMigration should not return an error for object failures: %v", err)
	}
	if sum.primaryNotInSource != 1 {
		t.Errorf("primaryNotInSource = %d, want 1 (404 should be not-in-source, not failed)", sum.primaryNotInSource)
	}
	if sum.failed() != 0 {
		t.Errorf("failed = %d, want 0 (a 404 is not a failure)", sum.failed())
	}
}

// TestRunMigrationLegacyAbsoluteKey confirms a legacy absolute-path key (the
// pre-B2 `/usr/src/app/files/<uuid>` scheme found in production) is classified
// as not-in-source rather than crashing the run.
func TestRunMigrationLegacyAbsoluteKey(t *testing.T) {
	const bucket, accessKey, secretKey, region = "b", "a", "s", "us-east-005"
	srv := newFakeB2(t, bucket, accessKey, secretKey, region, map[string]string{})
	t.Cleanup(srv.Close)

	db := setupFilesDB(t, [][3]any{
		{"/usr/src/app/files/legacy-uuid.pdf", "", false}, // legacy absolute path
		{"7/normal-key", "", false},                       // normal key (not in fake B2 → 404 → not-in-source too)
	})
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	src := sourceConfig{endpoint: srv.URL, region: region, bucket: bucket, accessKey: accessKey, secretKey: secretKey}

	sum, err := runMigration(context.Background(), db, store, http.DefaultClient, src, migrateOptions{timeout: time.Second})
	if err != nil {
		t.Fatalf("runMigration: %v", err)
	}
	if sum.primaryNotInSource != 2 {
		t.Errorf("primaryNotInSource = %d, want 2 (legacy invalid key + 404)", sum.primaryNotInSource)
	}
	if sum.failed() != 0 {
		t.Errorf("failed = %d, want 0", sum.failed())
	}
}
