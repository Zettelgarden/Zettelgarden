package storage

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestStore returns a LocalStore rooted at a fresh temp dir.
func newTestStore(t *testing.T) *LocalStore {
	t.Helper()
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	return store
}

func TestRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	want := []byte("hello, zettelgarden\n")
	key := "42/550e8400-e29b-41d4-a716-446655440000"

	if err := store.Upload(ctx, key, bytes.NewReader(want)); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	rc, err := store.Download(ctx, key)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer rc.Close()

	got := make([]byte, 0, len(want))
	buf := make([]byte, 32)
	for {
		n, readErr := rc.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
		}
		if readErr != nil {
			break
		}
	}
	if !bytes.Equal(got, want) {
		t.Errorf("downloaded bytes don't match uploaded: got %q, want %q", got, want)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// After delete, the object is gone.
	if _, err := store.Download(ctx, key); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Download after Delete: want fs.ErrNotExist, got %v", err)
	}
}

func TestDeleteMissingIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	// A key that was never uploaded; Delete must be a no-op (like B2).
	if err := store.Delete(context.Background(), "never/uploaded"); err != nil {
		t.Errorf("Delete on missing object: want nil, got %v", err)
	}
}

func TestDownloadMissingWrapsErrNotExist(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Download(context.Background(), "nope/missing")
	if err == nil {
		t.Fatal("Download missing: want error, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Download missing: want an error wrapping fs.ErrNotExist, got %v", err)
	}
}

func TestUploadCreatesNestedDirs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := "7/aaa/bbb/deep-key" // multiple path segments

	if err := store.Upload(ctx, key, strings.NewReader("deep")); err != nil {
		t.Fatalf("Upload into nested path: %v", err)
	}
	// The object lands under {dir}/7/aaa/bbb/deep-key.
	_, err := os.Stat(filepath.Join(store.Dir(), "7", "aaa", "bbb", "deep-key"))
	if err != nil {
		t.Errorf("expected nested file on disk: %v", err)
	}

	rc, err := store.Download(ctx, key)
	if err != nil {
		t.Fatalf("Download nested: %v", err)
	}
	defer rc.Close()
}

func TestUploadOverwrites(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	key := "1/abc"

	if err := store.Upload(ctx, key, strings.NewReader("first")); err != nil {
		t.Fatalf("Upload first: %v", err)
	}
	if err := store.Upload(ctx, key, strings.NewReader("second")); err != nil {
		t.Fatalf("Upload second: %v", err)
	}

	rc, err := store.Download(ctx, key)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer rc.Close()

	var got strings.Builder
	buf := make([]byte, 16)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			got.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	if got.String() != "second" {
		t.Errorf("overwrite: got %q, want %q", got.String(), "second")
	}
}

func TestUploadFileModeAndAtomicPlacement(t *testing.T) {
	store := newTestStore(t)
	key := "3/rename-target"

	if err := store.Upload(context.Background(), key, strings.NewReader("x")); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	info, err := os.Stat(filepath.Join(store.Dir(), "3", "rename-target"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o640 {
		t.Errorf("file mode: got %o, want 0640", perm)
	}

	// No leftover temp files in the parent directory.
	entries, err := os.ReadDir(filepath.Join(store.Dir(), "3"))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".upload-") {
			t.Errorf("leftover temp file after Upload: %s", e.Name())
		}
	}
}

func TestPathTraversalGuard(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for _, key := range []string{
		"",                    // empty
		".",                   // root
		"../escape",           // escapes up
		"42/../..",            // cleans to root / escapes
		"42/../../etc/passwd", // escapes
		"/abs/path",           // absolute
		"../",                 // leading dotdot
	} {
		t.Run("key="+key, func(t *testing.T) {
			if err := store.Upload(ctx, key, strings.NewReader("x")); err == nil {
				t.Errorf("Upload(%q): want error, got nil", key)
			}
			if _, err := store.Download(ctx, key); err == nil {
				t.Errorf("Download(%q): want error, got nil", key)
			}
			if err := store.Delete(ctx, key); err == nil {
				// Delete tolerates fs.ErrNotExist, but a bad key must be
				// rejected before it reaches the filesystem.
				t.Errorf("Delete(%q): want error, got nil", key)
			}
		})
	}

	// Sanity check: a normal key still works in the same store.
	good := "5/ok"
	if err := store.Upload(ctx, good, strings.NewReader("ok")); err != nil {
		t.Fatalf("Upload(good key) failed unexpectedly: %v", err)
	}
}

func TestNewLocalStoreCreatesMissingDir(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "nested", "store")
	store, err := NewLocalStore(target)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	if info, err := os.Stat(store.Dir()); err != nil || !info.IsDir() {
		t.Errorf("expected store dir to exist at %s", store.Dir())
	}
}

func TestExists(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	ctx := context.Background()
	key := "42/abc-uuid"

	if ok, err := store.Exists(ctx, key); err != nil || ok {
		t.Fatalf("before upload: Exists = (%v, %v), want (false, nil)", ok, err)
	}
	if err := store.Upload(ctx, key, strings.NewReader("payload")); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if ok, err := store.Exists(ctx, key); err != nil || !ok {
		t.Fatalf("after upload: Exists = (%v, %v), want (true, nil)", ok, err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, err := store.Exists(ctx, key); err != nil || ok {
		t.Fatalf("after delete: Exists = (%v, %v), want (false, nil)", ok, err)
	}

	// A traversal attempt surfaces as an error wrapping ErrInvalidKey.
	_, err = store.Exists(ctx, "../escape")
	if err == nil || !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Exists with traversal key: err=%v, want ErrInvalidKey", err)
	}
}
