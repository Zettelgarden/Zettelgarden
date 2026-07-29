# S3 (Backblaze B2) → Local File Storage Migration — Status

**Last updated:** 2026-07-29 (Phase 5 CUTOVER DEPLOYED on zg-internal / server-3 — backend re-imaged to local storage, 363 files served from `/home/nick/zg/files`; rollback image tagged; manual smoke + B2 bucket deletion pending.)
**Plan:** [`2026-07-29-s3-to-local-file-storage-design.md`](./2026-07-29-s3-to-local-file-storage-design.md)
**Tracking:** epic `Zettelgarden-yar` · Phase 0 `Zettelgarden-yar.1` (closed) · Phase 1 `Zettelgarden-yar.2` (closed) · Phase 2 `Zettelgarden-yar.3` (closed) · Phase 3 `Zettelgarden-yar.5` (closed) · Phase 4 `Zettelgarden-yar.6` (closed, code) · Security follow-up `Zettelgarden-yar.4` (open)

## TL;DR

**Phases 0–4 are done (code-complete).** The backend stores uploads on local
disk via `services/storage.LocalStore`; the AWS SDK is gone,
`handlers/s3.go` is deleted, every config/deploy artifact points at
`STORAGE_DIR`, and the one-time B2→local ETL tool (`cmd/migrate-storage`,
SDK-free) is built and tested. What remains is the production **run** of that
ETL + the Phase 5 cutover (both operator steps on the live host).

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
| 4 — Data migration (B2 → local) | ✅ **Done** (code + prod run) | `cmd/migrate-storage/` (D6): SDK-free SigV4 `GET` (`sign.go`, verified against an independent Python oracle — 4 vectors); reads `files.path`/``thumbnail_path` from sqlite or postgres; streams into `LocalStore`; idempotent via `Store.Exists`; classifies invalid-key / B2-404 as `not-in-source` (distinct from failures); 8 tests green. **Prod run on server-3: 363 objects copied (319 primary + 44 thumb, 480 MB), byte-verified vs B2; 48 orphaned rows not-in-source.** |
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

## Phase 4 progress

The B2→local ETL tool is built, SDK-free, and tested. Per design **D6** it has
no `aws-sdk-go` dependency, so it compiles from `main` today — after Phase 1
already removed the SDK — with no branch or re-add.

### What landed

- **`go-backend/cmd/migrate-storage/main.go`** — the tool. Flags: `--bucket`,
  `--endpoint` (default `https://s3.us-east-005.backblazeb2.com`), `--region`
  (default `us-east-005`, matching the old SDK config — **not** `us-east-1`),
  `--storage-dir` (`$STORAGE_DIR` / `./data/files`), `--db-driver`
  (sqlite|postgres), `--sqlite-path`, `--limit`, `--dry-run`, `--no-thumbnails`,
  `--timeout`. Source creds come from `B2_ACCESS_KEY_ID`/`B2_SECRET_ACCESS_KEY`.
  It reads `SELECT id, path, thumbnail_path FROM files WHERE
  COALESCE(is_deleted,false)=false AND path<>''`, copies each primary object +
  its thumbnail, and prints a summary (expected / downloaded / skipped / failed).
  `main()` is thin; the testable core is `runMigration(...)`.
- **`go-backend/cmd/migrate-storage/sign.go`** — hand-rolled **SigV4** for a
  path-style GET: `deriveSigningKey`, `escapeS3Path` (AWS-spec percent-encoding,
  preserving `/`), and `buildAuthorization`/`signGET`. ~150 lines, `crypto/hmac`
  + `net/http` only, **zero AWS modules**.
- **`go-backend/services/storage/store.go`** — added `Exists(ctx, key)` to the
  `Store` interface + `LocalStore` (only impl, so no other implementer breaks).
  The migration tool uses it for the idempotent "skip if already copied" guard.

### Testing (why the signer is trustworthy)

The SigV4 math is validated against **known-good vectors generated by an
independent Python stdlib oracle** (`hmac`/`hashlib`/`urllib`) — not from
memory, and not by re-running the Go code. Agreement on the full
`Authorization` header (incl. the 64-hex signature) across 4 cases — realistic
B2 file key + the real decommissioned dev creds, a thumbnail key, a nested
multi-segment key, and a special-character key (`' '` and `@` encoded, `/`
preserved) — validates canonical-request construction, the string-to-sign, the
signing-key chain, and path encoding all at once.

7 tests in `cmd/migrate-storage/`, all green:

| Test | Asserts |
|---|---|
| `TestBuildAuthorizationOracleVectors` | 4 Python-oracle signature vectors match exactly. |
| `TestEscapeS3Path` | `/` preserved; space/`@` encoded; unreserved passthrough. |
| `TestSignGETRoundTrip` | `signGET` wires host/escaped-path into the auth header; body streams over `httptest`. |
| `TestRunMigration` | E2E fake-B2 server (validates SigV4 + serves bytes) → temp SQLite `files` table → temp store: primary + thumbnail copied, **deleted row skipped**, empty-path row excluded, byte content matches, 2nd run 100% skipped. |
| `TestRunMigrationDryRun` | `--dry-run` fetches nothing, writes nothing. |
| `TestRunMigrationServerFailure` | a B2 404 is counted as `failed` (no crash); the run continues and can be re-run. |
| (`services/storage` `TestExists`) | `Exists` false→true→false across upload/delete; traversal key errors. |

### Verification

```
$ go build ./...            # clean
$ go vet ./...              # only the 2 pre-existing oauth.go warnings
$ go test ./...             # 16+ packages PASS (incl. cmd/migrate-storage: 7 tests)
$ gofmt -l cmd/migrate-storage services/storage   # clean
$ git diff --stat go-backend/go.mod go-backend/go.sum   # empty — NO new deps (SDK-free by design)
```

