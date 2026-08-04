package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OIDC/OAuth CSRF + PKCE state, carried in a short-lived signed cookie so no
// server-side session store is required. The cookie value is
// base64url(payload) + "." + base64url(HMAC-SHA256(payload, JwtSecretKey)).
//
// The payload holds everything the callback needs to validate the redirect and
// complete the token exchange:
//   - state:  OAuth `state` value, echoed back by the IdP (CSRF defense)
//   - nonce:  OIDC `nonce`, validated inside the id_token (replay defense)
//   - ver:    PKCE code_verifier (S256 challenge was sent in the auth request)
//   - exp:    unix seconds after which the cookie is rejected
//
// The same helpers are intended to back the GitHub OAuth flow's missing `state`
// protection in a follow-up (Phase 3 hardening).

const oidcCookieName = "zg_oauth_state"
const oauthStateTTL = 10 * time.Minute

type oauthStatePayload struct {
	State string `json:"state"`
	Nonce string `json:"nonce"`
	Ver   string `json:"ver"` // PKCE code_verifier
	Exp   int64  `json:"exp"` // unix seconds
}

// randomString returns n bytes of crypto-random data as base64url (≈1.33*n chars).
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// signOAuthState packs the payload (filling Exp from now+TTL), HMAC-signs it,
// and returns the "payload.sig" cookie value.
func signOAuthState(key []byte, p oauthStatePayload) (string, error) {
	p.Exp = time.Now().Add(oauthStateTTL).Unix()
	body, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sig := hmacSum(key, body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// verifyOAuthState validates the signature and expiry and returns the payload.
func verifyOAuthState(key []byte, raw string) (oauthStatePayload, error) {
	var p oauthStatePayload
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 {
		return p, fmt.Errorf("malformed state cookie")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return p, fmt.Errorf("bad state encoding: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return p, fmt.Errorf("bad signature encoding: %w", err)
	}
	want := hmacSum(key, body)
	if subtle.ConstantTimeCompare(sig, want) != 1 {
		return p, fmt.Errorf("invalid state signature")
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return p, fmt.Errorf("bad state payload: %w", err)
	}
	if time.Now().Unix() > p.Exp {
		return p, fmt.Errorf("state cookie expired")
	}
	return p, nil
}

func hmacSum(key, body []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(body)
	return mac.Sum(nil)
}

// setOAuthStateCookie writes the signed state cookie. Secure is set when the
// redirect URI (or app URL) is https; SameSite=Lax lets the IdP top-level
// redirect carry it back while keeping it JS-inaccessible.
func setOAuthStateCookie(w http.ResponseWriter, value, referenceURL string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   strings.HasPrefix(referenceURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	})
}

// clearOAuthStateCookie deletes the state cookie.
func clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
