# S3 (Backblaze B2) → Local File Storage Migration Plan

**Date:** 2026-07-29
**Status:** Decisions locked — peer-reviewed, ready for implementation
**Author:** Nick + Pi

## Revision history

- **v1 (2026-07-29):** Initial draft, verified against the live tree.
- **v2 (2026-07-29):** Open questions resolved by owner: production B2 data
  **must** be migrated (Phase 4 runs); `STORAGE_DIR` defaults to `./data/files`;
  backups handled at the system level (out of scope for this plan). D1 adopted
  as recommended (`FileStore` interface).
- **v3 (2026-07-29):** Peer-review corrections (verified against the live
  tree): fixed D7 to reuse the existing `/usr/src/app/data` mount instead of
  the unmounted `/app/data/files`; committed D6 to an SDK-free SigV4 HTTP
  reader, which dissolves the Phase 1 (removes SDK) vs Phase 4 (needs B2)
  contradiction; added `LocalStore` write invariants to D1; expanded D8 to
  cover the two remaining `Server.Testing` short-circuits in `files.go`; added
  a `Content-Disposition` fix-while-here to Phase 1; flagged real secrets
  committed in `.env-bash`.

## Overview

Migrate uploaded-file storage from **Backblaze B2** (accessed through the AWS
SDK for Go v2) to **local on-disk storage**. This continues the operational
simplification that motivated the Postgres → SQLite migration (see
`2026-07-17-postgres-to-sqlite-migration-design.md`): collapse toward a single
self-hosted binary with no external object-storage dependency, no B2 account,
and no per-GB egress costs.

The good news from the inventory below: **the frontend never touches B2.** All
file I/O flows through two REST routes (`/api/files/upload`,
`/api/files/download/{id}`) that stream through the Go backend. B2 is a
backend-internal detail, so this is a clean abstraction swap with **no schema
change** and **no frontend change**.

## Goal

- Uploaded files (and their thumbnails) persist to a directory on the server's
  local disk instead of a remote bucket.
- Zero behavior change for the end user (upload, download, thumbnail, delete,
  epub import all keep working).
- Remove the `aws-sdk-go-v2` dependency and the `B2_*` environment variables.
- Existing Go test suite passes against local storage.

## Non-Goals

- Building a generic multi-backend storage layer with pluggable providers.
  (See D1 — we *may* keep a thin interface, but the only production
  implementation will be local disk.)
- Preserving the `listObjects` helper — it is **dead code** (defined in
  `handlers/s3.go:69`, never called from any route or test). It will be deleted.
- Preserving the `TestInspector.FilesUploaded` counter — it is mutated in
  `s3.go` but **never asserted in any test**. It will be removed (see D8).
- Streaming-range / resumable-upload features B2 didn't give us either.

## Current-State Inventory (verified 2026-07-29)

| Area | Finding | Migration impact |
|---|---|---|
| Provider | Backblaze B2, hardcoded endpoint `https://s3.us-east-005.backblazeb2.com` (`s3.go:17`) | Eliminated |
| SDK | `github.com/aws/aws-sdk-go-v2/{config,credentials,service/s3}` — **4 direct** + ~**13 transitive** modules (`go.mod`) | All removed |
| Frontend | Talks only to `/api/files/upload` (multipart POST) and `/api/files/download/{id}` (binary stream). **No presigned URLs, no direct-to-B2 uploads.** (`zettelkasten-front/src/api/files.ts`) | **No change** |
| Schema | `files.path` and `files.filename` store the **object key** (`{userID}/{uuid}`); `files.thumbnail_path` stores `{key}_thumb.jpg`. Provider-agnostic. | **No change** — keys are reused as relative paths |
| S3 client lifecycle | `Server.S3 *s3.Client` (`server/server.go:14`); created once in `main.go:131` via `h.CreateS3Client()` | Replaced by a store |
| Call sites of S3 ops | `uploadObject`: `files.go:589` (upload), `files.go:101` (thumbnail); `downloadObject`: `files.go:679` (download route), `epub.go:168` (epub import); `deleteObject`: `files.go:716` (delete route) | 5 sites to repoint |
| `UploadObject` (public) | Only called via dead `handlerS3Uploader` in `main.go:34` (references a non-existent `jobs.S3Uploader` interface — **already dead code**) | Delete the wrapper |
| Text extraction | `services/llmprocessor.go:processFileTextExtractionJob` reads `s3Key` from DB but **never downloads** — explicit `// TODO: Download file from S3` stub | Store interface unblocks this (separate task) |
| Config | `S3Config{AccessKeyID, SecretAccessKey, BucketName}` from `B2_ACCESS_KEY_ID`, `B2_SECRET_ACCESS_KEY`, `B2_BUCKET_NAME` (`pkg/config/services.go`) | Replaced by `STORAGE_DIR` |
| `.env.example` | Lists stale `S3_*` vars that nothing reads; real vars are `B2_*` | Clean up both |
| Testing | `Server.Testing == true` short-circuits every S3 op to a no-op (client is `nil`); `TestInspector.FilesUploaded` incremented/decremented but never asserted | Replaced by real local store on tempdir (D8) |
| Deployment | `docker-zettel-run.yml` mounts `/var/log/zettel`; no storage volume exists | Add a storage volume |

