# S3 (Backblaze B2) → Local File Storage Migration — Status

**Last updated:** 2026-07-29 (Phase 0 spike COMPLETE — `FileStore` interface + `LocalStore` landed and unit-tested; no call sites touched yet)
**Plan:** [`2026-07-29-s3-to-local-file-storage-design.md`](./2026-07-29-s3-to-local-file-storage-design.md)
**Tracking:** epic `Zettelgarden-yar` · Phase 0 `Zettelgarden-yar.1` (closed)

## TL;DR

**Phase 0 (spike) is done.** The new `go-backend/services/storage/` package
introduces the `Store` interface and a `LocalStore` implementation rooted at a
configurable directory, exactly as specified in design decision **D1**. All
three operations (`Upload`/`Download`/`Delete`) are implemented with the
write invariants the plan calls out (atomic temp-file-then-rename, `0750`
dirs / `0640` files, idempotent delete, defensive path-traversal guard per
**D2**). Eight unit tests cover the upload→download→delete round-trip,
overwrite semantics, nested-key dir creation, file-mode placement, the
traversal guard, missing-object behavior, and constructor dir-creation.

**Nothing is wired in yet** — by design. Phase 0 is a spike to confirm the
interface shape before touching the five call sites. The AWS SDK, `S3Config`,
`Server.S3`, `handlers/s3.go`, and all `B2_*` config are untouched and the
full suite still builds (`go build ./...`) and the new package is green
(`go test ./services/storage/...`).

---

## Phase status

| Phase | Status | Notes |
|---|---|---|
| 0 — Spike (interface + LocalStore) | ✅ **Done** | `services/storage/store.go` + `store_test.go`. `Store{Upload,Download,Delete}` interface; `LocalStore` with atomic writes + traversal guard. 8 tests green. |
| 1 — Wire the store through | ⬜ Not started | Add `StorageConfig`/`Server.Store`; repoint the 5 call sites (`files.go` ×4, `epub.go` ×1); delete `handlers/s3.go` + dead `handlerS3Uploader` + `TestInspector`; `go mod tidy`. |
| 2 — Config & deploy artifacts | ⬜ Not started | `.env.example`/`.env-bash`/`conftest.go` (D3, D8); `docker-zettel-run.yml` + `zettel.env` (D7). Secret-rotation follow-up. |
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

## What's next (Phase 1 — wire the store through)

1. Add `StorageConfig{Dir}` in `pkg/config/services.go` (replacing `S3Config`)
   + `loadStorageConfig()`; env `STORAGE_DIR`, default `./data/files` (**D3**).
2. Add `Server.Store storage.Store`; initialize in `main.go` from
   `STORAGE_DIR`.
3. Repoint the 5 call sites (`files.go` ×4: upload `:589`, thumbnail `:101`,
   download `:679`, delete `:716`; `epub.go:168`) off `Server.S3` +
   `uploadObject`/`downloadObject`/`deleteObject` → `s.Server.Store`. Two
   fix-while-here items in the download route (`files.go:679`): move
   `Content-Disposition` above `io.Copy` (currently set *after* the copy, so
   never sent); drop the `if s.Server.Testing { return }` at `files.go:686`
   (**D8**).
4. Delete `handlers/s3.go`, the dead `handlerS3Uploader` in `main.go`, and
   `TestInspector`.
5. `go mod tidy`; confirm `grep -rn "aws" go.*` is empty (**D4**).

The migration tool in Phase 4 is SDK-free by design (**D6**), so removing the
AWS SDK here does not strand it.
