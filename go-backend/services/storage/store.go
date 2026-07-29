// Package storage abstracts on-disk (and, in tests, tempdir-backed) storage of
// uploaded file blobs and thumbnails.
//
// This is part of the S3/Backblaze B2 → local-file-storage migration
// (see docs/plans/2026-07-29-s3-to-local-file-storage-design.md). The only
// production implementation is LocalStore; the Store interface exists so call
// sites can be swapped off the AWS SDK and so tests get a real (tempdir) store
// instead of the Server.Testing no-op fork that s3.go used to carry.
//
// Keys are reused verbatim as relative paths (design decision D2): a key like
// "42/<uuid>" maps to {dir}/42/<uuid>, and a thumbnail key "{key}_thumb.jpg"
// maps similarly. Keys are server-generated, but resolve() defensively rejects
// empty / absolute / path-traversing keys.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Store is the file-blob abstraction used by the upload/download/delete routes.
//
// The three methods are context-aware for API symmetry, even though the local
// implementation's I/O is not context-interruptible (local copies are fast and
// blocking). Callers should still pass request contexts so future backends can
// honor cancellation.
type Store interface {
	// Upload writes the bytes read from r to the given key, overwriting any
	// existing object (matching B2/S3 PutObject semantics).
	Upload(ctx context.Context, key string, r io.Reader) error
	// Download opens the object at key for reading. The caller must close the
	// returned reader. A missing object is reported as an error wrapping
	// fs.ErrNotExist (use errors.Is to detect it).
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object at key. It is idempotent: a missing object
	// returns nil (matching B2/S3 DeleteObject).
	Delete(ctx context.Context, key string) error
	// Exists reports whether an object is present at key. It does not return an
	// error for a missing object (the bool is simply false); errors are reserved
	// for path-resolution or stat failures. Used by the one-time B2→local
	// migration tool to skip objects already copied on a re-run.
	Exists(ctx context.Context, key string) (bool, error)
}

// Compile-time guarantee that LocalStore satisfies Store.
var _ Store = (*LocalStore)(nil)

// LocalStore persists file blobs to a directory on local disk.
//
// Write invariants (the correctness properties B2 gave us for free and local
// disk does not — see design decision D1):
//
//   - Upload creates the key's parent directory with os.MkdirAll (mode 0750),
//     since keys contain "/" (e.g. "42/<uuid>").
//   - Upload writes to a temp file in the SAME directory and then renames it
//     into place, so a crash mid-write cannot leave a partial object.
//   - Final file mode is 0640.
//   - Download returns an *os.File (satisfies io.ReadCloser); the caller closes.
//   - Delete tolerates a missing file (returns nil for fs.ErrNotExist).
type LocalStore struct {
	dir string // absolute path to the storage root
}

// NewLocalStore returns a LocalStore rooted at dir. The directory is created
// (mode 0750) if it does not exist, and a probe write confirms it is writable.
func NewLocalStore(dir string) (*LocalStore, error) {
	if dir == "" {
		return nil, errors.New("storage: empty directory")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("storage: resolve dir %q: %w", dir, err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("storage: create dir %q: %w", abs, err)
	}
	// Confirm we can actually write into it before returning.
	probe, err := os.CreateTemp(abs, ".storage-probe-*")
	if err != nil {
		return nil, fmt.Errorf("storage: dir %q not writable: %w", abs, err)
	}
	probe.Close()
	os.Remove(probe.Name())
	return &LocalStore{dir: abs}, nil
}

// Dir returns the absolute path of the storage root. It is exposed for
// diagnostics and tests; production callers should never read/write files
// directly — go through the Store methods.
func (s *LocalStore) Dir() string { return s.dir }

// Upload writes r to the given key, atomically replacing any existing object.
func (s *LocalStore) Upload(ctx context.Context, key string, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target, err := s.resolve(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("storage: create dir %q: %w", dir, err)
	}

	// Write to a temp file in the same directory, then rename into place.
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("storage: create temp file for %q: %w", key, err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if any step below fails; a nil error means the
	// rename succeeded and tmpPath no longer exists.
	cleanup := func() { os.Remove(tmpPath) }

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("storage: write %q: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("storage: close temp for %q: %w", key, err)
	}
	if err := os.Chmod(tmpPath, 0o640); err != nil {
		cleanup()
		return fmt.Errorf("storage: chmod %q: %w", key, err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		cleanup()
		return fmt.Errorf("storage: rename into place for %q: %w", key, err)
	}
	return nil
}

// Download opens the object at key for reading.
func (s *LocalStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(target)
	if err != nil {
		return nil, fmt.Errorf("storage: open %q: %w", key, err)
	}
	return f, nil
}

// Delete removes the object at key, tolerating a missing file.
func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // idempotent, like B2's DeleteObject
		}
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	return nil
}

// Exists reports whether an object is present at key. A missing object returns
// (false, nil); path-traversal attempts surface as errors via resolve().
func (s *LocalStore) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	target, err := s.resolve(key)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("storage: stat %q: %w", key, err)
	}
	return !info.IsDir(), nil
}

// resolve maps a (server-generated) key to an absolute path under the storage
// root, rejecting keys that are empty, absolute, or that escape the root via
// ".." segments. The trailing filepath.Rel check is defense-in-depth: even if
// the path.Clean logic above missed a case, a key that resolves outside the
// root is rejected here.
func (s *LocalStore) resolve(key string) (string, error) {
	if key == "" {
		return "", errors.New("storage: empty key")
	}
	c := path.Clean(key)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") || strings.HasPrefix(c, "/") {
		return "", fmt.Errorf("storage: invalid key %q (must be a clean relative path)", key)
	}
	target := filepath.Join(s.dir, filepath.FromSlash(c))
	rel, err := filepath.Rel(s.dir, target)
	if err != nil {
		return "", fmt.Errorf("storage: invalid key %q: %w", key, err)
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("storage: key %q escapes storage root", key)
	}
	return target, nil
}