### Call graph (the entire surface to change)

```
UploadFileRoute      ──uploadObject──► B2   (files.go:589)
generateAndUploadThumb ──uploadObject──► B2 (files.go:101)
DownloadFileRoute    ──downloadObject──► B2 (files.go:679)
ImportEpubRoute      ──downloadObject──► B2 (epub.go:168)
DeleteFileRoute      ──deleteObject───► B2  (files.go:716)
```

That's it. Five call sites, three operations.

---

## Design Decisions

### D1 — Introduce a `FileStore` interface (RECOMMENDED) vs. inline `os` calls

**Recommended:** define a small interface and have `Server` hold it.

```go
// services/storage/store.go (new)
package storage

type Store interface {
    Upload(ctx context.Context, key string, r io.Reader) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

The only production implementation is `LocalStore`, rooted at a configurable
directory. Tests inject a `LocalStore` rooted at `t.TempDir()` (or a trivial
in-memory fake).

**`LocalStore` write invariants — specify these explicitly; they're the real
correctness concerns B2 gave us for free and local disk does not:**

- `Upload` calls `os.MkdirAll` on the key's parent dir (keys contain `/`,
  e.g. `42/<uuid>`), mode `0750`.
- `Upload` writes to a temp file **in the same directory** then `os.Rename`s
  it into place, so a crash mid-write cannot leave a partial object. File mode
  `0640`.
- `Download` returns an `*os.File` (satisfies `io.ReadCloser`); the caller
  closes it.
- `Delete` is tolerant of a missing file — B2's `DeleteObject` is idempotent
  and we keep that property (return `nil` for `fs.ErrNotExist`).

**Why an interface when we only have one impl:**
1. It removes the `Server.Testing` special-casing scattered through `s3.go` —
   tests just get a real store on a tempdir, which is *more* correct (it
   actually exercises the disk paths) and *less* code.
2. It makes the five call sites mockable for unit tests without DB round-trips.
3. It unblocks `processFileTextExtractionJob`, which needs a `Download` to do
   its job and currently can't get one cleanly.
4. Cost is ~30 lines. If we never add a second backend, nothing is lost.

**Alternative (rejected):** replace each S3 call inline with `os` calls and
keep the `Server.Testing` flag. Less new code, but keeps the test-mode
fork-lift and leaves text extraction still unable to fetch files. Only choose
this if you want the absolute minimum diff. (Decision locked in D1-yes — see
"Resolved decisions" below.)

### D2 — Reuse the existing key as a relative path (no schema change)

The current key is `{userID}/{uuid}` and thumbnails are `{key}_thumb.jpg`.
These are already safe relative path segments (userID is an int, uuid is hex).
The `LocalStore` maps a key to `{baseDir}/{key}` verbatim.

- **No migration of `files.path` / `files.filename` / `files.thumbnail_path`
  values is needed** — the stored strings work as relative paths unchanged.
- Add a defensive `path.Clean` + reject-any-`..`/absolute check in `Upload`/
  `Download`/`Delete` even though keys are server-generated, to harden against
  future callers.

### D3 — Config: replace `S3Config` with `StorageConfig`

```go
// pkg/config/services.go
type StorageConfig struct {
    Dir string // absolute or relative path to the file root
}
```

- Env var: `STORAGE_DIR`, default `./data/files` (sibling of `SQLITE_PATH`).
- `loadStorageConfig()` creates the dir (mode 0750) at load time and fails
  validation if it can't be stat'd/written.
- Remove `S3Config`, `loadS3Config()`, and the `B2_*` validation block.
- Update `.env.example`: drop the stale `S3_*` block and the `B2_*` block; add
  `STORAGE_DIR=./data/files`.
- Update `.env-bash` and `conftest.go`'s `setEnvIfNotSet("B2_*", …)` lines →
  `setEnvIfNotSet("STORAGE_DIR", t.TempDir() equivalent)`.

### D4 — Drop the AWS SDK

After the swap, `go mod tidy` removes all 17 `aws-sdk-go-v2` modules. Verify
with `grep -rn "aws" go.*` returning nothing. This is a meaningful dependency
shrink and removes the `aws` import from `server/server.go` and `main.go`.

### D5 — Thumbnails keep the `_thumb.jpg` suffix

No change to `generateAndUploadThumbnail` beyond calling `Store.Upload` with an
`os.File` reader instead of `uploadObject(client, key, path)`. The temp-file
lifecycle (write thumb → upload → `os.Remove`) stays the same.

> **Path-vs-reader tradeoff (conscious):** the current callers pass temp-file
> *paths*; the new `io.Reader` signature means we open-and-stream. For local
> disk a path-based `Upload` could have been a cheap `os.Rename` /
> `copy_file_range`. We accept the streaming shape deliberately for interface
> symmetry — noted here so nobody later "optimizes" the signature back thinking
> it was accidental.

### D6 — One-time data migration (existing B2 objects → local disk)

For the production instance with real uploads already in B2, a one-time ETL is
required (parallels the Postgres→SQLite ETL pattern). Sketch:

1. Add `go-backend/cmd/migrate-storage/main.go` — a **standalone tool with NO
   AWS SDK dependency**. It reads the key list from the DB and, for each key,
   issues a raw SigV4-signed
   `GET https://s3.us-east-005.backblazeb2.com/<bucket>/<key>` (~80 lines with
   `crypto/hmac` + `net/http`; no new modules). This is deliberate: because the
   tool needs no SDK, it can be built and run from `main` at *any* time —
   including after Phase 1 has removed `aws-sdk-go-v2`. No short-lived branch,
   no SDK re-add. (This dissolves the v2 phase contradiction where Phase 1
   removed the SDK that Phase 4 depended on.)
