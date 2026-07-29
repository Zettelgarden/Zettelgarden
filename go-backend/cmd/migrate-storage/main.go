// Command migrate-storage is the one-time ETL that copies existing file blobs
// (and their thumbnails) from a Backblaze B2 bucket into local on-disk storage
// under STORAGE_DIR. It is Phase 4 of the S3/B2 → local-file-storage migration
// (design decision D6).
//
// The tool has NO aws-sdk-go dependency: each object is fetched with a raw
// SigV4-signed GET (see sign.go) implemented on top of net/http + crypto/hmac.
// Because of this it builds from main at any time — including after Phase 1
// removed aws-sdk-go-v2 — so it does not need a short-lived branch or an SDK
// re-add.
//
// It reads the object keys straight from the files table
// (path = "{userID}/{uuid}"; thumbnail_path = "{key}_thumb.jpg") and writes
// each to {STORAGE_DIR}/{key} via the production storage.LocalStore, reusing
// the atomic-write and path-traversal guard from the app's storage layer. The
// run is idempotent: an object whose local file already exists is skipped, so
// the tool can be safely re-run to resume after an interruption. Verify with
// the printed counts (expected vs downloaded vs skipped vs failed).
//
// Run BEFORE the storage cutover (Phase 5), pointed at the same STORAGE_DIR the
// server will use afterwards. Keep the B2 bucket read-only for a rollback
// window, then delete it (see follow-up Zettelgarden-yar.4).
//
// Usage:
//
//	migrate-storage [--bucket NAME] [--endpoint URL] [--region R]
//	                [--storage-dir DIR] [--db-driver sqlite|postgres]
//	                [--sqlite-path PATH] [--limit N] [--dry-run]
//	                [--timeout DURATION] [--no-thumbnails]
//
// Source B2 credentials come from B2_ACCESS_KEY_ID / B2_SECRET_ACCESS_KEY (the
// same env the app used pre-migration). DB_DRIVER/SQLITE_PATH/DB_* select the
// metadata database; STORAGE_DIR (default ./data/files) is the destination.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"go-backend/server"
	"go-backend/services/storage"
)

const (
	defaultEndpoint = "https://s3.us-east-005.backblazeb2.com"
	defaultRegion   = "us-east-005" // B2 S3-API signing region (matches the old SDK config)
)

// sourceConfig identifies the B2 bucket to copy objects from.
type sourceConfig struct {
	endpoint  string
	region    string
	bucket    string
	accessKey string
	secretKey string
}

// migrateOptions tunes a run (mostly for testing/debugging).
type migrateOptions struct {
	limit        int
	dryRun       bool
	noThumbnails bool
	timeout      time.Duration
}

// summary tallies the outcome of a run. expected counts primary objects plus
// thumbnail objects that were eligible (non-empty thumbnail_path).
type summary struct {
	filesSeen                                  int
	expected                                   int
	primaryDownloaded, primarySkipped          int
	primaryFailed                              int
	thumbDownloaded, thumbSkipped, thumbFailed int
}

func (s summary) failed() int {
	return s.primaryFailed + s.thumbFailed
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	bucket := flag.String("bucket", os.Getenv("B2_BUCKET_NAME"),
		"B2 bucket name to copy from (default: $B2_BUCKET_NAME)")
	endpoint := flag.String("endpoint", defaultEndpoint, "B2 S3-compatible endpoint URL")
	region := flag.String("region", defaultRegion, "SigV4 signing region (B2 uses us-east-005)")
	storageDir := flag.String("storage-dir", envOr("STORAGE_DIR", "./data/files"),
		"destination local storage root (default: $STORAGE_DIR or ./data/files)")
	dbDriver := flag.String("db-driver", envOr("DB_DRIVER", "sqlite"),
		"metadata DB driver: sqlite (default) or postgres")
	sqlitePath := flag.String("sqlite-path", envOr("SQLITE_PATH", "./data/zettelgarden.db"),
		"SQLite DB path when --db-driver=sqlite (default: $SQLITE_PATH)")
	limit := flag.Int("limit", 0, "process at most N files (0 = all; for debugging)")
	dryRun := flag.Bool("dry-run", false, "log what would be downloaded without writing or fetching")
	noThumbnails := flag.Bool("no-thumbnails", false, "skip thumbnail objects (only copy primary files)")
	timeout := flag.Duration("timeout", 5*time.Minute, "per-object GET timeout (large files may need more)")
	flag.Parse()

	src := sourceConfig{
		endpoint:  *endpoint,
		region:    *region,
		bucket:    *bucket,
		accessKey: os.Getenv("B2_ACCESS_KEY_ID"),
		secretKey: os.Getenv("B2_SECRET_ACCESS_KEY"),
	}
	if src.bucket == "" || src.accessKey == "" || src.secretKey == "" {
		log.Fatalf("missing source config: set B2_BUCKET_NAME (or --bucket), B2_ACCESS_KEY_ID, B2_SECRET_ACCESS_KEY")
	}
	if *dryRun {
		log.Printf("DRY RUN — no objects will be fetched or written")
	}

	ctx := context.Background()
	db, err := openDB(*dbDriver, *sqlitePath)
	if err != nil {
		log.Fatalf("open db (driver=%s): %v", *dbDriver, err)
	}
	defer db.Close()

	store, err := storage.NewLocalStore(*storageDir)
	if err != nil {
		log.Fatalf("open storage dir %q: %v", *storageDir, err)
	}

	log.Printf("source: b2 bucket=%q endpoint=%q region=%q", src.bucket, src.endpoint, src.region)
	log.Printf("destination: %s", store.Dir())
	log.Printf("metadata db: driver=%s", *dbDriver)

	sum, err := runMigration(ctx, db, store, &http.Client{Timeout: 0}, src, migrateOptions{
		limit:        *limit,
		dryRun:       *dryRun,
		noThumbnails: *noThumbnails,
		timeout:      *timeout,
	})
	if err != nil {
		log.Fatalf("migration aborted: %v", err)
	}

	log.Printf("────────── migration summary ──────────")
	log.Printf("files seen:            %d", sum.filesSeen)
	log.Printf("expected objects:      %d (files + thumbnails)", sum.expected)
	log.Printf("primary  downloaded:   %d   skipped: %d   failed: %d", sum.primaryDownloaded, sum.primarySkipped, sum.primaryFailed)
	log.Printf("thumb    downloaded:   %d   skipped: %d   failed: %d", sum.thumbDownloaded, sum.thumbSkipped, sum.thumbFailed)
	log.Printf("destination dir:       %s", store.Dir())
	if sum.failed() > 0 {
		log.Printf("FAILED objects:        %d — re-run to resume (idempotent); investigate failures above", sum.failed())
		os.Exit(1)
	}
	log.Printf("done — no failures. Verify expected(%d) == downloaded+skipped, then proceed to Phase 5 cutover.", sum.expected)
}

