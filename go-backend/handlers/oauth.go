package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"go-backend/models"
)

type GitHubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type GitHubUser struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

func (s *Handler) StartGitHubOAuthRoute(w http.ResponseWriter, r *http.Request) {
	if !s.GitHubConfig.Enabled {
		http.NotFound(w, r)
		return
	}
	clientID := s.GitHubConfig.ClientID
	redirectURI := s.GitHubConfig.RedirectURI
	scope := "user:email"

	// CSRF defense: generate a random `state`, store it in the shared signed
	// state cookie (same primitive as the OIDC flow), and echo it on the
	// authorize request. The callback verifies GitHub returns the same value
	// before exchanging the code — see handlers/oauth_state.go.
	state, err := randomString(24)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	raw, err := signOAuthState(s.Server.JwtSecretKey, oauthStatePayload{State: state})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setOAuthStateCookie(w, raw, redirectURI)

	githubAuthURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		clientID, url.QueryEscape(redirectURI), scope, url.QueryEscape(state),
	)

	http.Redirect(w, r, githubAuthURL, http.StatusFound)
}

func (s *Handler) GitHubCallbackRoute(w http.ResponseWriter, r *http.Request) {
	if !s.GitHubConfig.Enabled {
		http.NotFound(w, r)
		return
	}
	frontendURL := os.Getenv("ZETTEL_URL")
	// Failure modes redirect back to the login page with an error code
	// (mirroring the OIDC callback) so the user lands somewhere sane. The
	// codes map to friendly messages in LoginPage.tsx.
	fail := func(code string) {
		http.Redirect(w, r, fmt.Sprintf("%s/login?error=%s", frontendURL, url.QueryEscape(code)), http.StatusFound)
	}

	// 1. Validate + consume the signed state cookie (CSRF defense, shared with
	//    the OIDC flow).
	cookie, err := r.Cookie(oidcCookieName)
	if err != nil {
		fail("missing_state")
		return
	}
	payload, err := verifyOAuthState(s.Server.JwtSecretKey, cookie.Value)
	if err != nil {
		fail("bad_state")
		return
	}
	clearOAuthStateCookie(w)

	// 2. CSRF: the state echoed by GitHub must match what we stored.
	if r.URL.Query().Get("state") != payload.State {
		fail("state_mismatch")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		fail("missing_code")
		return
	}

	clientID := s.GitHubConfig.ClientID
	clientSecret := s.GitHubConfig.ClientSecret
	redirectURI := s.GitHubConfig.RedirectURI

	body := url.Values{}
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("code", code)
	body.Set("redirect_uri", redirectURI)

	tokenReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(body.Encode()))
	tokenReq.Header.Set("Accept", "application/json")
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		fail("exchange_failed")
		return
	}
	defer resp.Body.Close()

	var tokenRes GitHubAccessTokenResponse
	json.NewDecoder(resp.Body).Decode(&tokenRes)

	if tokenRes.AccessToken == "" {
		fail("exchange_failed")
		return
	}

	// GitHub user info
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
	userRes, _ := http.DefaultClient.Do(userReq)
	defer userRes.Body.Close()

	var ghUser GitHubUser
	json.NewDecoder(userRes.Body).Decode(&ghUser)

	// Fetch emails for verified ones
	emailReq, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	emailReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
	emailRes, _ := http.DefaultClient.Do(emailReq)
	defer emailRes.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	json.NewDecoder(emailRes.Body).Decode(&emails)

	if ghUser.Email == "" {
		for _, e := range emails {
			if e.Primary && e.Verified {
				ghUser.Email = e.Email
				break
			}
		}
	}

	if ghUser.Email == "" {
		fail("no_email")
		return
	}

	// Find or create user
	user, err := s.QueryUserByEmail(ghUser.Email)
	if err != nil || user.ID == 0 {
		// no matching user, create new
		params := models.CreateUserParams{
			Username: ghUser.Login,
			Email:    ghUser.Email,
			Password: "github_oauth_" + fmt.Sprint(ghUser.ID),
		}
		newID, err := s.CreateUser(params)
		if err != nil {
			// A concurrent signup raced past the check above; the unique email
			// index makes duplicate creation impossible, so the account now
			// exists — re-resolve it and fall through to the link step below.
			user, err = s.QueryUserByEmail(ghUser.Email)
			if err != nil || user.ID == 0 {
				fail("user_resolve_failed")
				return
			}
		} else {
			user, err = s.QueryUser(newID)
			if err != nil {
				fail("user_resolve_failed")
				return
			}
		}
	}

	// Attach GitHub metadata. Guarded so an account already linked to a
	// different GitHub id keeps its original link (mirrors the pre-existing
	// behavior for existing users; a fresh user matches the guard too).
	_, err = s.DB.Exec(`UPDATE users SET auth_provider = 'github', github_id = $1 WHERE id = $2 AND (auth_provider = 'local' OR github_id IS NULL)`, ghUser.ID, user.ID)
	if err != nil {
		fail("user_resolve_failed")
		return
	}

	// Generate JWT
	token, err := s.generateAccessToken(user.ID)
	if err != nil {
		fail("jwt_failed")
		return
	}

	// Redirect back to frontend with token
	redirect := fmt.Sprintf("%s/login?token=%s", frontendURL, token)
	http.Redirect(w, r, redirect, http.StatusFound)
}