2. `SELECT id, path, thumbnail_path FROM files WHERE is_deleted = FALSE` → for
   each row, `GET` the object from B2 and write to `{STORAGE_DIR}/{key}`.
3. Run **before** the cutover deploy, with an idempotent "skip if local file
   exists" guard so it can be re-run.
4. Verify counts: `count(files)` == `count(local files)` (minus deleted).
5. Keep B2 read-only for a rollback window, then delete the bucket.

If there is little/no production data you care about, this phase can be skipped
entirely.

### D7 — Deployment: reuse the existing data volume

The `go_backend` service in `docker-zettel-run.yml` already mounts
`./data:/usr/src/app/data` (it carries the SQLite file today). **Reuse that
same volume** for file blobs rather than adding a second mount — one backup
target, one host dir to reason about:

```yaml
    volumes:
      - /var/log/zettel:/app/logs
      - ./data:/usr/src/app/data      # existing — now also holds files/
```

Set `STORAGE_DIR=/usr/src/app/data/files` in `zettel.env` for the container.
**Note the `/usr/src/app/...` prefix** — it matches the existing mount. The
v2 draft's `/app/data/files` pointed at an unmounted path and would have
silently written inside the container's ephemeral filesystem. The bare-metal
default (`./data/files`, see D3) is a *relative* path for non-containerized
dev; the container override is the absolute path matching its mount.

The host-side `./data` dir is what the owner's backup regime must cover. The
SQLite plan's posture (`VACUUM INTO` for the db, `rsync`/`restic` for the dir)
extends naturally to the new `files/` subtree.

### D8 — Testing cleanup

- Delete the `if s.Server.Testing || client == nil` blocks from the store
  methods (they no longer exist in the new code).
- Delete `TestInspector.FilesUploaded` (never asserted).
- **Two more `Server.Testing` short-circuits live outside `s3.go`** and must be
  decided explicitly (v2 missed them):
  - `files.go:686` — the download route early-returns in testing so no bytes
    are streamed. **Remove it.** With a real tempdir store the route can stream
    real bytes, which is exactly what lets `TestDownloadFile` assert content.
  - `files.go:636` — skips thumbnail generation in testing. **Keep this one,
    but for the right reason:** it's about the fire-and-forget goroutine
    outliving the test transaction, *not* about storage. Leave a comment saying
    so, so a future reader doesn't delete it thinking the store made it moot.
- `conftest.go` sets `STORAGE_DIR` to a per-test temp dir; tests that exercise
  upload/download now read/write real (temp) files. `uploadTestFile` in
  `files_test.go` becomes a real round-trip — `TestDownloadFile` can assert the
  bytes match `tests/test.txt`.