// runMigration copies every non-deleted file's primary object (and its
// thumbnail, unless opts.noThumbnails) from B2 into the local store. It is the
// testable core of the tool; main() wires flags/env into its arguments.
func runMigration(ctx context.Context, db *sql.DB, store *storage.LocalStore, client *http.Client,
	src sourceConfig, opts migrateOptions) (summary, error) {
	var sum summary

	rows, err := db.QueryContext(ctx, `
		SELECT id, path, thumbnail_path
		FROM files
		WHERE COALESCE(is_deleted, false) = false
		  AND path IS NOT NULL AND path <> ''
		ORDER BY id`)
	if err != nil {
		return sum, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if opts.limit > 0 && sum.filesSeen >= opts.limit {
			break
		}
		var (
			id            int64
			path          sql.NullString
			thumbnailPath sql.NullString
		)
		if err := rows.Scan(&id, &path, &thumbnailPath); err != nil {
			return sum, fmt.Errorf("scan files row: %w", err)
		}
		sum.filesSeen++
		sum.expected++

		switch st, err := downloadObject(ctx, client, store, src, path.String, opts); {
		case err != nil:
			sum.primaryFailed++
			log.Printf("FAIL  file  id=%d key=%q: %v", id, path.String, err)
		case st == statusSkipped:
			sum.primarySkipped++
		default:
			sum.primaryDownloaded++
		}

		if !opts.noThumbnails && thumbnailPath.Valid && thumbnailPath.String != "" {
			sum.expected++
			switch st, err := downloadObject(ctx, client, store, src, thumbnailPath.String, opts); {
			case err != nil:
				sum.thumbFailed++
				log.Printf("FAIL  thumb id=%d key=%q: %v", id, thumbnailPath.String, err)
			case st == statusSkipped:
				sum.thumbSkipped++
			default:
				sum.thumbDownloaded++
			}
		}

		if sum.filesSeen%50 == 0 {
			log.Printf("progress: %d files seen", sum.filesSeen)
		}
	}
	if err := rows.Err(); err != nil {
		return sum, fmt.Errorf("iterating files: %w", err)
	}
	return sum, nil
}

const (
	statusDownloaded = "downloaded"
	statusSkipped    = "skipped"
)

// downloadObject fetches a single object from B2 into the local store. It is
// idempotent: if the local file already exists it is skipped without fetching.
// The returned status is one of the status* constants.
func downloadObject(ctx context.Context, client *http.Client, store *storage.LocalStore,
	src sourceConfig, key string, opts migrateOptions) (string, error) {
	exists, err := store.Exists(ctx, key)
	if err != nil {
		return "", fmt.Errorf("stat local %q: %w", key, err)
	}
	if exists {
		return statusSkipped, nil
	}
	if opts.dryRun {
		log.Printf("  would download %q", key)
		return statusDownloaded, nil
	}

	escapedPath := escapeS3Path("/" + src.bucket + "/" + key)
	rawURL := src.endpoint + escapedPath
	reqCtx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request for %q: %w", key, err)
	}
	signGET(req, src.region, src.accessKey, src.secretKey, time.Now().UTC().Format("20060102T150405Z"))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %q: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("GET %q: HTTP %d: %s", key, resp.StatusCode, snippet)
	}
	// Stream straight into the store; LocalStore.Upload is atomic (temp + rename)
	// so a network interruption cannot leave a partial object.
	if err := store.Upload(ctx, key, resp.Body); err != nil {
		return "", fmt.Errorf("store %q: %w", key, err)
	}
	return statusDownloaded, nil
}

// openDB opens the metadata database (sqlite or postgres) that holds the
// files.path / files.thumbnail_path keys.
func openDB(driver, sqlitePath string) (*sql.DB, error) {
	switch driver {
	case "sqlite":
		return server.OpenSQLite(sqlitePath)
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOr("DB_HOST", "localhost"),
			envOr("DB_PORT", "5432"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASS"),
			envOr("DB_NAME", os.Getenv("DB_USER")),
		)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, err
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping (dsn hidden): %w", err)
		}
		db.SetMaxOpenConns(4)
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported db driver %q (use sqlite or postgres)", driver)
	}
}

func envOr(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}
