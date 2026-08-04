package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"go-backend/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// oidcProviderLabel is the value stored in users.oidc_provider. Kept constant
// for the single-provider first cut; a multi-provider rollout would derive
// this from a request param / config.
const oidcProviderLabel = "pocket-id"

// boolish tolerates email_verified arriving as a JSON bool or a quoted string
// ("true"/"false"), which keeps us robust to provider quirks.
type boolish bool

func (b *boolish) UnmarshalJSON(data []byte) error {
	switch strings.TrimSpace(string(data)) {
	case "true", "1", `"true"`:
		*b = true
	case "false", "0", `"false"`, `""`, "":
		*b = false
	default:
		return fmt.Errorf("invalid boolean value: %s", string(data))
	}
	return nil
}

// pkceS256Challenge returns the S256 code_challenge for a code_verifier:
// base64url(SHA-256(verifier)) with no padding. Sent in the authorize request;
// the verifier itself is only sent later in the token exchange.
func pkceS256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// getOIDCProvider lazily discovers (via .well-known/openid-configuration) and
// caches the Provider + oauth2.Config. The cache is process-local and never
// invalidated; changing OIDC_* env vars requires a restart.
func (h *Handler) getOIDCProvider(ctx context.Context) (*oidc.Provider, *oauth2.Config, error) {
	h.oidcInitMu.Lock()
	defer h.oidcInitMu.Unlock()
	if h.oidcProvider != nil && h.oidcOAuth2 != nil {
		return h.oidcProvider, h.oidcOAuth2, nil
	}
	provider, err := oidc.NewProvider(ctx, h.OIDCConfig.Issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc discovery for %q: %w", h.OIDCConfig.Issuer, err)
	}
	cfg := &oauth2.Config{
		ClientID:     h.OIDCConfig.ClientID,
		ClientSecret: h.OIDCConfig.ClientSecret,
		RedirectURL:  h.OIDCConfig.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
	h.oidcProvider = provider
	h.oidcOAuth2 = cfg
	return provider, cfg, nil
}

// StartOIDCRoute begins the OIDC Authorization Code (+PKCE) flow:
// generates state/nonce/verifier, stores them in a signed cookie, and
// redirects the browser to the IdP authorization endpoint.
func (h *Handler) StartOIDCRoute(w http.ResponseWriter, r *http.Request) {
	if !h.OIDCConfig.Enabled {
		http.NotFound(w, r)
		return
	}
	_, oauth2Config, err := h.getOIDCProvider(r.Context())
	if err != nil {
		log.Printf("oidc start: %v", err)
		http.Error(w, "OIDC unavailable", http.StatusServiceUnavailable)
		return
	}

	state, err := randomString(24)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonce, err := randomString(24)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()

	raw, err := signOAuthState(h.Server.JwtSecretKey, oauthStatePayload{
		State: state,
		Nonce: nonce,
		Ver:   verifier,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setOAuthStateCookie(w, raw, h.OIDCConfig.RedirectURI)

	authURL := oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceS256Challenge(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackOIDCRoute completes the OIDC flow: validates the signed state
// cookie, exchanges the code (PKCE), verifies the id_token (JWKS + nonce),
// resolves or creates the local user, then redirects to the frontend with an
// app JWT — identical to the GitHub OAuth path.
func (h *Handler) CallbackOIDCRoute(w http.ResponseWriter, r *http.Request) {
	if !h.OIDCConfig.Enabled {
		http.NotFound(w, r)
		return
	}
	frontendURL := os.Getenv("ZETTEL_URL")
	// All failure modes redirect back to the login page with an error code
	// rather than a raw HTTP error, so the user lands somewhere sane.
	fail := func(code string) {
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=%s", frontendURL, url.QueryEscape(code)), http.StatusFound)
	}

	// 1. Validate + consume the signed state cookie.
	cookie, err := r.Cookie(oidcCookieName)
	if err != nil {
		fail("missing_state")
		return
	}
	payload, err := verifyOAuthState(h.Server.JwtSecretKey, cookie.Value)
	if err != nil {
		fail("bad_state")
		return
	}
	clearOAuthStateCookie(w)

	// 2. CSRF: the state echoed by the IdP must match what we stored.
	if r.URL.Query().Get("state") != payload.State {
		fail("state_mismatch")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		fail("missing_code")
		return
	}

	provider, oauth2Config, err := h.getOIDCProvider(r.Context())
	if err != nil {
		fail("oidc_unavailable")
		return
	}

	// 3. Exchange the authorization code, sending the PKCE verifier.
	token, err := oauth2Config.Exchange(r.Context(), code, oauth2.VerifierOption(payload.Ver))
	if err != nil {
		log.Printf("oidc token exchange failed: %v", err)
		fail("exchange_failed")
		return
	}

	// 4. Verify the id_token: signature (JWKS), iss, aud, exp — done by the
	//    provider verifier pinned to our client_id.
	verifier := provider.Verifier(&oidc.Config{ClientID: h.OIDCConfig.ClientID})
	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		fail("no_id_token")
		return
	}
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("oidc id_token verification failed: %v", err)
		fail("bad_id_token")
		return
	}

	// 5. Replay defense: nonce from the auth request must be in the id_token.
	if idToken.Nonce != payload.Nonce {
		fail("nonce_mismatch")
		return
	}

	// 6. Extract identity claims.
	var claims struct {
		Email             string  `json:"email"`
		EmailVerified     boolish `json:"email_verified"`
		Name              string  `json:"name"`
		PreferredUsername string  `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("oidc claims parse failed: %v", err)
		fail("bad_claims")
		return
	}

	// 7. Resolve or create the local user.
	user, err := h.findOrCreateOIDCUser(
		oidcProviderLabel,
		idToken.Subject,
		claims.Email,
		bool(claims.EmailVerified),
		claims.PreferredUsername,
		claims.Name,
	)
	if err != nil {
		log.Printf("oidc user resolution failed: %v", err)
		fail("user_resolve_failed")
		return
	}

	// 8. Issue our app JWT and redirect to the frontend (same path as GitHub).
	jwtToken, err := h.generateAccessToken(user.ID)
	if err != nil {
		fail("jwt_failed")
		return
	}
	h.LogLastLogin(user)
	http.Redirect(w, r, fmt.Sprintf("%s/login?token=%s", frontendURL, jwtToken), http.StatusFound)
}

// findOrCreateOIDCUser resolves the local account for an OIDC login in order:
//  1. (oidc_provider, oidc_sub) — stable per-user, per-IdP re-authentication
//  2. email — only when the IdP asserts email_verified (auto-link decision).
//     If the email matches but is NOT verified, we refuse (no link, no
//     duplicate) so an unverified-email IdP cannot take over an account.
//  3. create a new account (email_validated=true, unusable random password)
//
// It goes through GetDB(): in production the OIDC callback has no request
// transaction so this returns the connection pool (identical to writing on
// h.DB directly, like the GitHub/Stripe callbacks); in tests it returns the
// per-test transaction for isolation.
func (h *Handler) findOrCreateOIDCUser(provider, subject, email string, emailVerified bool, preferredUsername, name string) (models.User, error) {
	if subject == "" {
		return models.User{}, fmt.Errorf("missing oidc subject")
	}

	// 1. Match by stable (provider, sub).
	var id int
	err := h.GetDB().QueryRow(
		`SELECT id FROM users WHERE oidc_provider = $1 AND oidc_sub = $2`,
		provider, subject,
	).Scan(&id)
	if err == nil {
		return h.QueryUser(id)
	}
	if err != sql.ErrNoRows {
		return models.User{}, fmt.Errorf("lookup by oidc_sub: %w", err)
	}

	// 2. Email match: link only when the IdP verified the email. If the email
	//    matches an existing account but is unverified, refuse rather than
	//    create a duplicate or silently link.
	if email != "" {
		var existingID int
		lookupErr := h.GetDB().QueryRow(`SELECT id FROM users WHERE email = $1`, email).Scan(&existingID)
		if lookupErr == nil {
			if !emailVerified {
				return models.User{}, fmt.Errorf("email matches existing account but IdP did not verify it")
			}
			_, err := h.GetDB().Exec(
				`UPDATE users SET oidc_provider = $1, oidc_sub = $2, email_validated = TRUE WHERE id = $3 AND (oidc_sub IS NULL OR oidc_sub = '')`,
				provider, subject, existingID,
			)
			if err != nil {
				return models.User{}, fmt.Errorf("link existing user: %w", err)
			}
			return h.QueryUser(existingID)
		}
		if lookupErr != sql.ErrNoRows {
			return models.User{}, fmt.Errorf("lookup by email: %w", lookupErr)
		}
	}

	// 3. Create a new account.
	if email == "" {
		return models.User{}, fmt.Errorf("oidc provider returned no email; cannot create account")
	}
	username := preferredUsername
	if username == "" {
		username = name
	}
	if username == "" {
		username = email
	}

	// Random, unusable password — OIDC users never log in with a password.
	rnd := make([]byte, 32)
	if _, err := rand.Read(rnd); err != nil {
		return models.User{}, fmt.Errorf("generate password: %w", err)
	}
	hashed, err := hashPassword(base64.RawURLEncoding.EncodeToString(rnd))
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}

	err = h.GetDB().QueryRow(`
		INSERT INTO users (username, email, password, created_at, updated_at,
			stripe_subscription_status, email_validated, auth_provider,
			oidc_provider, oidc_sub)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'free', TRUE,
			'oidc', $4, $5) RETURNING id`,
		username, email, hashed, provider, subject,
	).Scan(&id)
	if err != nil {
		return models.User{}, fmt.Errorf("create oidc user: %w", err)
	}

	// Mirror CreateUser's onboarding defaults (best-effort, non-fatal).
	if err := h.createDefaultCards(id); err != nil {
		log.Printf("oidc: error creating default cards for user %d: %v", id, err)
	}
	if err := h.createDefaultTags(id); err != nil {
		log.Printf("oidc: error creating default tags for user %d: %v", id, err)
	}
	return h.QueryUser(id)
}