### Design notes / decisions

- **Region `us-east-005`, not `us-east-1`.** Recovered from the deleted
  `handlers/s3.go` (`git show d7635835^:go-backend/handlers/s3.go`): the old
  SDK client set `WithRegion("us-east-005")`. B2's S3 API signs with the
  bucket's actual region; using `us-east-1` would 403.
- **Idempotency via `Store.Exists`, not by re-downloading.** `LocalStore.Upload`
  is already atomic (temp+rename) so partials can't survive; `Exists` just gates
  the fetch so a resume skips already-copied objects.
- **Sequential copy.** A one-time, monitored run doesn't need a worker pool;
  a `--jobs` flag is a trivial follow-up if the production file count is large.
- **No new deps.** Reuses `lib/pq` (postgres path), `go-backend/server`
  (`OpenSQLite`), and `go-backend/services/storage`. `go.mod`/`go.sum` untouched.

### Production ETL run (2026-07-29, server-3 / `192.168.0.20`)

Ran for real against the live `zettelgarden-files` B2 bucket + the live SQLite
DB (`DB_DRIVER=sqlite`, `SQLITE_PATH=/usr/src/app/data/zettelgarden.db`, host
path `/home/nick/zg/zettelgarden.db`). Destination `STORAGE_DIR=/home/nick/zg/files`
(host) = the container's `/usr/src/app/data/files` post-cutover (the compose
mount is `/home/nick/zg:/usr/src/app/data`).

```
files seen: 367   expected objects: 411
primary  downloaded: 319   skipped: 0   not-in-source: 48   failed: 0
thumb    downloaded:  44   skipped: 0   not-in-source:  0   failed: 0
```

Verification:
- **363 local files** (319 primary + 44 thumb), 480 MB; dirs mode `0750`,
  files `0640` (as designed).
- **Byte-perfect**: SHA-256 of a migrated primary (`1/0499be71-…`, 355415 B)
  and a thumbnail (`1/0a2f1e58-…_thumb.jpg`, 20355 B) both matched the live
  B2 object exactly. A throwaway SigV4 probe confirmed the signer returns
  HTTP 200 against real B2.
- **Atomic writes clean**: no leftover `.upload-*` / `.storage-probe-*` temps.
- **One transient `unexpected EOF`** (B2 closed a stream mid-transfer on
  `id=360`); the atomic write discarded the partial and the idempotent re-run
  fetched it cleanly on the second pass — exactly the resume design.

**Data discovery (pre-existing, not caused by the migration):** 48 rows are
orphaned — their files are absent in B2 (HTTP 404 `NoSuchKey`) and were never
migratable:
- **46 legacy rows** (ids ≈ 5–53) store `files.path` as a full absolute path
  `/usr/src/app/files/<uuid>.<ext>` — a pre-B2 scheme that kept files on the
  container's *ephemeral* disk. Those files were never uploaded to B2 and are
  long gone. The store's path-traversal guard (correctly) rejects absolute
  keys; the tool now classifies these as `not-in-source` instead of `failed`.
- **2 normal-format keys** (`{userID}/{uuid}`) also 404 in B2 — additional
  orphaned rows from some past data loss.

These 48 rows were **already broken pre-cutover** (a B2 download 404s today);
the migration doesn't worsen them. Cleanup (mark deleted) is filed as a
follow-up (`Zettelgarden-yar.7`) and is independent of cutover.

The running backend is still the pre-Phase-1 image (built 2026-07-28, uploads
still go to B2), so this run was a pure pre-cutover populate — the live app
was untouched.

---

## What's next (Phase 5 — cutover DEPLOYED on zg-internal; smoke + cleanup remain)

**Cutover is live on `server-3` (192.168.0.20, `zg-internal`) as of 2026-07-29.**
The backend container was re-imaged from current `main` (Phase 1+ code + Phase 2
Dockerfile `ENV STORAGE_DIR=/usr/src/app/data/files`) via `docker save | load`
(scoped to server-3 — no registry mutation, so the public instance on
192.168.0.93 is unaffected). Verified: container runs the new image
(`sha256:3927b89c…`), `STORAGE_DIR=/usr/src/app/data/files` set, 363 files
visible inside the container at that path, `/api/files/download/1` returns 401
(route alive, auth working, not a 500), container can write new uploads to the
mount. **Rollback image tagged**
`nsavage/zettelgarden_go_backend:rollback-pre-local-storage`
(`sha256:b030f456…`, the pre-cutover B2 image).

Remaining:
- **Manual smoke (Nick):** authenticated upload→thumbnail→download→epub-import→delete against the re-deployed backend to confirm end-to-end for real users. (Infra-level checks pass; this needs a logged-in session.)
- **Rollback if needed:** on server-3, `docker tag nsavage/zettelgarden_go_backend:rollback-pre-local-storage nsavage/zettelgarden_go_backend:latest && cd /mnt/nas-2-fast-data/config/services/zg-internal && docker compose up -d --no-deps go_backend`.
- **After a green window:** delete the B2 bucket + revoke keys (`Zettelgarden-yar.4`), then remove the now-inert `B2_*` lines from server-3's `.env`. Cleanup of the 48 orphaned rows is `Zettelgarden-yar.7` (independent).

Note: this cutover was scoped to `zg-internal` (server-3). The public instance
on `192.168.0.93` still runs the old B2 image and would need its own ETL +
cutover (the `build.sh` flow targets it).

The `migrate-storage` binary is left at `/home/nick/migrate-storage` on
server-3 for an idempotent re-run.
