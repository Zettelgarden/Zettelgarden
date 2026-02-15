package services

import (
	"context"
	"go-backend/models"
	"go-backend/tests"
	"testing"
	"time"
)

func TestCreateEmail(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Create first email
	email1 := models.Email{
		UserID:      userID,
		MessageID:   "test-message-1@example.com",
		Subject:     strPtr("Test Subject 1"),
		FromAddress: strPtr("sender@example.com"),
		FromName:    strPtr("Test Sender"),
		BodyText:    strPtr("This is a test email body"),
		Folder:      strPtr("Inbox"),
		Status:      "unprocessed",
	}

	created, err := service.CreateEmail(ctx, email1)
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	if created.MessageID != email1.MessageID {
		t.Errorf("expected message_id %s, got %s", email1.MessageID, created.MessageID)
	}

	if created.UserID != userID {
		t.Errorf("expected user_id %d, got %d", userID, created.UserID)
	}

	if created.Subject == nil || *created.Subject != *email1.Subject {
		t.Errorf("expected subject %s, got %v", *email1.Subject, created.Subject)
	}

	if created.FromAddress == nil || *created.FromAddress != *email1.FromAddress {
		t.Errorf("expected from_address %s, got %v", *email1.FromAddress, created.FromAddress)
	}

	if created.Status != "unprocessed" {
		t.Errorf("expected status 'unprocessed', got %s", created.Status)
	}

	if created.Folder == nil || *created.Folder != "Inbox" {
		t.Errorf("expected folder 'Inbox', got %v", created.Folder)
	}
}

func TestCreateEmailUpsert(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Create initial email
	email1 := models.Email{
		UserID:      userID,
		MessageID:   "upsert-test@example.com",
		Subject:     strPtr("Original Subject"),
		FromAddress: strPtr("sender@example.com"),
		FromName:    strPtr("Test Sender"),
		BodyText:    strPtr("Original body"),
		Folder:      strPtr("Inbox"),
		Status:      "unprocessed",
	}

	created, err := service.CreateEmail(ctx, email1)
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	// Update with same message_id (should upsert)
	email2 := models.Email{
		UserID:      userID,
		MessageID:   "upsert-test@example.com", // Same message ID
		Subject:     strPtr("Updated Subject"),
		FromAddress: strPtr("sender@example.com"),
		FromName:    strPtr("Updated Sender"),
		BodyText:    strPtr("Updated body"),
		BodyHTML:    strPtr("<p>Updated HTML</p>"),
		Folder:      strPtr("Archive"),
		Status:      "triaged",
	}

	updated, err := service.CreateEmail(ctx, email2)
	if err != nil {
		t.Fatalf("failed to upsert email: %v", err)
	}

	// Should have updated the existing record
	if updated.ID != created.ID {
		t.Errorf("expected same ID after upsert, got %d vs %d", created.ID, updated.ID)
	}

	if updated.Subject == nil || *updated.Subject != "Updated Subject" {
		t.Errorf("expected subject to be updated to 'Updated Subject', got %v", updated.Subject)
	}

	if updated.FromName == nil || *updated.FromName != "Updated Sender" {
		t.Errorf("expected from_name to be updated to 'Updated Sender', got %v", updated.FromName)
	}

	if updated.BodyHTML == nil || *updated.BodyHTML != "<p>Updated HTML</p>" {
		t.Errorf("expected body_html to be updated, got %v", updated.BodyHTML)
	}

	if updated.Folder == nil || *updated.Folder != "Archive" {
		t.Errorf("expected folder to be updated to 'Archive', got %v", updated.Folder)
	}

	// Status should NOT be updated during upsert (per requirements)
	if updated.Status != "unprocessed" {
		t.Errorf("expected status to remain 'unprocessed' after upsert, got %s", updated.Status)
	}

	// Verify only one record exists
	emails, _, err := service.ListEmails(ctx, userID, models.EmailListFilters{})
	if err != nil {
		t.Fatalf("failed to list emails: %v", err)
	}

	if len(emails) != 1 {
		t.Errorf("expected 1 email after upsert, got %d", len(emails))
	}
}

