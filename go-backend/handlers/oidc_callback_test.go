package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go-backend/pkg/config"
	"go-backend/tests"
)

// stubOIDCClient is a test double for oidcClient: it returns a fixed verified
// identity (or error) without touching the network.
type stubOIDCClient struct {
	identity *oidcVerifiedIdentity
	err      error
}

func (s *stubOIDCClient) ExchangeAndVerify(ctx context.Context, code, verifier string) (*oidcVerifiedIdentity, error) {
	return s.identity, s.err
}

// newOIDCCallbackHarness returns a Handler with OIDC enabled, a fixed frontend
// URL, and (optionally) a stub client installed.
func newOIDCCallbackHarness(t *testing.T, client oidcClient) *Handler {
	t.Helper()
	s := NewHandler()
	t.Cleanup(tests.Teardown)
	s.OIDCConfig = config.OIDCConfig{
		Enabled:       true,
		Issuer:        "https://idp.example",
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:8079/api/auth/oidc/callback",
		ProviderLabel: "test-idp",
	}
	s.oidcClientOverride = client
	t.Setenv("ZETTEL_URL", "http://localhost:5173")
	return s
}

// oidcCallbackRequest builds a GET to the OIDC callback with the given query
// params and a signed state cookie holding cookieState (fixed nonce/verifier).
func oidcCallbackRequest(s *Handler, t *testing.T, cookieState, queryState, code string) *http.Request {
	t.Helper()
	u := "/api/auth/oidc/callback?state=" + url.QueryEscape(queryState)
	if code != "" {
		u += "&code=" + url.QueryEscape(code)
	}
	req := httptest.NewRequest(http.MethodGet, u, nil)
	raw, err := signOAuthState(s.Server.JwtSecretKey, oauthStatePayload{
		State: cookieState,
		Nonce: "nonce-1",
		Ver:   "verifier-1",
	})
	if err != nil {
		t.Fatalf("sign state cookie: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: oidcCookieName, Value: raw})
	return req
}

// assertOIDCRedirect asserts the handler redirected to the frontend login page
// with the given error code.
func assertOIDCRedirect(t *testing.T, rr *httptest.ResponseRecorder, code string) {
	t.Helper()
	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rr.Code)
	}
	want := "http://localhost:5173/login?error=" + url.QueryEscape(code)
	if got := rr.Header().Get("Location"); got != want {
		t.Fatalf("expected redirect %q, got %q", want, got)
	}
}

func TestCallbackOIDCRoute_HappyPath(t *testing.T) {
	stub := &stubOIDCClient{identity: &oidcVerifiedIdentity{
		Subject:           "sub-alice",
		Nonce:             "nonce-1",
		Email:             "alice@example.com",
		EmailVerified:     true,
		Name:              "Alice Example",
		PreferredUsername: "alice",
	}}
	s := newOIDCCallbackHarness(t, stub)

	req := oidcCallbackRequest(s, t, "state-1", "state-1", "code-1")
	rr := httptest.NewRecorder()
	s.CallbackOIDCRoute(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.HasPrefix(loc, "http://localhost:5173/login?token=") {
		t.Fatalf("expected JWT redirect to frontend, got %q", loc)
	}

	// The state cookie must be consumed (cleared) after a successful login.
	cleared := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == oidcCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected state cookie to be cleared after successful callback")
	}

	// The issued token must belong to the newly created OIDC user.
	rawToken := strings.TrimPrefix(loc, "http://localhost:5173/login?token=")
	claims, err := s.decodeToken(rawToken)
	if err != nil {
		t.Fatalf("issued token does not parse: %v", err)
	}
	var email string
	if err := s.GetDB().QueryRow(`SELECT email FROM users WHERE id = $1`, claims.Sub).Scan(&email); err != nil {
		t.Fatalf("lookup user %d: %v", claims.Sub, err)
	}
	if email != "alice@example.com" {
		t.Fatalf("expected token for alice@example.com, got %q", email)
	}
}

func TestCallbackOIDCRoute_MissingState(t *testing.T) {
	s := newOIDCCallbackHarness(t, &stubOIDCClient{identity: &oidcVerifiedIdentity{}})

	// No state cookie at all.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state=state-1&code=code-1", nil)
	rr := httptest.NewRecorder()
	s.CallbackOIDCRoute(rr, req)

	assertOIDCRedirect(t, rr, "missing_state")
}

func TestCallbackOIDCRoute_BadState(t *testing.T) {
	// Cookie signature/format verification fails before any provider step, so
	// no client is installed and the network is never touched.
	s := newOIDCCallbackHarness(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state=state-1&code=code-1", nil)
	req.AddCookie(&http.Cookie{Name: oidcCookieName, Value: "tampered-not-signed"})
	rr := httptest.NewRecorder()
	s.CallbackOIDCRoute(rr, req)

	assertOIDCRedirect(t, rr, "bad_state")
}

func TestCallbackOIDCRoute_StateMismatch(t *testing.T) {
	s := newOIDCCallbackHarness(t, nil)

	// Cookie holds state-1 but the IdP echoes a different state.
	req := oidcCallbackRequest(s, t, "state-1", "state-ATTACKER", "code-1")
	rr := httptest.NewRecorder()
	s.CallbackOIDCRoute(rr, req)

	assertOIDCRedirect(t, rr, "state_mismatch")
}

func TestCallbackOIDCRoute_NonceMismatch(t *testing.T) {
	// Stub verifies the token but the nonce does not match the one we stored
	// in the state cookie — the replay defense must reject it.
	stub := &stubOIDCClient{identity: &oidcVerifiedIdentity{
		Subject:       "sub-alice",
		Nonce:         "nonce-STOLEN",
		Email:         "alice@example.com",
		EmailVerified: true,
	}}
	s := newOIDCCallbackHarness(t, stub)

	req := oidcCallbackRequest(s, t, "state-1", "state-1", "code-1")
	rr := httptest.NewRecorder()
	s.CallbackOIDCRoute(rr, req)

	assertOIDCRedirect(t, rr, "nonce_mismatch")
}

func TestCallbackOIDCRoute_ClientErrorCodes(t *testing.T) {
	// Each client failure must map to the matching frontend error code. Errors
	// are wrapped (as the real client does) to exercise the errors.Is path.
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"exchange_failed", errOIDCExchangeFailed, "exchange_failed"},
		{"no_id_token", errOIDCNoIDToken, "no_id_token"},
		{"bad_id_token", errOIDCBadIDToken, "bad_id_token"},
		{"bad_claims", errOIDCBadClaims, "bad_claims"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubOIDCClient{err: fmt.Errorf("%w: boom", tc.err)}
			s := newOIDCCallbackHarness(t, stub)

			req := oidcCallbackRequest(s, t, "state-1", "state-1", "code-1")
			rr := httptest.NewRecorder()
			s.CallbackOIDCRoute(rr, req)

			assertOIDCRedirect(t, rr, tc.want)
		})
	}
}

func TestCallbackOIDCRoute_Disabled(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	// OIDCConfig.Enabled is false by default, so the route must 404.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback", nil)
	rr := httptest.NewRecorder()
	s.CallbackOIDCRoute(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when OIDC disabled, got %d", rr.Code)
	}
}
