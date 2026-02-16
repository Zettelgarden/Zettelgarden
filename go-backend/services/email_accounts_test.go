package services

import (
	"context"
	"go-backend/models"
	"go-backend/tests"
	"testing"
	"time"
)

func TestCreateEmailAccount(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user@example.com",
		ApiToken:     &password,
	}

	service := NewEmailAccountService(s.DB)
	account, err := service.CreateEmailAccount(ctx, userID, params, "test-encryption-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	if account.EmailAddress != params.EmailAddress {
		t.Errorf("expected email_address %s, got %s", params.EmailAddress, account.EmailAddress)
	}

	if account.UserID != userID {
		t.Errorf("expected user_id %d, got %d", userID, account.UserID)
	}

	// Check default values
	if account.JMAPServerURL != "https://api.fastmail.com/jmap/session" {
		t.Errorf("expected default JMAP server URL, got %s", account.JMAPServerURL)
	}

	if !account.IsActive {
		t.Error("expected is_active to be true by default")
	}

	if account.SyncStatus != "active" {
		t.Errorf("expected sync_status 'active', got %s", account.SyncStatus)
	}

	if account.ApiTokenEncrypted == nil {
		t.Error("expected api_token_encrypted to be set")
	}
}

func TestGetEmailAccounts(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Should be empty initially
	accounts, err := service.GetEmailAccounts(ctx, userID)
	if err != nil {
		t.Errorf("failed to get email accounts: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user1@example.com",
		ApiToken:     &password,
	}
	_, err = service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Should have one account now
	accounts, err = service.GetEmailAccounts(ctx, userID)
	if err != nil {
		t.Errorf("failed to get email accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(accounts))
	}
}

func TestGetEmailAccountByID(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Test not found
	_, err := service.GetEmailAccountByID(ctx, userID, 99999)
	if err == nil {
		t.Error("expected error for non-existent account")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user2@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Get the account by ID
	account, err := service.GetEmailAccountByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to get email account: %v", err)
	}

	if account.ID != created.ID {
		t.Errorf("expected id %d, got %d", created.ID, account.ID)
	}

	if account.EmailAddress != params.EmailAddress {
		t.Errorf("expected email_address %s, got %s", params.EmailAddress, account.EmailAddress)
	}
}

func TestGetEmailAccountByIDWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user3@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Try to get with different user ID
	_, err = service.GetEmailAccountByID(ctx, 999, created.ID)
	if err == nil {
		t.Error("expected error when accessing account with wrong user")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}
}

func TestDeleteEmailAccount(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Test delete non-existent
	err := service.DeleteEmailAccount(ctx, userID, 99999)
	if err == nil {
		t.Error("expected error for non-existent account")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user4@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Delete the account
	err = service.DeleteEmailAccount(ctx, userID, created.ID)
	if err != nil {
		t.Errorf("failed to delete email account: %v", err)
	}

	// Verify it's deleted
	_, err = service.GetEmailAccountByID(ctx, userID, created.ID)
	if err == nil {
		t.Error("expected error after deleting account")
	}
}

func TestDeleteEmailAccountWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user5@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Try to delete with different user ID
	err = service.DeleteEmailAccount(ctx, 999, created.ID)
	if err == nil {
		t.Error("expected error when deleting account with wrong user")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}

	// Verify original account still exists
	_, err = service.GetEmailAccountByID(ctx, userID, created.ID)
	if err != nil {
		t.Errorf("account should still exist: %v", err)
	}
}

func TestUpdateLastSync(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user6@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Update last sync time
	syncTime := time.Now().UTC()
	err = service.UpdateLastSync(ctx, userID, created.ID, syncTime)
	if err != nil {
		t.Errorf("failed to update last sync: %v", err)
	}

	// Verify the update
	account, err := service.GetEmailAccountByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to get email account: %v", err)
	}

	if account.LastSyncAt == nil {
		t.Error("expected last_sync_at to be set")
	}
}

func TestUpdateLastSyncWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user6b@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Try to update with wrong user ID
	syncTime := time.Now().UTC()
	err = service.UpdateLastSync(ctx, 999, created.ID, syncTime)
	if err == nil {
		t.Error("expected error when updating with wrong user")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}
}

func TestUpdateJMAPState(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user7@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Update JMAP state
	state := "test-state-123"
	err = service.UpdateJMAPState(ctx, userID, created.ID, state)
	if err != nil {
		t.Errorf("failed to update JMAP state: %v", err)
	}

	// Verify the update
	account, err := service.GetEmailAccountByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to get email account: %v", err)
	}

	if account.JMAPState == nil {
		t.Error("expected jmap_state to be set")
	} else if *account.JMAPState != state {
		t.Errorf("expected jmap_state %s, got %s", state, *account.JMAPState)
	}
}

func TestUpdateJMAPStateWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user7b@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Try to update with wrong user ID
	err = service.UpdateJMAPState(ctx, 999, created.ID, "test-state")
	if err == nil {
		t.Error("expected error when updating with wrong user")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}
}

func TestUpdateSyncStatus(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user8@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Update sync status
	err = service.UpdateSyncStatus(ctx, userID, created.ID, "error")
	if err != nil {
		t.Errorf("failed to update sync status: %v", err)
	}

	// Verify the update
	account, err := service.GetEmailAccountByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to get email account: %v", err)
	}

	if account.SyncStatus != "error" {
		t.Errorf("expected sync_status 'error', got %s", account.SyncStatus)
	}
}

func TestUpdateSyncStatusWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user8b@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Try to update with wrong user ID
	err = service.UpdateSyncStatus(ctx, 999, created.ID, "error")
	if err == nil {
		t.Error("expected error when updating with wrong user")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}
}

func TestUpdateEmailAccount(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user9@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Update with no changes (empty params)
	updated, err := service.UpdateEmailAccount(ctx, userID, created.ID, models.UpdateEmailAccountParams{})
	if err != nil {
		t.Errorf("failed to update with empty params: %v", err)
	}
	if updated.SyncStatus != "active" {
		t.Errorf("expected sync_status to remain 'active', got %s", updated.SyncStatus)
	}

	// Update IsActive to false
	isActive := false
	updateParams := models.UpdateEmailAccountParams{
		IsActive: &isActive,
	}
	updated, err = service.UpdateEmailAccount(ctx, userID, created.ID, updateParams)
	if err != nil {
		t.Errorf("failed to update is_active: %v", err)
	}
	if updated.IsActive {
		t.Error("expected is_active to be false")
	}

	// Update SyncStatus
	syncStatus := "paused"
	updateParams = models.UpdateEmailAccountParams{
		SyncStatus: &syncStatus,
	}
	updated, err = service.UpdateEmailAccount(ctx, userID, created.ID, updateParams)
	if err != nil {
		t.Errorf("failed to update sync_status: %v", err)
	}
	if updated.SyncStatus != "paused" {
		t.Errorf("expected sync_status 'paused', got %s", updated.SyncStatus)
	}

	// Update both fields
	isActive = true
	syncStatus = "active"
	updateParams = models.UpdateEmailAccountParams{
		IsActive:   &isActive,
		SyncStatus: &syncStatus,
	}
	updated, err = service.UpdateEmailAccount(ctx, userID, created.ID, updateParams)
	if err != nil {
		t.Errorf("failed to update both fields: %v", err)
	}
	if !updated.IsActive {
		t.Error("expected is_active to be true")
	}
	if updated.SyncStatus != "active" {
		t.Errorf("expected sync_status 'active', got %s", updated.SyncStatus)
	}
}

func TestUpdateEmailAccountWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user9b@example.com",
		ApiToken:     &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create email account: %v", err)
	}

	// Try to update with wrong user ID
	isActive := false
	updateParams := models.UpdateEmailAccountParams{
		IsActive: &isActive,
	}
	_, err = service.UpdateEmailAccount(ctx, 999, created.ID, updateParams)
	if err == nil {
		t.Error("expected error when updating with wrong user")
	}
	if err != nil && err.Error() != "email account not found" {
		t.Errorf("expected 'email account not found' error, got: %v", err)
	}
}

func TestEncryptAppPassword(t *testing.T) {
	password := "my-secret-password"
	key := "encryption-key"

	encrypted, err := encryptApiToken(password, key)
	if err != nil {
		t.Errorf("failed to encrypt password: %v", err)
	}

	if encrypted == "" {
		t.Error("expected encrypted password to not be empty")
	}

	// Decrypt and verify
	decrypted, err := decryptApiToken(encrypted, key)
	if err != nil {
		t.Errorf("failed to decrypt password: %v", err)
	}

	if decrypted != password {
		t.Errorf("expected decrypted password %s, got %s", password, decrypted)
	}
}

func TestDecryptAppPasswordExported(t *testing.T) {
	password := "my-secret-password"
	key := "encryption-key"

	encrypted, err := encryptApiToken(password, key)
	if err != nil {
		t.Errorf("failed to encrypt password: %v", err)
	}

	// Test the exported function
	decrypted, err := DecryptApiToken(encrypted, key)
	if err != nil {
		t.Errorf("failed to decrypt password with exported function: %v", err)
	}

	if decrypted != password {
		t.Errorf("expected decrypted password %s, got %s", password, decrypted)
	}
}
