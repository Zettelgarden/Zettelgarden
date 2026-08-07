package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBillingRoutesDisabled verifies that every billing route returns 404 when
// Stripe billing is switched off (STRIPE_ENABLED=false), and that the status
// endpoint still answers with {enabled:false} so the frontend can react.
func TestBillingRoutesDisabled(t *testing.T) {
	s := NewHandler()
	s.StripeConfig.Enabled = false

	cases := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"subscribe", http.MethodPost, "/api/billing/subscribe", s.CreateSubscriptionRoute},
		{"portal", http.MethodGet, "/api/billing/portal", s.BillingPortalRoute},
		{"public-key", http.MethodGet, "/api/billing/public-key", s.StripePublicKeyRoute},
		{"webhook", http.MethodPost, "/api/stripe/webhook", s.StripeWebhookRoute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			tc.handler(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("expected 404 when billing disabled, got %d", rr.Code)
			}
		})
	}

	// Status route still answers (protected), reporting disabled.
	req := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)
	rr := httptest.NewRecorder()
	s.BillingStatusRoute(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from status route, got %d", rr.Code)
	}
	var resp BillingStatusResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if resp.Enabled {
		t.Fatalf("expected enabled=false when billing disabled, got true")
	}
}

// TestBillingStatusEnabled verifies the status endpoint reports enabled when
// Stripe billing is on (the default).
func TestBillingStatusEnabled(t *testing.T) {
	s := NewHandler()
	if !s.StripeConfig.Enabled {
		t.Fatal("expected billing enabled by default in tests")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)
	rr := httptest.NewRecorder()
	s.BillingStatusRoute(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from status route, got %d", rr.Code)
	}
	var resp BillingStatusResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if !resp.Enabled {
		t.Fatalf("expected enabled=true when billing enabled, got false")
	}
}
