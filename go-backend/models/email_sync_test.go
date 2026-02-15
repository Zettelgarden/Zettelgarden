package models

import (
	"testing"
	"time"
)

func TestEmailAccountModel(t *testing.T) {
	jmapServerURL := "https://api.fastmail.com/jmap/session"
	appPasswordEncrypted := "encrypted_password_here"
	syncTime := time.Now()
	jmapState := "state_token"

	account := EmailAccount{
		ID:                  1,
		UserID:              1,
		EmailAddress:        "user@example.com",
		JMAPServerURL:       jmapServerURL,
		AppPasswordEncrypted: &appPasswordEncrypted,
		IsActive:            true,
		LastSyncAt:          &syncTime,
		SyncStatus:          "active",
		JMAPState:           &jmapState,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if account.EmailAddress != "user@example.com" {
		t.Errorf("expected EmailAddress to be user@example.com, got %s", account.EmailAddress)
	}

	if account.JMAPServerURL != jmapServerURL {
		t.Errorf("expected JMAPServerURL to be %s, got %s", jmapServerURL, account.JMAPServerURL)
	}

	if account.AppPasswordEncrypted == nil || *account.AppPasswordEncrypted != appPasswordEncrypted {
		t.Errorf("expected AppPasswordEncrypted to be %s, got %v", appPasswordEncrypted, account.AppPasswordEncrypted)
	}

	if account.IsActive != true {
		t.Errorf("expected IsActive to be true, got %v", account.IsActive)
	}

	if account.SyncStatus != "active" {
		t.Errorf("expected SyncStatus to be active, got %s", account.SyncStatus)
	}

	if account.JMAPState == nil || *account.JMAPState != jmapState {
		t.Errorf("expected JMAPState to be %s, got %v", jmapState, account.JMAPState)
	}
}

func TestEmailModel(t *testing.T) {
	messageID := "msg123@example.com"
	threadID := "thread456"
	subject := "Test Subject"
	fromAddress := "sender@example.com"
	fromName := "Sender Name"
	toAddresses := "recipient@example.com"
	bodyText := "Plain text body"
	bodyHTML := "<p>HTML body</p>"
	receivedAt := time.Now()
	folder := "Inbox"

	email := Email{
		ID:             1,
		UserID:         1,
		EmailAccountID: &[]int{1}[0],
		MessageID:      messageID,
		ThreadID:       &threadID,
		Subject:        &subject,
		FromAddress:    &fromAddress,
		FromName:       &fromName,
		ToAddresses:    &toAddresses,
		BodyText:       &bodyText,
		BodyHTML:       &bodyHTML,
		ReceivedAt:     &receivedAt,
		Folder:         &folder,
		Status:         "unprocessed",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if email.MessageID != messageID {
		t.Errorf("expected MessageID to be %s, got %s", messageID, email.MessageID)
	}

	if email.ThreadID == nil || *email.ThreadID != threadID {
		t.Errorf("expected ThreadID to be %s, got %v", threadID, email.ThreadID)
	}

	if email.Subject == nil || *email.Subject != subject {
		t.Errorf("expected Subject to be %s, got %v", subject, email.Subject)
	}

	if email.FromAddress == nil || *email.FromAddress != fromAddress {
		t.Errorf("expected FromAddress to be %s, got %v", fromAddress, email.FromAddress)
	}

	if email.BodyText == nil || *email.BodyText != bodyText {
		t.Errorf("expected BodyText to be %s, got %v", bodyText, email.BodyText)
	}

	if email.Status != "unprocessed" {
		t.Errorf("expected Status to be unprocessed, got %s", email.Status)
	}

	if email.Folder == nil || *email.Folder != folder {
		t.Errorf("expected Folder to be %s, got %v", folder, email.Folder)
	}
}

func TestEmailTriageDecisionModel(t *testing.T) {
	reasoning := "This email looks like spam"

	decision := EmailTriageDecision{
		ID:             1,
		EmailID:        1,
		Decision:       "delete",
		Confidence:     0.95,
		Reasoning:      &reasoning,
		IsAutoExecuted: true,
		CreatedAt:      time.Now(),
	}

	if decision.Decision != "delete" {
		t.Errorf("expected Decision to be delete, got %s", decision.Decision)
	}

	if decision.Confidence != 0.95 {
		t.Errorf("expected Confidence to be 0.95, got %f", decision.Confidence)
	}

	if decision.Reasoning == nil || *decision.Reasoning != reasoning {
		t.Errorf("expected Reasoning to be %s, got %v", reasoning, decision.Reasoning)
	}

	if decision.IsAutoExecuted != true {
		t.Errorf("expected IsAutoExecuted to be true, got %v", decision.IsAutoExecuted)
	}
}

func TestEmailCardLinkModel(t *testing.T) {
	link := EmailCardLink{
		ID:        1,
		EmailID:   1,
		CardID:    1,
		CreatedAt: time.Now(),
	}

	if link.EmailID != 1 {
		t.Errorf("expected EmailID to be 1, got %d", link.EmailID)
	}

	if link.CardID != 1 {
		t.Errorf("expected CardID to be 1, got %d", link.CardID)
	}
}

func TestCreateEmailAccountParams(t *testing.T) {
	appPassword := "my_app_password"

	params := CreateEmailAccountParams{
		EmailAddress: "user@example.com",
		AppPassword:  &appPassword,
	}

	if params.EmailAddress != "user@example.com" {
		t.Errorf("expected EmailAddress to be user@example.com, got %s", params.EmailAddress)
	}

	if params.AppPassword == nil || *params.AppPassword != appPassword {
		t.Errorf("expected AppPassword to be %s, got %v", appPassword, params.AppPassword)
	}
}

func TestUpdateEmailAccountParams(t *testing.T) {
	isActive := true
	syncStatus := "active"

	params := UpdateEmailAccountParams{
		IsActive:   &isActive,
		SyncStatus: &syncStatus,
	}

	if params.IsActive == nil || *params.IsActive != true {
		t.Errorf("expected IsActive to be true, got %v", params.IsActive)
	}

	if params.SyncStatus == nil || *params.SyncStatus != "active" {
		t.Errorf("expected SyncStatus to be active, got %v", params.SyncStatus)
	}

	// Test with nil values
	nilParams := UpdateEmailAccountParams{}

	if nilParams.IsActive != nil {
		t.Errorf("expected IsActive to be nil, got %v", nilParams.IsActive)
	}

	if nilParams.SyncStatus != nil {
		t.Errorf("expected SyncStatus to be nil, got %v", nilParams.SyncStatus)
	}
}

func TestEmailListFilters(t *testing.T) {
	status := "unprocessed"
	folder := "Inbox"
	limit := 50
	offset := 0

	filters := EmailListFilters{
		Status: &status,
		Folder: &folder,
		Limit:  &limit,
		Offset: &offset,
	}

	if filters.Status == nil || *filters.Status != "unprocessed" {
		t.Errorf("expected Status to be unprocessed, got %v", filters.Status)
	}

	if filters.Folder == nil || *filters.Folder != "Inbox" {
		t.Errorf("expected Folder to be Inbox, got %v", filters.Folder)
	}

	if filters.Limit == nil || *filters.Limit != 50 {
		t.Errorf("expected Limit to be 50, got %v", filters.Limit)
	}

	if filters.Offset == nil || *filters.Offset != 0 {
		t.Errorf("expected Offset to be 0, got %v", filters.Offset)
	}

	// Test with nil values
	nilFilters := EmailListFilters{}

	if nilFilters.Status != nil {
		t.Errorf("expected Status to be nil, got %v", nilFilters.Status)
	}

	if nilFilters.Folder != nil {
		t.Errorf("expected Folder to be nil, got %v", nilFilters.Folder)
	}
}
