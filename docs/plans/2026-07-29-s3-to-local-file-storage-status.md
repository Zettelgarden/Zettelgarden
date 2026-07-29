# S3 (Backblaze B2) → Local File Storage Migration — Status

**Last updated:** 2026-07-29 (Phase 2 COMPLETE — all config/deploy artifacts swapped from S3/B2 to `STORAGE_DIR`; CI dead `B2_*` secrets removed; README updated; suite green. Phase 3 automated gate re-confirmed green.)
**Plan:** [`2026-07-29-s3-to-local-file-storage-design.md`](./2026-07-29-s3-to-local-file-storage-design.md)
**Tracking:** epic `Zettelgarden-yar` · Phase 0 `Zettelgarden-yar.1` (closed) · Phase 1 `Zettelgarden-yar.2` (closed) · Phase 2 `Zettelgarden-yar.3` (closed) · Phase 3 `Zettelgarden-yar.5` (closed, automated) · Security follow-up `Zettelgarden-yar.4` (open)

## TL;DR

**Phase 0 + Phase 1 + Phase 2 are done (Phase 3 automated gate green).** The
backend stores uploads on local disk via `services/storage.LocalStore`; the
AWS SDK is gone, `handlers/s3.go` is deleted, and every config/deploy artifact
now points at `STORAGE_DIR` instead of S3/B2.

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
| 2 — Config & deploy artifacts | ✅ **Done** | `.env.example` S3 block → `STORAGE_DIR` (D3); `.env-bash` `B2_*` → `STORAGE_DIR`; `Dockerfile` `ENV STORAGE_DIR=/usr/src/app/data/files` (D7); `docker-zettel-run.yml` documents the shared data volume carrying `files/` (D7); `.github/workflows/go.yml` dead `B2_*` GitHub-secret lines removed; `conftest.go` stale comment fixed; README File-Storage bullets → local disk. |
| 3 — Tests green | ✅ **Done** (automated) | Re-ran `go test ./...` — 16/16 packages PASS (Phase 1 already made `TestDownloadFile` assert real bytes). Manual upload→thumb→download→epub→delete smoke folded into the Phase 5 cutover runbook. |
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

## Phase 2 progress

All config & deploy artifacts swapped from S3/B2 to `STORAGE_DIR`; CI dead
secrets removed; README updated. Suite still green.

### Changes (D3 / D7)

- **`go-backend/.env.example`** — stale `S3_*` block replaced by a documented
  `STORAGE_DIR=./data/files` (default, sits next to `SQLITE_PATH`).
- **`go-backend/.env-bash`** — decommissioned `B2_ACCESS_KEY_ID` /
  `B2_SECRET_ACCESS_KEY` / `B2_BUCKET_NAME` lines removed; `STORAGE_DIR` added.
  *(Confirmed `.env-bash` is **gitignored and never tracked** — `git log --all`
  empty — so its removal here is local hygiene only, not a history scrub.)*
- **`go-backend/Dockerfile`** — added `ENV STORAGE_DIR=/usr/src/app/data/files`
  (D7). This lands on the existing `./data:/usr/src/app/data` mount via the
  `WORKDIR`, is overridable at runtime by `env_file`/`-e`, and means the
  container works even with an empty `zettel.env`.
- **`docker-zettel-run.yml`** — the existing volume is now documented as also
  carrying the `files/` subtree; one backup target covers DB + uploads (D7).
  `STORAGE_DIR` is set in the (host-side, untracked) `zettel.env`, inheriting
  the Dockerfile default otherwise.
- **`.github/workflows/go.yml`** — removed the 3 dead `B2_*` GitHub-secret env
  lines. The conftest already defaults `STORAGE_DIR` to a process temp dir, so
  no replacement secret is needed. *(The matching GitHub repo secrets should
  now be deleted from Settings → Secrets — see follow-up `yar.4`.)*
- **`go-backend/tests/conftest.go`** — dropped the stale "Phase 2 swaps this"
  comment (the `setEnvIfNotSet("STORAGE_DIR", …)` was already correct from
  Phase 1).
- **`README.md`** — File-Storage bullets (features, backend stack,
  self-hosting requirements) updated from "S3-compatible" to local on-disk.

### Out of scope (left as-is, intentionally)

- The DB column / job-payload key `s3_key` and the `pending_s3_integration`
  status string in `services/llmprocessor.go` — renaming is a schema/contract
  change; the plan is explicitly **no schema change**. The text-extraction
  download wiring is its own task (the store now unblocks it).
- Historical design docs (`docs/plans/2026-03-09-filevault-*.md`) mention
  `s3_key` — they're immutable records, left untouched.
- The broad pre-existing `gofmt` drift (47 files) is unchanged by this phase;
  the one Go file Phase 2 touched (`tests/conftest.go`) is gofmt-clean.

### Verification

```
$ go build ./...            # clean
$ go vet ./...              # only the 2 pre-existing oauth.go warnings
$ go test ./...             # 16/16 packages PASS
$ git grep -n 'S3_\|B2_\|aws-sdk\|backblaze' -- '*.go' '*.yml' '*.example' 'Dockerfile' 'README.md'
                            # (no matches in tracked config/deploy artifacts)
```

### Follow-up filed

- `Zettelgarden-yar.4` (open) — rotate/revoke the decommissioned B2 keys in
  the Backblaze console, delete the now-unused GitHub repo secrets, and delete
  the bucket after the Phase 5 rollback window.

---

## What's next (Phase 4 — B2→local ETL, then Phase 5 cutover)

- **Phase 4 (D6):** build & run the SDK-free `cmd/migrate-storage` on the
  production host, writing into the populated `STORAGE_DIR`. It reads the key
  list from the DB and raw SigV4-`GET`s each object from B2 (no AWS module), so
  it builds from `main` at any time — including now, after Phase 1 dropped the
  SDK. Idempotent (skip if local file exists); verify `count(files) ≈` local
  file count. Keep B2 read-only for the rollback window.
- **Phase 5 (cutover):** deploy the new binary pointed at the populated
  `STORAGE_DIR`; smoke-test a real upload/download (this is where the deferred
  manual upload→thumb→download→epub→delete check from Phase 3 runs); after a
  green window, delete the B2 bucket and revoke keys (`yar.4`).

The `.env-bash` B2 keys are already removed locally and the `B2_*` GitHub
secrets are unreferenced, so the only remaining B2 touchpoints are the
production host's `zettel.env` (operator) and the Backblaze console itself
(`yar.4`).
