package utils

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Run("returns key with correct prefix", func(t *testing.T) {
		key, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		if !strings.HasPrefix(key, "zg_live_") {
			t.Errorf("GenerateAPIKey() key = %q, want prefix 'zg_live_'", key)
		}
	})

	t.Run("returns key of correct length", func(t *testing.T) {
		key, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		// Prefix (8) + 32 bytes encoded as hex (64) = 72
		expectedLen := len("zg_live_") + 64
		if len(key) != expectedLen {
			t.Errorf("GenerateAPIKey() key length = %d, want %d", len(key), expectedLen)
		}
	})

	t.Run("generates unique keys", func(t *testing.T) {
		keys := make(map[string]bool)
		for i := 0; i < 100; i++ {
			key, err := GenerateAPIKey()
			if err != nil {
				t.Fatalf("GenerateAPIKey() error = %v", err)
			}
			if keys[key] {
				t.Errorf("GenerateAPIKey() generated duplicate key: %s", key)
			}
			keys[key] = true
		}
	})

	t.Run("contains only valid hex characters after prefix", func(t *testing.T) {
		key, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		hexPart := strings.TrimPrefix(key, "zg_live_")
		for _, c := range hexPart {
			if !isHexDigit(c) {
				t.Errorf("GenerateAPIKey() key contains non-hex character: %c", c)
			}
		}
	})
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

func TestHashAPIKey(t *testing.T) {
	t.Run("produces different hash than key", func(t *testing.T) {
		key := "zg_live_test123"
		hash, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		if hash == key {
			t.Errorf("HashAPIKey() hash = key, should be different")
		}
	})

	t.Run("produces different hashes for same key", func(t *testing.T) {
		key := "zg_live_test123"
		hash1, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		hash2, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		if hash1 == hash2 {
			t.Errorf("HashAPIKey() should produce different hashes due to salt (bcrypt)")
		}
	})

	t.Run("produces non-empty hash", func(t *testing.T) {
		key := "zg_live_test123"
		hash, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		if hash == "" {
			t.Errorf("HashAPIKey() returned empty hash")
		}
	})
}

func TestVerifyAPIKey(t *testing.T) {
	t.Run("correct key passes", func(t *testing.T) {
		key := "zg_live_test123"
		hash, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		if !VerifyAPIKey(key, hash) {
			t.Errorf("VerifyAPIKey() = false for correct key, want true")
		}
	})

	t.Run("wrong key fails", func(t *testing.T) {
		key := "zg_live_test123"
		hash, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		if VerifyAPIKey("zg_live_wrong", hash) {
			t.Errorf("VerifyAPIKey() = true for wrong key, want false")
		}
	})

	t.Run("empty key fails", func(t *testing.T) {
		key := "zg_live_test123"
		hash, err := HashAPIKey(key)
		if err != nil {
			t.Fatalf("HashAPIKey() error = %v", err)
		}
		if VerifyAPIKey("", hash) {
			t.Errorf("VerifyAPIKey() = true for empty key, want false")
		}
	})

	t.Run("invalid hash fails", func(t *testing.T) {
		key := "zg_live_test123"
		if VerifyAPIKey(key, "invalid_hash") {
			t.Errorf("VerifyAPIKey() = true for invalid hash, want false")
		}
	})
}