---

## Phased Implementation

### Phase 0 — Spike (½ day)
- Stand up `services/storage/store.go` + `LocalStore` with the three methods
  and a tempdir-based unit test (`store_test.go`) covering upload→download→
  delete round-trip and the path-traversal guard.
- Confirm the interface shape feels right before touching call sites.

### Phase 1 — Wire the store through (1 day)
1. Add `StorageConfig` (D3); load + validate in `loadServiceConfig`.
2. Add `Server.Store storage.Store`; initialize in `main.go` from `STORAGE_DIR`.
3. Rewrite the five call sites (`files.go` x4, `epub.go` x1) to use
   `s.Server.Store` instead of `s.Server.S3` + the `uploadObject`/
   `downloadObject`/`deleteObject` helpers. Two **fix-while-here** items in the
   download route (`files.go:679`): (a) move `Content-Disposition` above
   `io.Copy` — it's currently set *after* the copy, so the header is never
   actually sent; (b) drop the `if s.Server.Testing { return }` at
   `files.go:686` (see D8). The epub `downloadEpubToTemp` call site becomes a
   redundant local-file→tempfile copy post-migration — leave a
   `// TODO: read directly from the stored path` rather than refactoring now.
4. Delete `handlers/s3.go` entirely, the `handlerS3Uploader` dead code in
   `main.go`, and `TestInspector`.
5. `go mod tidy`; confirm AWS modules gone. (The Phase 4 migration tool is
   SDK-free by design — see D6 — so removing the SDK here does not strand it.)

### Phase 2 — Config & deploy artifacts (½ day)
- Update `.env.example`, `.env-bash`, `conftest.go` (D3, D8).
- Update `docker-zettel-run.yml` + `zettel.env` (D7).
- Update `Dockerfile` only if `STORAGE_DIR` needs a default `ENV`.
- **Security (timely — file a follow-up issue):** `go-backend/.env-bash` has
  real-looking B2 / LLM / Stripe secrets committed to the repo. Since the B2
  keys are being decommissioned anyway, rotate them now; move the surviving
  secrets out of git and scrub history if practical.

### Phase 3 — Tests green (½ day)
- Re-run `go test ./...`; fix the `files_test.go`/`setup_test.go` adjustments
  from D8. `TestDownloadFile` now asserts real bytes.
- Manually exercise upload → thumbnail → download → epub import → delete
  locally.

### Phase 4 — Data migration (D6) — confirmed required, runs before cutover
- Build & run `cmd/migrate-storage` (SDK-free per D6) on the production host,
  writing into the new `STORAGE_DIR`. It can be built from `main` at any time
  because it pulls in no AWS modules.
- Verify object counts (`count(files)` ≈ local file count, minus deleted);
  leave B2 read-only for the rollback window.

### Phase 5 — Cutover & cleanup
- Deploy the new binary; point `STORAGE_DIR` at the populated dir.
- Smoke-test a real upload and download.
- After a green window (e.g. 1 week), delete the B2 bucket and revoke keys.

---

## Rollback

- Pre-cutover: the old binary + B2 keys still work; revert is a redeploy.
- Post-cutover (Phase 5): local files are the source of truth. To roll back to
  B2 you'd need to re-upload local files to B2 — expensive and not recommended.
  Prefer fixing forward. Keep B2 read-only until you're confident (D6).

## Risk register

| Risk | Likelihood | Mitigation |
|---|---|---|
| Disk fills up (no B2 "infinite" bucket) | Medium | Already have per-user `max_file_storage` quota (`files.go`); add disk-space monitoring/alerting in D7 |
| Lost files if host disk dies | Medium | Backup regime for `STORAGE_DIR` (D7) — same posture as the SQLite file |
| Egress/perf regression on download | Low | Files are served via the Go process either way; local disk is faster than B2 |
| Path traversal via crafted key | Low | Keys are server-generated (`userID/uuid`); defensive check in D2 |
| Multi-instance horizontal scaling breaks | Low (self-hosted, single-binary target) | Out of scope per non-goals; would need shared FS or S3 again |

## Resolved decisions (owner, 2026-07-29)

1. **D1:** Adopt the `FileStore` interface as recommended.
2. **D6:** Production B2 data **must** be migrated — Phase 4 runs.
3. **`STORAGE_DIR`:** `./data/files` (next to the SQLite db).
4. **Backups:** handled at the system level by the owner; this plan only
   documents the volume mount and does not prescribe a backup tool.