func TestListEmails(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Create test emails
	email1 := models.Email{
		UserID:      userID,
		MessageID:   "list-test-1@example.com",
		Subject:     strPtr("Subject 1"),
		FromAddress: strPtr("sender1@example.com"),
		BodyText:    strPtr("Body 1"),
		Folder:      strPtr("Inbox"),
		Status:      "unprocessed",
	}
	email2 := models.Email{
		UserID:      userID,
		MessageID:   "list-test-2@example.com",
		Subject:     strPtr("Subject 2"),
		FromAddress: strPtr("sender2@example.com"),
		BodyText:    strPtr("Body 2"),
		Folder:      strPtr("Inbox"),
		Status:      "triaged",
	}
	email3 := models.Email{
		UserID:      userID,
		MessageID:   "list-test-3@example.com",
		Subject:     strPtr("Subject 3"),
		FromAddress: strPtr("sender3@example.com"),
		BodyText:    strPtr("Body 3"),
		Folder:      strPtr("Archive"),
		Status:      "unprocessed",
	}

	for _, email := range []models.Email{email1, email2, email3} {
		_, err := service.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("failed to create email: %v", err)
		}
	}

	// List all emails
	emails, total, err := service.ListEmails(ctx, userID, models.EmailListFilters{})
	if err != nil {
		t.Fatalf("failed to list emails: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total count 3, got %d", total)
	}

	if len(emails) != 3 {
		t.Errorf("expected 3 emails, got %d", len(emails))
	}

	// Test status filter
	statusFilter := "unprocessed"
	emails, total, err = service.ListEmails(ctx, userID, models.EmailListFilters{Status: &statusFilter})
	if err != nil {
		t.Fatalf("failed to list emails with status filter: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total count 2 for status filter, got %d", total)
	}

	if len(emails) != 2 {
		t.Errorf("expected 2 emails with status 'unprocessed', got %d", len(emails))
	}

	// Test folder filter
	folderFilter := "Inbox"
	emails, total, err = service.ListEmails(ctx, userID, models.EmailListFilters{Folder: &folderFilter})
	if err != nil {
		t.Fatalf("failed to list emails with folder filter: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total count 2 for folder filter, got %d", total)
	}

	// Test combined filters
	emails, total, err = service.ListEmails(ctx, userID, models.EmailListFilters{
		Status: &statusFilter,
		Folder: &folderFilter,
	})
	if err != nil {
		t.Fatalf("failed to list emails with combined filters: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total count 1 for combined filters, got %d", total)
	}

	if len(emails) != 1 {
		t.Errorf("expected 1 email with combined filters, got %d", len(emails))
	}

	// Test pagination
	limit := 2
	emails, total, err = service.ListEmails(ctx, userID, models.EmailListFilters{Limit: &limit})
	if err != nil {
		t.Fatalf("failed to list emails with limit: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total count 3 with limit, got %d", total)
	}

	if len(emails) != 2 {
		t.Errorf("expected 2 emails with limit, got %d", len(emails))
	}

	// Test offset
	offset := 1
	emails, total, err = service.ListEmails(ctx, userID, models.EmailListFilters{
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		t.Fatalf("failed to list emails with offset: %v", err)
	}

	if len(emails) != 2 {
		t.Errorf("expected 2 emails with offset, got %d", len(emails))
	}

	// Verify offset skipped first email (email1 should be at index 0 in the first page,
	// so with offset=1, we should get email2 and email3)
	// Note: ordering is by received_at DESC NULLS LAST, then created_at DESC
	// Since received_at is null for all, order is by created_at DESC (newest first)
	// So email3 (created last) is first, email2 is second, email1 (created first) is last
	if len(emails) > 0 && emails[0].MessageID == email3.MessageID {
		t.Error("expected offset to skip first email (email3)")
	}
}

func TestListEmailsDefaults(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Test with no filters (should use defaults)
	emails, total, err := service.ListEmails(ctx, userID, models.EmailListFilters{})
	if err != nil {
		t.Fatalf("failed to list emails: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total count 0, got %d", total)
	}

	if len(emails) != 0 {
		t.Errorf("expected 0 emails, got %d", len(emails))
	}
}

func TestGetEmailByID(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Test not found
	_, err := service.GetEmailByID(ctx, userID, 99999)
	if err == nil {
		t.Error("expected error for non-existent email")
	}
	if err != nil && err.Error() != "email not found" {
		t.Errorf("expected 'email not found' error, got: %v", err)
	}

	// Create an email
	email := models.Email{
		UserID:      userID,
		MessageID:   "get-by-id-test@example.com",
		Subject:     strPtr("Get By ID Test"),
		FromAddress: strPtr("sender@example.com"),
		FromName:    strPtr("Test Sender"),
		BodyText:    strPtr("Test body"),
		Folder:      strPtr("Inbox"),
		Status:      "unprocessed",
	}

	created, err := service.CreateEmail(ctx, email)
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	// Get by ID
	found, err := service.GetEmailByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("failed to get email by ID: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("expected id %d, got %d", created.ID, found.ID)
	}

	if found.MessageID != email.MessageID {
		t.Errorf("expected message_id %s, got %s", email.MessageID, found.MessageID)
	}

	if found.Subject == nil || *found.Subject != *email.Subject {
		t.Errorf("expected subject %s, got %v", *email.Subject, found.Subject)
	}
}

