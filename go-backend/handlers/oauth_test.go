package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-backend/tests"
)

// These tests cover the CSRF `state` retrofit on the GitHub OAuth flow
// (see handlers/oauth_state.go for the shared signed-cookie primitive). They
// exercise the start route's cookie/URL emission and the callback's state
// validation WITHOUT hitting GitHub's API: the state checks run before the
// token exchange, so failure modes are reachable without a network mock.

// withStateKey sets a known JwtSecretKey for the duration of a test and
// restores the original afterward. The handlers read s.Server.JwtSecretKey to
// sign/verify the state cookie, but Server is the shared tests.S singleton —
// leaving a non-empty key behind breaks every other test because
// GenerateTestJWT signs tokens with an empty key while JwtMiddleware
// validates with s.Server.JwtSecretKey.
func withStateKey(t *testing.T, s *Handler) {
	t.Helper()
	prev := s.Server.JwtSecretKey
	s.Server.JwtSecretKey = []byte("test-secret-key-for-state-cookie-32+")
	t.Cleanup(func() { s.Server.JwtSecretKey = prev })
}

// assertLoginError checks the callback redirected to /login?error=<code>.
func assertLoginError(t *testing.T, rr *httptest.ResponseRecorder, code string) {
	t.Helper()
	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	loc := rr.Header().Get("Location")
	want := "/login?error=" + code
	if !strings.HasSuffix(loc, want) {
		t.Fatalf("expected redirect ending in %q, got %q", want, loc)
	}
}

func TestStartGitHubOAuthRoute_SetsStateCookieAndRedirect(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	withStateKey(t, s)

	req := httptest.NewRequest("GET", "/api/auth/github", nil)
	rr := httptest.NewRecorder()
	s.StartGitHubOAuthRoute(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://github.com/login/oauth/authorize") {
		t.Fatalf("unexpected redirect target: %s", loc)
	}
	// The authorize URL must carry a (non-empty) state parameter.
	if !strings.Contains(loc, "state=") {
		t.Fatalf("authorize URL missing state param: %s", loc)
	}

	// A signed state cookie must be set so the callback can verify the echo.
	var cookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == oidcCookieName {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("expected zg_oauth_state cookie to be set")
	}
	if cookie.Value == "" || !strings.Contains(cookie.Value, ".") {
		t.Fatalf("expected signed payload.sig cookie value, got %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("state cookie must be HttpOnly")
	}
}

func TestGitHubCallbackRoute_RejectsMissingStateCookie(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	withStateKey(t, s)

	// No state cookie at all → CSRF check fails before the code is touched.
	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=abc&state=mystate", nil)
	rr := httptest.NewRecorder()
	s.GitHubCallbackRoute(rr, req)

	assertLoginError(t, rr, "missing_state")
}

func TestGitHubCallbackRoute_RejectsTamperedStateCookie(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	withStateKey(t, s)

	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=abc&state=xyz", nil)
	req.AddCookie(&http.Cookie{Name: oidcCookieName, Value: "tampered.garbage"})
	rr := httptest.NewRecorder()
	s.GitHubCallbackRoute(rr, req)

	assertLoginError(t, rr, "bad_state")
}

func TestGitHubCallbackRoute_RejectsStateMismatch(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	withStateKey(t, s)

	// Valid, correctly-signed cookie carrying state "cookie-state", but the
	// IdP echoes a different `state` on the query string → mismatch.
	raw, err := signOAuthState(s.Server.JwtSecretKey, oauthStatePayload{State: "cookie-state"})
	if err != nil {
		t.Fatalf("signOAuthState: %v", err)
	}
	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=abc&state=different", nil)
	req.AddCookie(&http.Cookie{Name: oidcCookieName, Value: raw})
	rr := httptest.NewRecorder()
	s.GitHubCallbackRoute(rr, req)

	assertLoginError(t, rr, "state_mismatch")
}

func TestGitHubCallbackRoute_StateOKThenRequiresCode(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	withStateKey(t, s)

	// Valid cookie + matching state, but no `code`. This proves the state
	// check passed (cookie verified, state matched) and we reached the next
	// guard. No network call happens because `code` is checked first.
	raw, err := signOAuthState(s.Server.JwtSecretKey, oauthStatePayload{State: "s1"})
	if err != nil {
		t.Fatalf("signOAuthState: %v", err)
	}
	req := httptest.NewRequest("GET", "/api/auth/github/callback?state=s1", nil)
	req.AddCookie(&http.Cookie{Name: oidcCookieName, Value: raw})
	rr := httptest.NewRecorder()
	s.GitHubCallbackRoute(rr, req)

	assertLoginError(t, rr, "missing_code")
}

func TestGitHubCallbackRoute_RejectsExpiredStateCookie(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()
	withStateKey(t, s)
	key := s.Server.JwtSecretKey

	// Hand-craft an already-expired (but correctly signed) cookie so the
	// callback rejects it as bad_state rather than proceeding.
	body, _ := json.Marshal(oauthStatePayload{State: "s1", Exp: time.Now().Add(-time.Minute).Unix()})
	raw := base64.RawURLEncoding.EncodeToString(body) + "." +
		base64.RawURLEncoding.EncodeToString(hmacSum(key, body))
	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=abc&state=s1", nil)
	req.AddCookie(&http.Cookie{Name: oidcCookieName, Value: raw})
	rr := httptest.NewRecorder()
	s.GitHubCallbackRoute(rr, req)

	assertLoginError(t, rr, "bad_state")
}
