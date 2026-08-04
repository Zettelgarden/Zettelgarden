package handlers

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSignVerifyOAuthStateRoundTrip(t *testing.T) {
	key := []byte("test-secret-key-for-state-cookie-32+")
	p := oauthStatePayload{State: "abc", Nonce: "xyz", Ver: "pkce-verifier"}

	raw, err := signOAuthState(key, p)
	if err != nil {
		t.Fatalf("signOAuthState: %v", err)
	}
	if !strings.Contains(raw, ".") {
		t.Fatalf("expected payload.sig format, got %q", raw)
	}

	got, err := verifyOAuthState(key, raw)
	if err != nil {
		t.Fatalf("verifyOAuthState: %v", err)
	}
	if got.State != p.State || got.Nonce != p.Nonce || got.Ver != p.Ver {
		t.Fatalf("payload mismatch: got %+v, want state/nonce/ver", got)
	}
}

func TestVerifyOAuthStateRejectsBadKey(t *testing.T) {
	raw, _ := signOAuthState([]byte("correct-secret-key-aaaaaaaaaaaaaa"), oauthStatePayload{State: "s"})
	if _, err := verifyOAuthState([]byte("WRONG-secret-key-bbbbbbbbbbbbb"), raw); err == nil {
		t.Fatal("expected signature mismatch error")
	}
}

func TestVerifyOAuthStateRejectsTamperedPayload(t *testing.T) {
	key := []byte("test-secret-key-for-state-cookie-32+")
	raw, _ := signOAuthState(key, oauthStatePayload{State: "abc"})
	// Flip a character in the payload half (before the ".").
	tampered := raw[:5] + "X" + raw[6:]
	if _, err := verifyOAuthState(key, tampered); err == nil {
		t.Fatal("expected tamper detection")
	}
}

func TestVerifyOAuthStateRejectsExpired(t *testing.T) {
	key := []byte("test-secret-key-for-state-cookie-32+")
	// signOAuthState forces exp=now+TTL, so craft an already-expired payload
	// manually using the same internal layout (payload.sig).
	body, _ := json.Marshal(oauthStatePayload{State: "s", Exp: time.Now().Add(-time.Minute).Unix()})
	raw := base64.RawURLEncoding.EncodeToString(body) + "." +
		base64.RawURLEncoding.EncodeToString(hmacSum(key, body))
	if _, err := verifyOAuthState(key, raw); err == nil {
		t.Fatal("expected expiry error")
	}
}

func TestVerifyOAuthStateRejectsMalformed(t *testing.T) {
	key := []byte("test-secret-key-for-state-cookie-32+")
	for _, bad := range []string{"", "no-signature", "a.b.c", "!!!.@@@"} {
		if _, err := verifyOAuthState(key, bad); err == nil {
			t.Fatalf("expected error for malformed input %q", bad)
		}
	}
}

func TestRandomStringUnique(t *testing.T) {
	a, _ := randomString(24)
	b, _ := randomString(24)
	if a == "" || b == "" {
		t.Fatal("empty random string")
	}
	if a == b {
		t.Fatal("expected distinct random strings")
	}
}
