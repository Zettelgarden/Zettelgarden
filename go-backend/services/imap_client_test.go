package services

import (
	"context"
	"testing"
)

// TestIMAPClient_NewClient tests creating a new IMAP client
func TestIMAPClient_NewClient(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	if client == nil {
		t.Fatal("NewIMAPClient returned nil")
	}

	if client.server != "imap.example.com:993" {
		t.Errorf("expected server 'imap.example.com:993', got '%s'", client.server)
	}

	if client.username != "user@example.com" {
		t.Errorf("expected username 'user@example.com', got '%s'", client.username)
	}

	if client.password != "password" {
		t.Errorf("expected password 'password', got '%s'", client.password)
	}

	if client.mailbox != "INBOX" {
		t.Errorf("expected default mailbox 'INBOX', got '%s'", client.mailbox)
	}
}

// TestIMAPClient_NewClient_DefaultPort tests that default port is appended
func TestIMAPClient_NewClient_DefaultPort(t *testing.T) {
	client := NewIMAPClient("imap.fastmail.com", "user@example.com", "password")

	if client.server != "imap.fastmail.com" {
		t.Errorf("expected server 'imap.fastmail.com', got '%s'", client.server)
	}
	// Note: The port is added during Connect, not in constructor
}

// TestIMAPClient_SelectInbox_BeforeConnect tests error when selecting mailbox before connecting
func TestIMAPClient_SelectInbox_BeforeConnect(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	ctx := context.Background()
	err := client.SelectInbox(ctx)

	if err == nil {
		t.Error("expected error when selecting inbox before connect, got nil")
	}
}

// TestIMAPClient_FetchRecentEmails_BeforeConnect tests error when fetching before connecting
func TestIMAPClient_FetchRecentEmails_BeforeConnect(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	ctx := context.Background()
	_, _, err := client.FetchRecentEmails(ctx, 10)

	if err == nil {
		t.Error("expected error when fetching before connect, got nil")
	}
}

// TestIMAPClient_Close_WhenNotConnected tests that Close doesn't error when not connected
func TestIMAPClient_Close_WhenNotConnected(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	// Should not error
	if err := client.Close(); err != nil {
		t.Errorf("expected no error when closing unconnected client, got: %v", err)
	}
}

// TestIMAPClient_GetUIDValidity tests getting UIDVALIDITY
func TestIMAPClient_GetUIDValidity(t *testing.T) {
	client := NewIMAPClient("imap.example.com:993", "user@example.com", "password")

	// Before connect, UIDVALIDITY should be 0
	if uid := client.GetUIDValidity(); uid != 0 {
		t.Errorf("expected UIDVALIDITY 0 before connect, got %d", uid)
	}
}
