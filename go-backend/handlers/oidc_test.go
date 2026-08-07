package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"go-backend/tests"
	"testing"
)

// insertTestUser inserts a bare, unlinked (oidc_sub NULL) user row for OIDC
// linking tests and returns its id.
func insertTestUser(t *testing.T, s *Handler, email string, emailValidated bool) int {
	t.Helper()
	var id int
	err := s.GetDB().QueryRow(`
		INSERT INTO users (username, email, password, created_at, updated_at,
			stripe_subscription_status, email_validated, auth_provider)
		VALUES ($1, $2, 'x', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'free', $3, 'local')
		RETURNING id`,
		"u-"+email, email, emailValidated,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insertTestUser: %v", err)
	}
	return id
}

func TestPkceS256Challenge(t *testing.T) {
	// Known vector: S256 challenge is base64url(sha256(verifier)), no padding.
	// 'verifier' -> sha256 -> base64url. Verified independently.
	got := pkceS256Challenge("verifier")
	// Recompute expected here to keep the test self-contained.
	sum := sha256.Sum256([]byte("verifier"))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("pkceS256Challenge: got %q want %q", got, want)
	}
	// Must be base64url with no padding and not equal to the raw verifier.
	if got == "verifier" {
		t.Fatal("challenge must not equal the verifier")
	}
	if len(got) < 40 {
		t.Fatalf("challenge looks too short: %q", got)
	}
}

func TestFindOrCreateOIDCUser_CreatesNewAccount(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	user, err := s.findOrCreateOIDCUser("pocket-id", "sub-new-1", "alice@example.com", true, "alice", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected non-zero user id")
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", user.Email)
	}
	if user.Username != "alice" {
		t.Fatalf("expected username alice, got %s", user.Username)
	}

	// New OIDC accounts are considered email-validated by the IdP.
	var validated bool
	if err := s.GetDB().QueryRow(`SELECT email_validated FROM users WHERE id = $1`, user.ID).Scan(&validated); err != nil {
		t.Fatalf("query validated: %v", err)
	}
	if !validated {
		t.Fatal("expected email_validated=true for new OIDC account")
	}
}

func TestFindOrCreateOIDCUser_MatchBySubReauth(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// First login creates the account.
	first, err := s.findOrCreateOIDCUser("pocket-id", "sub-reauth", "bob@example.com", true, "bob", "")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	// Second login with the same sub (email changed / unverified) must return
	// the SAME account via the stable (provider, sub) match.
	second, err := s.findOrCreateOIDCUser("pocket-id", "sub-reauth", "changed@example.com", false, "", "")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected stable re-auth to same user: %d vs %d", first.ID, second.ID)
	}
	if second.Email != "bob@example.com" {
		t.Fatalf("expected original email retained, got %s", second.Email)
	}
}

func TestFindOrCreateOIDCUser_AutoLinkVerifiedEmail(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Pre-existing password user, not yet linked to OIDC.
	existingID := insertTestUser(t, s, "carol@example.com", true)

	user, err := s.findOrCreateOIDCUser("pocket-id", "sub-carol", "carol@example.com", true, "carol", "")
	if err != nil {
		t.Fatalf("auto-link: %v", err)
	}
	if user.ID != existingID {
		t.Fatalf("expected link to existing user %d, got %d", existingID, user.ID)
	}

	// The existing account is now linked.
	var provider, sub string
	err = s.GetDB().QueryRow(`SELECT oidc_provider, oidc_sub FROM users WHERE id = $1`, existingID).Scan(&provider, &sub)
	if err != nil {
		t.Fatalf("query link: %v", err)
	}
	if provider != "pocket-id" || sub != "sub-carol" {
		t.Fatalf("expected link stored, got provider=%q sub=%q", provider, sub)
	}
}

func TestFindOrCreateOIDCUser_RejectsUnverifiedEmailLink(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Pre-existing password user.
	existingID := insertTestUser(t, s, "dave@example.com", true)

	// Unverified email that matches the existing account must NOT link and
	// must NOT create a duplicate.
	_, err := s.findOrCreateOIDCUser("pocket-id", "sub-dave", "dave@example.com", false, "dave", "")
	if err == nil {
		t.Fatal("expected error for unverified email matching existing account")
	}

	// Confirm no duplicate account was created and the existing one is untouched.
	var count int
	if err := s.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE email = $1`, "dave@example.com").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 account for dave@example.com, got %d (duplicate created?)", count)
	}
	var provider, sub string
	_ = s.GetDB().QueryRow(`SELECT oidc_provider, oidc_sub FROM users WHERE id = $1`, existingID).Scan(&provider, &sub)
	if provider != "" || sub != "" {
		t.Fatalf("expected existing account unlinked, got provider=%q sub=%q", provider, sub)
	}
}

func TestFindOrCreateOIDCUser_AlreadyLinkedDifferentSub(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	// Account already linked to provider-a / sub-a.
	existingID := insertTestUser(t, s, "erin@example.com", true)
	if _, err := s.GetDB().Exec(
		`UPDATE users SET oidc_provider = 'provider-a', oidc_sub = 'sub-a' WHERE id = $1`,
		existingID,
	); err != nil {
		t.Fatalf("pre-link user: %v", err)
	}

	// A second provider presenting the same verified email must NOT silently
	// overwrite the existing link: the link UPDATE affects 0 rows and the
	// existing user is returned unchanged (the odd state is logged, not
	// acted on).
	user, err := s.findOrCreateOIDCUser("provider-b", "sub-b", "erin@example.com", true, "erin", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != existingID {
		t.Fatalf("expected existing user %d, got %d", existingID, user.ID)
	}

	// The original link must be preserved (no re-link happened).
	var provider, sub string
	if err := s.GetDB().QueryRow(`SELECT oidc_provider, oidc_sub FROM users WHERE id = $1`, existingID).Scan(&provider, &sub); err != nil {
		t.Fatalf("query link: %v", err)
	}
	if provider != "provider-a" || sub != "sub-a" {
		t.Fatalf("expected original link preserved, got provider=%q sub=%q", provider, sub)
	}
}

func TestFindOrCreateOIDCUser_MissingSubject(t *testing.T) {
	s := NewHandler()
	defer tests.Teardown()

	if _, err := s.findOrCreateOIDCUser("pocket-id", "", "x@example.com", true, "", ""); err == nil {
		t.Fatal("expected error for missing subject")
	}
}