func TestGetEmailByIDWrongUser(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Create an email
	email := models.Email{
		UserID:      userID,
		MessageID:   "wrong-user-test@example.com",
		Subject:     strPtr("Wrong User Test"),
		FromAddress: strPtr("sender@example.com"),
		BodyText:    strPtr("Test body"),
		Folder:      strPtr("Inbox"),
		Status:      "unprocessed",
	}

	created, err := service.CreateEmail(ctx, email)
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	// Try to get with different user ID
	_, err = service.GetEmailByID(ctx, 999, created.ID)
	if err == nil {
		t.Error("expected error when accessing email with wrong user")
	}
	if err != nil && err.Error() != "email not found" {
		t.Errorf("expected 'email not found' error, got: %v", err)
	}
}

func TestGetEmailStats(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Initially should be empty
	stats, err := service.GetEmailStats(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get email stats: %v", err)
	}

	if stats == nil {
		t.Error("expected stats map, got nil")
	}

	if len(stats) != 0 {
		t.Errorf("expected empty stats initially, got %d entries", len(stats))
	}

	// Create test emails with different statuses
	statuses := []string{"unprocessed", "unprocessed", "triaged", "reviewed", "archived"}
	messageIDs := []string{"stats-test-1@example.com", "stats-test-2@example.com", "stats-test-3@example.com", "stats-test-4@example.com", "stats-test-5@example.com"}
	for i, status := range statuses {
		email := models.Email{
			UserID:      userID,
			MessageID:   messageIDs[i],
			Subject:     strPtr("Stats Test"),
			FromAddress: strPtr("sender@example.com"),
			BodyText:    strPtr("Test body"),
			Folder:      strPtr("Inbox"),
			Status:      status,
		}
		_, err := service.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("failed to create email: %v", err)
		}
	}

	// Get stats
	stats, err = service.GetEmailStats(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get email stats: %v", err)
	}

	// Verify counts
	if stats["unprocessed"] != 2 {
		t.Errorf("expected unprocessed count 2, got %d", stats["unprocessed"])
	}

	if stats["triaged"] != 1 {
		t.Errorf("expected triaged count 1, got %d", stats["triaged"])
	}

	if stats["reviewed"] != 1 {
		t.Errorf("expected reviewed count 1, got %d", stats["reviewed"])
	}

	if stats["archived"] != 1 {
		t.Errorf("expected archived count 1, got %d", stats["archived"])
	}

	if stats["converted"] != 0 {
		t.Errorf("expected converted count 0, got %d", stats["converted"])
	}
}

func TestCreateEmailWithOptionalFields(t *testing.T) {
	s := tests.Setup()
	defer tests.Teardown()

	ctx := context.Background()
	userID := 1
	now := time.Now().UTC()

	// Use transaction during testing for proper isolation
	var db models.Database
	if s.Tx != nil {
		db = s.Tx
	} else {
		db = s.DB
	}
	service := NewEmailService(db)

	// Create email with all optional fields (but without email_account_id since it doesn't exist)
	threadID := "thread-123"
	email := models.Email{
		UserID:      userID,
		MessageID:   "optional-fields@example.com",
		ThreadID:    &threadID,
		Subject:     strPtr("Optional Fields Test"),
		FromAddress: strPtr("sender@example.com"),
		FromName:    strPtr("Sender Name"),
		ToAddresses: strPtr("recipient@example.com"),
		BodyText:    strPtr("Plain text body"),
		BodyHTML:    strPtr("<p>HTML body</p>"),
		ReceivedAt:  &now,
		Folder:      strPtr("Sent"),
		Status:      "reviewed",
	}

	created, err := service.CreateEmail(ctx, email)
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	// Verify all fields
	if created.EmailAccountID != nil {
		t.Errorf("expected email_account_id to be nil, got %v", created.EmailAccountID)
	}

	if created.ThreadID == nil || *created.ThreadID != threadID {
		t.Errorf("expected thread_id %s, got %v", threadID, created.ThreadID)
	}

	if created.ToAddresses == nil || *created.ToAddresses != *email.ToAddresses {
		t.Errorf("expected to_addresses %s, got %v", *email.ToAddresses, created.ToAddresses)
	}

	if created.ReceivedAt == nil {
		t.Error("expected received_at to be set")
	}

	if created.Status != "reviewed" {
		t.Errorf("expected status 'reviewed', got %s", created.Status)
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}
