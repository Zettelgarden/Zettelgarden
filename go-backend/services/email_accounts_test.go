package services

import (
	"context"
	"go-backend/models"
	"go-backend/tests"
	"testing"
)

func TestCreateEmailAccount(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1
	password := "test-app-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "user@example.com",
		AppPassword:  &password,
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
	if account.IMAPServer != "imap.fastmail.com:993" {
		t.Errorf("expected default IMAP server, got %s", account.IMAPServer)
	}

	if account.IMAPServerType != "imap" {
		t.Errorf("expected server type 'imap', got %s", account.IMAPServerType)
	}

	if !account.IsActive {
		t.Error("expected is_active to be true by default")
	}

	if account.SyncStatus != "active" {
		t.Errorf("expected sync_status 'active', got %s", account.SyncStatus)
	}

	if account.AppPasswordEncrypted == nil {
		t.Error("expected app_password_encrypted to be set")
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
		AppPassword:  &password,
	}
	_, err = service.CreateEmailAccount(ctx, userID, params, "test-key")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// Should have one account now
	accounts, err = service.GetEmailAccounts(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get email accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	// Check the account
	account := accounts[0]
	if account.EmailAddress != "user1@example.com" {
		t.Errorf("expected email_address user1@example.com, got %s", account.EmailAddress)
	}

	if account.SyncStatus != "active" {
		t.Errorf("expected sync_status 'active', got %s", account.SyncStatus)
	}
}

func TestGetEmailAccountByID(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account first
	password := "my-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "test@example.com",
		AppPassword:  &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "key")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// Get the account by ID
	account, err := service.GetEmailAccountByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to get email account by ID: %v", err)
	}

	if account.ID != created.ID {
		t.Errorf("expected id %d, got %d", created.ID, account.ID)
	}

	if account.EmailAddress != "test@example.com" {
		t.Errorf("expected email_address test@example.com, got %s", account.EmailAddress)
	}
}

func TestDeleteEmailAccount(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	service := NewEmailAccountService(s.DB)

	// Create an account
	password := "my-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "to-delete@example.com",
		AppPassword:  &password,
	}
	created, err := service.CreateEmailAccount(ctx, userID, params, "key")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// Delete the account
	err = service.DeleteEmailAccount(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to delete email account: %v", err)
	}

	// Verify it's gone
	_, err = service.GetEmailAccountByID(ctx, userID, created.ID)
	if err == nil {
		t.Error("expected error when getting deleted account")
	}

	if err.Error() != "email account not found" {
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
	password := "my-password"
	params := models.CreateEmailAccountParams{
		EmailAddress: "test@example.com",
		AppPassword:  &password,
	}
	account, err := service.CreateEmailAccount(ctx, userID, params, "key")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// Deactivate the account
	isActive := false
	updateParams := models.UpdateEmailAccountParams{
		IsActive: &isActive,
	}

	updated, err := service.UpdateEmailAccount(ctx, userID, account.ID, updateParams)
	if err != nil {
		t.Fatalf("failed to update email account: %v", err)
	}

	if updated.IsActive {
		t.Error("expected account to be deactivated")
	}
}

func TestEncryptAppPassword(t *testing.T) {
	password := "my-secret-password"
	key := "encryption-key"

	encrypted, err := encryptAppPassword(password, key)
	if err != nil {
		t.Errorf("failed to encrypt password: %v", err)
	}

	if encrypted == "" {
		t.Error("expected encrypted password to not be empty")
	}

	// Decrypt and verify
	decrypted, err := decryptAppPassword(encrypted, key)
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

	encrypted, err := encryptAppPassword(password, key)
	if err != nil {
		t.Errorf("failed to encrypt password: %v", err)
	}

	// Test the exported function
	decrypted, err := DecryptAppPassword(encrypted, key)
	if err != nil {
		t.Errorf("failed to decrypt password with exported function: %v", err)
	}

	if decrypted != password {
		t.Errorf("expected decrypted password %s, got %s", password, decrypted)
	}
}
