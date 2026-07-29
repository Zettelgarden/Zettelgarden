# S3 (Backblaze B2) → Local File Storage Migration — Status

**Last updated:** 2026-07-29 (Phase 1 COMPLETE — store wired through all 5 call sites; AWS SDK + `s3.go` + dead code removed; full suite green)
**Plan:** [`2026-07-29-s3-to-local-file-storage-design.md`](./2026-07-29-s3-to-local-file-storage-design.md)
**Tracking:** epic `Zettelgarden-yar` · Phase 0 `Zettelgarden-yar.1` (closed) · Phase 1 `Zettelgarden-yar.2` (closed)

## TL;DR

**Phase 0 + Phase 1 are done.** The backend now stores uploads on local disk
via `services/storage.LocalStore`; the AWS SDK is gone and `handlers/s3.go`
is deleted.

- **Phase 0 (spike):** `services/storage/` ships the `Store` interface
  (`Upload`/`Download`/`Delete`) and `LocalStore` with atomic writes, `0750`/
  `0640` modes, idempotent delete, and a path-traversal guard (decisions **D1**/**D2**).
  8 unit tests green.
- **Phase 1 (wire-through):** `StorageConfig`/`STORAGE_DIR` replace `S3Config`/
  `B2_*` (**D3**); `Server.Store` replaces `Server.S3`; the 5 call sites
  (`files.go` ×4, `epub.go` ×1) use `s.Server.Store`; `handlers/s3.go`, dead
  `handlerS3Uploader`, and `TestInspector` are deleted; `go mod tidy` dropped
  all `aws-sdk-go-v2` modules (**D4** — 0 refs in `go.mod`/`go.sum`/source).
  Two fix-while-here items landed in the download route: `Content-Disposition`
  is set *before* `io.Copy` (was silently never sent), and the
  `Server.Testing` short-circuit is gone so routes stream real bytes (**D8**).

The full Go suite is green (16 packages). **No frontend change, no schema
change.** What remains is config/deploy artifact polish (Phase 2 — `.env.*`,
Docker, secret rotation), the one-time B2→local data ETL (Phase 4), and
cutover (Phase 5).

---

## Phase status

| Phase | Status | Notes |
|---|---|---|
| 0 — Spike (interface + LocalStore) | ✅ **Done** | `services/storage/store.go` + `store_test.go`. `Store{Upload,Download,Delete}` interface; `LocalStore` with atomic writes + traversal guard. 8 tests green. |
| 1 — Wire the store through | ✅ **Done** | `StorageConfig`/`STORAGE_DIR` (D3); `Server.Store` (D1); 5 call sites repointed; download `Content-Disposition` fix + `Testing` guard removal (D8); `handlers/s3.go` + dead `handlerS3Uploader` + `TestInspector` deleted; `go mod tidy` removed all AWS modules (D4). Full suite green. |
| 2 — Config & deploy artifacts | ⬜ Not started | `.env.example`/`.env-bash` (D3); `docker-zettel-run.yml` + `zettel.env` (D7). Secret-rotation follow-up. (conftest `STORAGE_DIR` env default already swapped in Phase 1.) |
| 3 — Tests green | ⬜ Not started | Re-run `go test ./...`; `TestDownloadFile` asserts real bytes; manual upload→thumb→download→epub→delete. |
| 4 — Data migration (B2 → local) | ⬜ Not started | SDK-free `cmd/migrate-storage` (D6); runs before cutover. |
| 5 — Cutover & cleanup | ⬜ Not started | Deploy; smoke; keep B2 read-only for rollback window; then delete bucket. |

---

## Phase 0 progress

### What landed

`go-backend/services/storage/store.go` (new package `storage`):

```go
type Store interface {
    Upload(ctx context.Context, key string, r io.Reader) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

**`LocalStore`** is the only production implementation, rooted at an absolute
dir (validated + created by `NewLocalStore`, which probes writability).

Write invariants from **D1**, all implemented and asserted by tests:

- `Upload` `os.MkdirAll`s the key's parent dir (keys contain `/`, e.g.
  `42/<uuid>`) at mode `0750`.
- `Upload` writes to a temp file **in the same directory** (`os.CreateTemp`)
  then `os.Rename`s it into place — a crash mid-write cannot leave a partial
  object. Final file mode `0640` (explicit `Chmod`, independent of umask).
- `Download` returns an `*os.File` (`io.ReadCloser`); the caller closes it. A
  missing object is wrapped (`%w`) so `errors.Is(err, fs.ErrNotExist)` works.
- `Delete` is idempotent — returns `nil` for a missing file, like B2's
  `DeleteObject`.

Path safety (**D2**): every method funnels through `resolve(key)`, which
`path.Clean`s the key and rejects empty / absolute / `..`-escaping keys, then
belt-and-suspenders confirms via `filepath.Rel` that the resolved path stays
within the storage root.

`go-backend/services/storage/store_test.go` — 8 tests:

| Test | Asserts |
|---|---|
| `TestRoundTrip` | Upload bytes → Download → content matches → Delete → Download now `fs.ErrNotExist`. |
| `TestDeleteMissingIsIdempotent` | `Delete` on a never-uploaded key returns nil (B2 parity). |
| `TestDownloadMissingWrapsErrNotExist` | Missing download errors and `errors.Is(err, fs.ErrNotExist)`. |
| `TestUploadCreatesNestedDirs` | Multi-segment key (`7/aaa/bbb/deep-key`) lands at the right nested path. |
| `TestUploadOverwrites` | Second `Upload` to the same key replaces the first (B2 `PutObject` parity). |
| `TestUploadFileModeAndAtomicPlacement` | File mode is exactly `0640`; no `.upload-*` temp files left behind. |
| `TestPathTraversalGuard` | Empty / `.` / `../x` / `42/../..` / `42/../../etc/passwd` / `/abs` / `../` all rejected on Upload/Download/Delete; a normal key still works afterward. |
| `TestNewLocalStoreCreatesMissingDir` | Constructor creates a missing nested dir. |

### Verification

```
$ go vet ./services/storage/...      # clean
$ go build ./...                      # clean (no call-site changes)
$ go test ./services/storage/... -v   # 8/8 PASS
$ gofmt -l services/storage/          # clean
```

### Interface-shape notes (confirmations for Phase 1)

- The three-method `Store` matches **D1** verbatim. Signature is
  `io.Reader`/`io.ReadCloser` (not file paths), per the conscious tradeoff in
  **D5** — current callers pass temp-file paths and will be opened/streamed.
- `*os.File` satisfies `io.ReadCloser` directly, so `Download` needs no
  adapter.
- The key→path mapping in `resolve` reuses the existing DB key verbatim
  (`{userID}/{uuid}`, thumbnails `{key}_thumb.jpg`) — confirming **D2**: no
  schema change, no key migration.
- `ctx` is accepted for symmetry but local I/O isn't interruptible; each
  method checks `ctx.Err()` up front (sufficient for the local-only impl).

No surprises. The interface is ready to wire through in Phase 1.

---

## Phase 1 progress

The store is wired through end to end. AWS SDK gone; full suite green.

### Config (D3)

- `pkg/config/services.go`: `S3Config`/`loadS3Config()` replaced by
  `StorageConfig{Dir}`/`loadStorageConfig()`. Env `STORAGE_DIR`, default
  `./data/files`. The loader `os.MkdirAll`s the root (`0750`) at load time and
  appends a validation error if it can't be created/written (`checkDirWritable`
  probe) — the one correctness property local disk doesn't give for free.
- `ServiceConfig.S3` → `ServiceConfig.Storage`.

### Server + main (D1)

- `server/server.go`: `S3 *s3.Client` → `Store storage.Store`; the `aws-sdk-go-v2/service/s3` import dropped; `TestInspector` field + struct removed (**D8**).
- `main.go`: the S3 init block is now `storage.NewLocalStore(cfg.Services.Storage.Dir)` (fatals on failure); `Server.Store` set. Dead
  `handlerS3Uploader` (referenced a non-existent `jobs.S3Uploader` interface —
  never instantiated) deleted.

### Call sites (5) — `files.go` ×4, `epub.go` ×1

| Site | Change |
|---|---|
| `generateAndUploadThumbnail` | opens the temp thumb, `Store.Upload(ctx, key, file)`, closes it. |
| `UploadFileRoute` | rewinds the temp file to start, `Store.Upload(r.Context(), s3Key, tempFile)`. |
| `DownloadFileRoute` | `Store.Download` → `defer rc.Close()`; **`Content-Disposition` moved above `io.Copy`** (was set after the copy → never sent); **`Server.Testing` short-circuit removed** so real bytes stream (**D8**). |
| `DeleteFileRoute` | `Store.Delete(r.Context(), file.Path)`. |
| `epub.go downloadEpubToTemp` | `Store.Download(ctx, key)`; left a `// TODO: read directly from the stored path` (post-migration this is a redundant local→tempfile copy — deferred refactor, per the plan). |

### Deletions (D4 / D8)

- `handlers/s3.go` deleted entirely (`CreateS3Client`, `listObjects` dead code,
  `uploadObject`/`downloadObject`/`deleteObject`/`UploadObject`, `getBucketName`).
- `TestInspector` (struct + field + conftest init) removed — its
  `FilesUploaded` counter was never asserted in any test.
- `go mod tidy` removed **all `aws-sdk-go-v2` modules** (`config`, `credentials`, `service/s3`, + the root SDK and ~13 transitives).
  `grep -rn aws go.mod go.sum` → 0; no `github.com/aws` import remains in any
  `.go` file. This is the meaningful dependency shrink the plan targets, and it
  does **not** strand the Phase 4 ETL because that tool is SDK-free by design (**D6**).

### Tests (D8)

- `handlers/setup_test.go`: `S.S3 = s.CreateS3Client()` removed (the store is
  set in `tests.Setup`).
- `tests/conftest.go`: `S.TestInspector` init removed; a real
  `storage.NewLocalStore(...)` is wired onto `S.Store`; the `B2_*`
  `setEnvIfNotSet` defaults are swapped for a `STORAGE_DIR` temp dir (so
  `loadStorageConfig` doesn't litter `./data/files` during the suite).
- `handlers/files_test.go`: `uploadTestFile` is now a real round-trip through
  the store (writes a real blob); `TestDownloadFile` asserts the streamed bytes
  equal `../tests/test.txt` (`"hello world"`) — exactly what the now-removed
  `Server.Testing` guard used to prevent.

### Verification

```
$ go build ./...           # clean
$ go vet ./...             # clean (only the 2 pre-existing oauth.go warnings)
$ go test ./...            # 16/16 packages PASS
$ gofmt -l                 # clean
$ grep -rn aws go.mod go.sum   # 0
```

### What's *not* in Phase 1 (deferred)

- `.env.example` / `.env-bash` still carry stale `S3_*` / `B2_*` lines → **Phase 2**.
- `docker-zettel-run.yml` / `zettel.env` volume + `STORAGE_DIR` override (**D7**) → **Phase 2**.
- Secret rotation (real B2 keys committed in `.env-bash`) → **Phase 2** follow-up.
- `services/llmprocessor.go` text-extraction still has the `// TODO: Download
  file from S3` stub (reads the `s3_key` column but never downloads). The store
  unblocks this; wiring it is a separate task.
- The thumbnail-skip on `Server.Testing` in `UploadFileRoute` is **kept** (with
  a re-documented reason): it's about the fire-and-forget goroutine outliving
  the rolled-back test transaction, *not* storage (**D8**).

---

## What's next (Phase 2 — config & deploy artifacts)

- `.env.example`: drop the stale `S3_*` block and the `B2_*` block; add
  `STORAGE_DIR=./data/files`.
- `.env-bash` + any remaining `B2_*` references: remove / rotate.
- `docker-zettel-run.yml` + `zettel.env`: reuse the existing
  `./data:/usr/src/app/data` mount and set `STORAGE_DIR=/usr/src/app/data/files`
  (**D7** — note the `/usr/src/app/...` prefix matches the existing mount).
- `Dockerfile`: add a `STORAGE_DIR` default `ENV` only if needed.
- **Security follow-up (file an issue):** rotate the decommissioned B2 keys
  and move surviving secrets out of git.

Phase 3 (tests green) is effectively already satisfied — the suite passes and
`TestDownloadFile` asserts real bytes — so Phase 2 can flow straight into
Phase 4 (B2→local ETL) and Phase 5 (cutover).
