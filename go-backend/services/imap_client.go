package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"

	"go-backend/models"
)

const (
	// DefaultIMAPPort is the default IMAP over SSL port
	DefaultIMAPPort = "993"
	// DefaultIMAPServer is the default Fastmail IMAP server
	DefaultIMAPServer = "imap.fastmail.com"
	// DefaultMailbox is the default mailbox to select
	DefaultMailbox = "INBOX"
)

// IMAPClient handles communication with an IMAP server
// Note: This client is not safe for concurrent use
type IMAPClient struct {
	client         *client.Client
	server         string
	username       string
	password       string
	mailbox        string
	uidValidity    uint32
	connected      bool
	mailboxSelected bool
}

// NewIMAPClient creates a new IMAP client with the default INBOX mailbox
func NewIMAPClient(server, username, password string) *IMAPClient {
	return &IMAPClient{
		server:   server,
		username: username,
		password: password,
		mailbox:  DefaultMailbox,
	}
}

// NewIMAPClientWithMailbox creates a new IMAP client with a specific mailbox
func NewIMAPClientWithMailbox(server, username, password, mailbox string) *IMAPClient {
	return &IMAPClient{
		server:   server,
		username: username,
		password: password,
		mailbox:  mailbox,
	}
}

// Connect establishes a connection to the IMAP server and authenticates
func (c *IMAPClient) Connect(ctx context.Context) error {
	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Split server into host and port if not specified
	serverAddr := c.server
	if !strings.Contains(serverAddr, ":") {
		serverAddr = serverAddr + ":" + DefaultIMAPPort
	}

	log.Printf("[imap] connecting to %s as %s", serverAddr, c.username)

	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: strings.Split(serverAddr, ":")[0],
		// For Fastmail, we'll use standard TLS verification
		InsecureSkipVerify: false,
	}

	// Connect with timeout using DialTLS
	type result struct {
		client *client.Client
		err    error
	}
	connectChan := make(chan result, 1)
	go func() {
		cl, err := client.DialTLS(serverAddr, tlsConfig)
		connectChan <- result{cl, err}
	}()

	select {
	case res := <-connectChan:
		if res.err != nil {
			return fmt.Errorf("failed to connect to IMAP server %s: %w", serverAddr, res.err)
		}
		c.client = res.client
	case <-ctx.Done():
		return fmt.Errorf("connection timed out")
	}

	// Authenticate
	authChan := make(chan error, 1)
	go func() {
		authChan <- c.client.Login(c.username, c.password)
	}()

	select {
	case err := <-authChan:
		if err != nil {
			// Clean up connection on auth failure
			_ = c.client.Close()
			c.client = nil
			return fmt.Errorf("IMAP authentication failed for user %s: %w", c.username, err)
		}
	case <-ctx.Done():
		// Clean up connection on timeout
		_ = c.client.Close()
		c.client = nil
		return fmt.Errorf("authentication timed out")
	}

	c.connected = true
	log.Printf("[imap] connected and authenticated to %s", serverAddr)

	return nil
}

// SelectInbox selects the configured mailbox
func (c *IMAPClient) SelectInbox(ctx context.Context) error {
	if !c.connected {
		return fmt.Errorf("not connected - call Connect() first")
	}

	if c.mailboxSelected {
		return nil // Already selected
	}

	mbox, err := c.client.Select(c.mailbox, false)
	if err != nil {
		return fmt.Errorf("failed to select mailbox %s: %w", c.mailbox, err)
	}

	// Store UIDVALIDITY for incremental sync
	c.uidValidity = mbox.UidValidity
	c.mailboxSelected = true

	log.Printf("[imap] selected mailbox %s (UIDVALIDITY: %d, Messages: %d)", c.mailbox, c.uidValidity, mbox.Messages)

	return nil
}

// FetchRecentEmails fetches the most recent emails from the selected mailbox
// Returns emails and the highest UID seen (for state tracking)
func (c *IMAPClient) FetchRecentEmails(ctx context.Context, limit int) ([]models.Email, uint32, error) {
	if !c.connected {
		return nil, 0, fmt.Errorf("not connected - call Connect() first")
	}
	if !c.mailboxSelected {
		return nil, 0, fmt.Errorf("mailbox not selected - call SelectInbox() first")
	}

	// Get mailbox status to know total messages
	mboxStatus, err := c.client.Status(c.mailbox, []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get mailbox status: %w", err)
	}

	totalMessages := mboxStatus.Messages
	if totalMessages == 0 {
		return []models.Email{}, 0, nil
	}

	// Calculate sequence numbers for recent emails (limit to 'limit' most recent)
	fromSeq := uint32(1)
	if totalMessages > uint32(limit) {
		fromSeq = totalMessages - uint32(limit) + 1
	}

	// Create a sequence set for the range
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(fromSeq, totalMessages)

	// Fetch the emails using sequence numbers with body section
	// Note: Using Fetch() not UidFetch() because seqSet contains sequence numbers, not UIDs
	messages := make(chan *imap.Message, limit)
	errChan := make(chan error, 1)

	go func() {
		// Fetch envelope, flags, UID, and the full RFC822 message for body extraction
		// Use BodySectionName with Peek: true to reliably avoid marking emails as read
		bodySection := &imap.BodySectionName{Peek: true}
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
			bodySection.FetchItem(),
		}
		errChan <- c.client.Fetch(seqSet, items, messages)
	}()

	var emails []models.Email
	var maxUID uint32

	// Process messages
	done := make(chan bool)
	go func() {
		for msg := range messages {
			if msg.Uid > maxUID {
				maxUID = msg.Uid
			}
			email := c.convertIMAPToEmail(msg)
			emails = append(emails, email)
		}
		done <- true
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch emails: %w", err)
		}
		// Wait for message processing to complete
		<-done
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("fetch emails timed out")
	}

	log.Printf("[imap] fetched %d emails (UID range: %d - %d)", len(emails), fromSeq, maxUID)

	return emails, maxUID, nil
}

// FetchRecentEmailsWithAttachments fetches the most recent emails with raw attachment data
// This is used for automatic file vault creation during sync
func (c *IMAPClient) FetchRecentEmailsWithAttachments(ctx context.Context, limit int) ([]EmailWithAttachments, uint32, error) {
	if !c.connected {
		return nil, 0, fmt.Errorf("not connected - call Connect() first")
	}
	if !c.mailboxSelected {
		return nil, 0, fmt.Errorf("mailbox not selected - call SelectInbox() first")
	}

	// Get mailbox status to know total messages
	mboxStatus, err := c.client.Status(c.mailbox, []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get mailbox status: %w", err)
	}

	totalMessages := mboxStatus.Messages
	if totalMessages == 0 {
		return []EmailWithAttachments{}, 0, nil
	}

	// Calculate sequence numbers for recent emails (limit to 'limit' most recent)
	fromSeq := uint32(1)
	if totalMessages > uint32(limit) {
		fromSeq = totalMessages - uint32(limit) + 1
	}

	// Create a sequence set for the range
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(fromSeq, totalMessages)

	// Fetch the emails using sequence numbers with body section
	messages := make(chan *imap.Message, limit)
	errChan := make(chan error, 1)

	go func() {
		bodySection := &imap.BodySectionName{Peek: true}
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
			bodySection.FetchItem(),
		}
		errChan <- c.client.Fetch(seqSet, items, messages)
	}()

	var emails []EmailWithAttachments
	var maxUID uint32

	// Process messages
	done := make(chan bool)
	go func() {
		for msg := range messages {
			if msg.Uid > maxUID {
				maxUID = msg.Uid
			}
			emailWithAtt := c.convertIMAPToEmailWithAttachments(msg)
			emails = append(emails, emailWithAtt)
		}
		done <- true
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch emails: %w", err)
		}
		<-done
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("fetch emails timed out")
	}

	log.Printf("[imap] fetched %d emails with attachments (UID range: %d - %d)", len(emails), fromSeq, maxUID)

	return emails, maxUID, nil
}

// FetchEmailsSinceUID fetches emails with UID greater than the specified lastUID
// This is used for incremental sync.
// Note: This does NOT check for UIDVALIDITY changes. Callers should check GetUIDValidity()
// to detect mailbox resets and handle them appropriately.
func (c *IMAPClient) FetchEmailsSinceUID(ctx context.Context, lastUID uint32) ([]models.Email, uint32, error) {
	if !c.connected {
		return nil, 0, fmt.Errorf("not connected - call Connect() first")
	}
	if !c.mailboxSelected {
		return nil, 0, fmt.Errorf("mailbox not selected - call SelectInbox() first")
	}

	// Get mailbox status
	mboxStatus, err := c.client.Status(c.mailbox, []imap.StatusItem{imap.StatusUidNext})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get mailbox status: %w", err)
	}

	uidNext := mboxStatus.UidNext
	if uidNext == 0 || uidNext <= lastUID {
		// No new messages
		return []models.Email{}, lastUID, nil
	}

	// Create sequence set for UIDs greater than lastUID
	fromUID := lastUID + 1
	toUID := uidNext - 1

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(fromUID, toUID)

	// Fetch the emails with full RFC822 content
	messages := make(chan *imap.Message, int(toUID-fromUID+1))
	errChan := make(chan error, 1)

	go func() {
		// Use BodySectionName with Peek: true to reliably avoid marking emails as read
		bodySection := &imap.BodySectionName{Peek: true}
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
			bodySection.FetchItem(),
		}
		errChan <- c.client.UidFetch(seqSet, items, messages)
	}()

	var emails []models.Email
	var maxUID uint32

	done := make(chan bool)
	go func() {
		for msg := range messages {
			if msg.Uid > maxUID {
				maxUID = msg.Uid
			}
			email := c.convertIMAPToEmail(msg)
			emails = append(emails, email)
		}
		done <- true
	}()

	select {
	case err := <-errChan:
		if err != nil {
			// IMAP errors for non-existent UIDs are common in incremental sync
			// Treat as "no new messages" rather than hard error
			return []models.Email{}, lastUID, nil
		}
		// Wait for message processing
		<-done
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("fetch emails since UID timed out")
	}

	if maxUID == 0 {
		maxUID = lastUID // Preserve state if no new messages
	}

	log.Printf("[imap] fetched %d new emails since UID %d", len(emails), lastUID)

	return emails, maxUID, nil
}

// FetchEmailsSinceUIDWithAttachments fetches emails with UID greater than the specified lastUID
// This includes raw attachment data for automatic file vault creation
func (c *IMAPClient) FetchEmailsSinceUIDWithAttachments(ctx context.Context, lastUID uint32) ([]EmailWithAttachments, uint32, error) {
	if !c.connected {
		return nil, 0, fmt.Errorf("not connected - call Connect() first")
	}
	if !c.mailboxSelected {
		return nil, 0, fmt.Errorf("mailbox not selected - call SelectInbox() first")
	}

	// Get mailbox status
	mboxStatus, err := c.client.Status(c.mailbox, []imap.StatusItem{imap.StatusUidNext})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get mailbox status: %w", err)
	}

	uidNext := mboxStatus.UidNext
	if uidNext == 0 || uidNext <= lastUID {
		// No new messages
		return []EmailWithAttachments{}, lastUID, nil
	}

	// Create sequence set for UIDs greater than lastUID
	fromUID := lastUID + 1
	toUID := uidNext - 1

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(fromUID, toUID)

	// Fetch the emails with full RFC822 content
	messages := make(chan *imap.Message, int(toUID-fromUID+1))
	errChan := make(chan error, 1)

	go func() {
		bodySection := &imap.BodySectionName{Peek: true}
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
			bodySection.FetchItem(),
		}
		errChan <- c.client.UidFetch(seqSet, items, messages)
	}()

	var emails []EmailWithAttachments
	var maxUID uint32

	done := make(chan bool)
	go func() {
		for msg := range messages {
			if msg.Uid > maxUID {
				maxUID = msg.Uid
			}
			emailWithAtt := c.convertIMAPToEmailWithAttachments(msg)
			emails = append(emails, emailWithAtt)
		}
		done <- true
	}()

	select {
	case err := <-errChan:
		if err != nil {
			// IMAP errors for non-existent UIDs are common in incremental sync
			return []EmailWithAttachments{}, lastUID, nil
		}
		<-done
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("fetch since UID timed out")
	}

	if maxUID == 0 {
		maxUID = lastUID
	}

	log.Printf("[imap] fetched %d new emails with attachments since UID %d", len(emails), lastUID)

	return emails, maxUID, nil
}

// GetUIDValidity returns the current UIDVALIDITY value for the selected mailbox
// Callers should compare this with previously stored values to detect mailbox resets
func (c *IMAPClient) GetUIDValidity() uint32 {
	return c.uidValidity
}

// Close closes the IMAP connection
func (c *IMAPClient) Close() error {
	if !c.connected || c.client == nil {
		return nil
	}

	if err := c.client.Logout(); err != nil {
		log.Printf("[imap] warning: error during logout: %v", err)
	}

	c.connected = false
	c.mailboxSelected = false
	c.client = nil

	log.Printf("[imap] connection closed")

	return nil
}

// GetMailbox returns the currently configured mailbox name
func (c *IMAPClient) GetMailbox() string {
	return c.mailbox
}

// ListMailboxes returns a list of all available mailboxes
func (c *IMAPClient) ListMailboxes(ctx context.Context) ([]string, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected - call Connect() first")
	}

	mailboxes := make(chan *imap.MailboxInfo, 20)
	errChan := make(chan error, 1)

	go func() {
		errChan <- c.client.List("", "*", mailboxes)
	}()

	var mailboxNames []string
	for mbox := range mailboxes {
		mailboxNames = append(mailboxNames, mbox.Name)
	}

	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	log.Printf("[imap] found %d mailboxes: %v", len(mailboxNames), mailboxNames)
	return mailboxNames, nil
}

// FindArchiveMailbox tries to find the archive folder by trying common names
func (c *IMAPClient) FindArchiveMailbox(ctx context.Context) (string, error) {
	mailboxes, err := c.ListMailboxes(ctx)
	if err != nil {
		return "", err
	}

	// Common archive folder names (case-insensitive)
	archiveNames := []string{"Archive", "Archives", "All Mail", "AllMail", "archived", "archival"}

	// Try exact matches first
	for _, name := range archiveNames {
		for _, mbox := range mailboxes {
			if strings.EqualFold(mbox, name) {
				log.Printf("[imap] found archive mailbox: %s", mbox)
				return mbox, nil
			}
		}
	}

	// Try partial matches
	for _, mbox := range mailboxes {
		lower := strings.ToLower(mbox)
		for _, name := range archiveNames {
			if strings.Contains(lower, strings.ToLower(name)) {
				log.Printf("[imap] found archive mailbox (partial match): %s", mbox)
				return mbox, nil
			}
		}
	}

	return "", fmt.Errorf("archive mailbox not found")
}

// SelectMailbox selects a specific mailbox by name
func (c *IMAPClient) SelectMailbox(ctx context.Context, mailbox string) error {
	if !c.connected {
		return fmt.Errorf("not connected - call Connect() first")
	}

	mbox, err := c.client.Select(mailbox, false)
	if err != nil {
		return fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	// Update the client's mailbox and store UIDVALIDITY
	c.mailbox = mailbox
	c.uidValidity = mbox.UidValidity
	c.mailboxSelected = true

	log.Printf("[imap] selected mailbox %s (UIDVALIDITY: %d, Messages: %d)", mailbox, c.uidValidity, mbox.Messages)

	return nil
}

// NormalizeMessageID removes angle brackets and normalizes Message-ID for comparison
// Exported for use in other packages
func NormalizeMessageID(msgID string) string {
	msgID = strings.TrimSpace(msgID)
	msgID = strings.TrimPrefix(msgID, "<")
	msgID = strings.TrimSuffix(msgID, ">")
	return strings.ToLower(msgID)
}

// GetAllMessageUIDs retrieves all UIDs and Message-IDs from the currently selected mailbox
// This is used for reconciliation to detect emails that have been moved externally
func (c *IMAPClient) GetAllMessageUIDs(ctx context.Context) (map[string]bool, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected - call Connect() first")
	}
	if !c.mailboxSelected {
		return nil, fmt.Errorf("mailbox not selected - call SelectMailbox() first")
	}

	// Get mailbox status to know total messages
	mboxStatus, err := c.client.Status(c.mailbox, []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return nil, fmt.Errorf("failed to get mailbox status: %w", err)
	}

	totalMessages := mboxStatus.Messages
	if totalMessages == 0 {
		return make(map[string]bool), nil
	}

	// Create sequence set for all messages
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(1, totalMessages)

	// Fetch only UID and envelope (which contains Message-ID)
	messages := make(chan *imap.Message, 100)
	errChan := make(chan error, 1)

	go func() {
		items := []imap.FetchItem{
			imap.FetchUid,
			imap.FetchEnvelope,
		}
		errChan <- c.client.Fetch(seqSet, items, messages)
	}()

	// Build set of normalized Message-IDs (we only need presence, not UIDs)
	messageIDs := make(map[string]bool)
	done := make(chan bool)
	go func() {
		for msg := range messages {
			messageID := ""
			if msg.Envelope != nil && msg.Envelope.MessageId != "" {
				messageID = msg.Envelope.MessageId
			} else {
				// Fallback to IMAP UID-based ID if no Message-ID
				messageID = fmt.Sprintf("imap-%d", msg.Uid)
			}
			normalized := NormalizeMessageID(messageID)
			messageIDs[normalized] = true
			if len(messageIDs) <= 5 {
				log.Printf("[imap] debug: UID %d -> MessageID '%s' -> normalized '%s'", msg.Uid, messageID, normalized)
			}
		}
		done <- true
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages: %w", err)
		}
		<-done
	case <-ctx.Done():
		return nil, fmt.Errorf("fetch messages timed out")
	}

	log.Printf("[imap] fetched %d Message-IDs from mailbox %s", len(messageIDs), c.mailbox)

	return messageIDs, nil
}

// FindUIDByMessageID searches for an email's UID by its Message-ID header
// Returns the UID if found, 0 if not found
func (c *IMAPClient) FindUIDByMessageID(ctx context.Context, messageID string) (uint32, error) {
	if !c.connected {
		return 0, fmt.Errorf("not connected - call Connect() first")
	}
	if !c.mailboxSelected {
		return 0, fmt.Errorf("mailbox not selected - call SelectInbox() first")
	}

	// Search for all messages, then scan for matching Message-ID
	// IMAP SEARCH doesn't support searching by Message-ID header directly
	mboxStatus, err := c.client.Status(c.mailbox, []imap.StatusItem{imap.StatusMessages})
	if err != nil {
		return 0, fmt.Errorf("failed to get mailbox status: %w", err)
	}

	totalMessages := mboxStatus.Messages
	if totalMessages == 0 {
		return 0, fmt.Errorf("no messages in mailbox")
	}

	// Create sequence set for all messages
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(1, totalMessages)

	// Fetch only the Message-ID header and UID
	messages := make(chan *imap.Message, 10)
	errChan := make(chan error, 1)

	go func() {
		items := []imap.FetchItem{
			imap.FetchUid,
			imap.FetchBodyStructure,
		}
		errChan <- c.client.Fetch(seqSet, items, messages)
	}()

	// Search through messages for matching Message-ID
	var foundUID uint32
	done := make(chan bool)
	go func() {
		for msg := range messages {
			// Try to get Message-ID from envelope first
			if msg.Envelope != nil && msg.Envelope.MessageId == messageID {
				foundUID = msg.Uid
				break
			}
			// If not in envelope, we'd need to fetch headers, but for now
			// just return what we found. The envelope should have Message-ID.
		}
		done <- true
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return 0, fmt.Errorf("failed to search messages: %w", err)
		}
		<-done
	case <-ctx.Done():
		return 0, fmt.Errorf("search timed out")
	case <-done:
		// Found early, wait for fetch to complete
		<-errChan
	}

	if foundUID == 0 {
		return 0, fmt.Errorf("message not found")
	}

	return foundUID, nil
}

// MoveToArchive moves an email to the Archive folder
func (c *IMAPClient) MoveToArchive(ctx context.Context, uid uint32) error {
	if !c.connected {
		return fmt.Errorf("not connected - call Connect() first")
	}

	archiveMailbox := "Archive"
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Try MOVE command first (IMAP4rev2)
	err := c.client.UidMove(seqSet, archiveMailbox)
	if err == nil {
		log.Printf("[imap] moved email UID %d to %s", uid, archiveMailbox)
		return nil
	}

	// If MOVE is not supported, fall back to COPY + STORE + EXPUNGE
	if strings.Contains(err.Error(), "MOVE") || strings.Contains(err.Error(), "unsupported") {
		log.Printf("[imap] MOVE not supported, falling back to COPY + DELETE")

		// First, copy the message to Archive
		if err := c.client.UidCopy(seqSet, archiveMailbox); err != nil {
			return fmt.Errorf("failed to copy email to %s: %w", archiveMailbox, err)
		}

		// Then, mark the original as deleted and expunge
		ch := make(chan *imap.Message, 1)
		if err := c.client.Store(seqSet, imap.AddFlags, imap.DeletedFlag, ch); err != nil {
			return fmt.Errorf("failed to mark email as deleted: %w", err)
		}
		<-ch // Wait for operation to complete

		// Expunge to permanently remove the original
		if err := c.client.Expunge(nil); err != nil {
			return fmt.Errorf("failed to expunge email: %w", err)
		}

		log.Printf("[imap] copied email UID %d to %s and deleted original", uid, archiveMailbox)
		return nil
	}

	return fmt.Errorf("failed to move email to %s: %w", archiveMailbox, err)
}

// MoveFromArchive moves an email from Archive back to INBOX
func (c *IMAPClient) MoveFromArchive(ctx context.Context, uid uint32) error {
	if !c.connected {
		return fmt.Errorf("not connected - call Connect() first")
	}

	// First, select the Archive mailbox to get the message
	if _, err := c.client.Select("Archive", false); err != nil {
		return fmt.Errorf("failed to select Archive mailbox: %w", err)
	}

	inboxMailbox := "INBOX"
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Try MOVE command first
	err := c.client.UidMove(seqSet, inboxMailbox)
	if err == nil {
		log.Printf("[imap] moved email UID %d from Archive to INBOX", uid)
		return nil
	}

	// If MOVE is not supported, fall back to COPY + STORE + EXPUNGE
	if strings.Contains(err.Error(), "MOVE") || strings.Contains(err.Error(), "unsupported") {
		log.Printf("[imap] MOVE not supported, falling back to COPY + DELETE")

		// Copy to INBOX
		if err := c.client.UidCopy(seqSet, inboxMailbox); err != nil {
			return fmt.Errorf("failed to copy email to INBOX: %w", err)
		}

		// Mark as deleted
		ch := make(chan *imap.Message, 1)
		if err := c.client.Store(seqSet, imap.AddFlags, imap.DeletedFlag, ch); err != nil {
			return fmt.Errorf("failed to mark email as deleted: %w", err)
		}
		<-ch // Wait for operation to complete

		// Expunge
		if err := c.client.Expunge(nil); err != nil {
			return fmt.Errorf("failed to expunge email: %w", err)
		}

		log.Printf("[imap] copied email UID %d to INBOX and deleted from Archive", uid)
		return nil
	}

	return fmt.Errorf("failed to move email to INBOX: %w", err)
}

// extractBodyText extracts the text/plain and text/html body from an IMAP message
// Also extracts any attachments found in the message
func (c *IMAPClient) extractBodyText(msg *imap.Message) (string, string, []EmailAttachment, error) {
	// Look for the RFC822 literal in the message body
	for _, section := range msg.Body {
		// section is an imap.Literal interface
		literal, ok := section.(imap.Literal)
		if !ok {
			continue
		}

		// Read the literal content
		data, err := io.ReadAll(literal)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to read message body: %w", err)
		}

		if len(data) == 0 {
			continue
		}

		// Parse the email message using go-message
		reader := bytes.NewReader(data)
		mr, err := mail.CreateReader(reader)
		if err != nil {
			// If parsing fails, try to extract text directly
			return c.extractPlainText(data), "", nil, nil
		}

		// Iterate through message parts to find text/plain, text/html, and attachments
		var textBody, htmlBody strings.Builder
		var attachments []EmailAttachment

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				// Failed to parse multipart, try to get body from entire message
				return c.extractPlainText(data), "", nil, nil
			}

			contentType := part.Header.Get("Content-Type")
			log.Printf("[imap] processing part with Content-Type: %s", contentType)

			// Get Content-Disposition to determine if this is an attachment
			contentDisposition := part.Header.Get("Content-Disposition")
			contentID := part.Header.Get("Content-ID")

			// Parse filename from Content-Type or Content-Disposition
			var filename string
			if contentDisposition != "" {
			 disposition, params, err := mime.ParseMediaType(contentDisposition)
				if err == nil {
					if fname := params["filename"]; fname != "" {
						filename = fname
					}
					if disposition == "inline" && filename == "" {
						// Inline without filename might have Content-ID
						if contentID != "" {
							filename = "inline_" + contentID
						}
					}
				}
			}

			// Check if this part is an attachment (has disposition or is non-text)
			isAttachment := contentDisposition != "" ||
				(contentType != "" && !strings.HasPrefix(contentType, "text/") && !strings.HasPrefix(contentType, "multipart/"))

			// Determine if this is an inline attachment
			isInline := strings.HasPrefix(contentDisposition, "inline") || contentID != ""

			switch {
			case strings.HasPrefix(contentType, "text/plain") && !isAttachment:
				data, err := io.ReadAll(part.Body)
				if err != nil {
					continue
				}
				textBody.WriteString(string(data))

			case strings.HasPrefix(contentType, "text/html") && !isAttachment:
				data, err := io.ReadAll(part.Body)
				if err != nil {
					continue
				}
				htmlBody.WriteString(string(data))

			case strings.HasPrefix(contentType, "multipart/"):
				// multipart sections are handled by NextPart()
				continue

			default:
				// This is an attachment or inline image
				attachmentData, err := io.ReadAll(part.Body)
				if err != nil {
					log.Printf("[imap] failed to read attachment data: %v", err)
					continue
				}

				// If we couldn't determine filename earlier, try to get it from Content-Type params
				if filename == "" {
					_, params, err := mime.ParseMediaType(contentType)
					if err == nil {
						if fname := params["name"]; fname != "" {
							filename = fname
						}
					}
				}

				// Last resort: generate a filename
				if filename == "" {
					mediaType, _, _ := mime.ParseMediaType(contentType)
					ext := ".bin"
					if mediaType != "" {
						ext = "." + strings.Split(mediaType, "/")[0]
						if ext == ".image" {
							ext = ".jpg"
						} else if ext == ".application" {
							ext = ".bin"
						}
					}
					filename = fmt.Sprintf("attachment_%d%s", len(attachments)+1, ext)
				}

				// Clean up Content-ID (remove angle brackets if present)
				cleanContentID := strings.Trim(contentID, "<>")

				attachment := EmailAttachment{
					Filename:    filename,
					ContentType: contentType,
					ContentID:   cleanContentID,
					IsInline:    isInline,
					Size:        int64(len(attachmentData)),
					Data:        attachmentData,
				}

				attachments = append(attachments, attachment)
				log.Printf("[imap] extracted attachment: %s (size: %d, inline: %v, cid: %s)", filename, len(attachmentData), isInline, cleanContentID)
			}
		}

		// Return the extracted bodies and attachments
		return textBody.String(), htmlBody.String(), attachments, nil
	}

	return "", "", nil, fmt.Errorf("no body literal found in message")
}

// extractPlainText attempts to extract plain text from raw email data
func (c *IMAPClient) extractPlainText(data []byte) string {
	// Simple heuristic: skip headers and look for the first blank line
	dataStr := string(data)

	// Find the end of headers (first empty line)
	headerEnd := strings.Index(dataStr, "\n\n")
	if headerEnd == -1 {
		headerEnd = strings.Index(dataStr, "\r\n\r\n")
	}

	if headerEnd != -1 {
		// Extract everything after headers
		body := dataStr[headerEnd:]
		// Remove leading newlines
		body = strings.TrimLeft(body, "\r\n")
		return body
	}

	// If no clear header boundary, return as-is (truncated if too large)
	if len(dataStr) > 10000 {
		return dataStr[:10000] + "... (truncated)"
	}
	return dataStr
}

// EmailAttachment represents an attachment parsed from an email
type EmailAttachment struct {
	Filename    string
	ContentType string
	ContentID   string // Content-ID for inline images
	IsInline    bool
	Size        int64
	Data        []byte
}

// EmailWithAttachments represents an email with its raw attachment data
type EmailWithAttachments struct {
	Email       models.Email
	Attachments []EmailAttachment
}

// convertIMAPToEmail converts an IMAP message to models.Email
func (c *IMAPClient) convertIMAPToEmail(msg *imap.Message) models.Email {
	uid := int64(msg.Uid)
	email := models.Email{
		// Use UID as message ID fallback, will be overridden by RFC822 Message-ID if available
		MessageID: fmt.Sprintf("imap-%d", msg.Uid),
		Folder:    &c.mailbox,
		IMAPUID:   &uid, // Store IMAP UID for folder operations like archive
		Status:    "unprocessed",
	}

	// Extract envelope
	if msg.Envelope != nil {
		// Subject
		if msg.Envelope.Subject != "" {
			email.Subject = &msg.Envelope.Subject
		}

		// From
		if len(msg.Envelope.From) > 0 {
			from := msg.Envelope.From[0]
			if from.Address() != "" {
				addr := from.Address()
				email.FromAddress = &addr
			}
			if from.PersonalName != "" {
				email.FromName = &msg.Envelope.From[0].PersonalName
			}
		}

		// To
		if len(msg.Envelope.To) > 0 {
			var toAddresses []string
			for _, addr := range msg.Envelope.To {
				if addr.PersonalName != "" {
					toAddresses = append(toAddresses, fmt.Sprintf("%s <%s>", addr.PersonalName, addr.Address()))
				} else {
					toAddresses = append(toAddresses, addr.Address())
				}
			}
			toStr := strings.Join(toAddresses, ", ")
			email.ToAddresses = &toStr
		}

		// Message-ID header if available (preferred over UID-based ID)
		if msg.Envelope.MessageId != "" {
			email.MessageID = msg.Envelope.MessageId
		}
	}

	// Extract internal date (received date)
	if !msg.InternalDate.IsZero() {
		email.ReceivedAt = &msg.InternalDate
	}

	// Check for \Seen flag to determine if email has been read
	email.IsRead = false
	for _, flag := range msg.Flags {
		if flag == imap.SeenFlag {
			email.IsRead = true
			break
		}
	}

	// Extract body content (text/plain and text/html) and attachments
	textBody, htmlBody, attachments, err := c.extractBodyText(msg)
	if err != nil {
		log.Printf("[imap] warning: failed to extract body from email UID %d: %v", msg.Uid, err)
		// Continue without body rather than failing the entire sync
	} else {
		if textBody != "" {
			email.BodyText = &textBody
		}
		if htmlBody != "" {
			email.BodyHTML = &htmlBody
		}
		// Store attachments in email for later processing
		// Convert services.EmailAttachment to models.EmailAttachment
		if len(attachments) > 0 {
			modelAttachments := make([]models.EmailAttachment, len(attachments))
			for i, att := range attachments {
				var contentType *string
				if att.ContentType != "" {
					contentType = &att.ContentType
				}
				var contentID *string
				if att.ContentID != "" {
					contentID = &att.ContentID
				}
				size := att.Size
				modelAttachments[i] = models.EmailAttachment{
					Filename:    att.Filename,
					ContentType: contentType,
					ContentID:   contentID,
					IsInline:    att.IsInline,
					Size:        &size,
					// Note: Data is not stored in models.EmailAttachment as it's only used during processing
				}
			}
			email.Attachments = modelAttachments
		}
	}

	return email
}

// convertIMAPToEmailWithAttachments converts an IMAP message to EmailWithAttachments
// This includes the raw attachment data for upload to S3
func (c *IMAPClient) convertIMAPToEmailWithAttachments(msg *imap.Message) EmailWithAttachments {
	// Call extractBodyText once to get both body content AND attachments
	textBody, htmlBody, attachments, _ := c.extractBodyText(msg)

	// Build the email object (similar to convertIMAPToEmail but using extracted data)
	uid := int64(msg.Uid)
	email := models.Email{
		MessageID: fmt.Sprintf("imap-%d", msg.Uid),
		Folder:    &c.mailbox,
		IMAPUID:   &uid,
		Status:    "unprocessed",
	}

	// Extract envelope
	if msg.Envelope != nil {
		if msg.Envelope.Subject != "" {
			email.Subject = &msg.Envelope.Subject
		}
		if len(msg.Envelope.From) > 0 {
			from := msg.Envelope.From[0]
			if from.Address() != "" {
				addr := from.Address()
				email.FromAddress = &addr
			}
			if from.PersonalName != "" {
				email.FromName = &msg.Envelope.From[0].PersonalName
			}
		}
		if len(msg.Envelope.To) > 0 {
			var toAddresses []string
			for _, addr := range msg.Envelope.To {
				if addr.PersonalName != "" {
					toAddresses = append(toAddresses, fmt.Sprintf("%s <%s>", addr.PersonalName, addr.Address()))
				} else {
					toAddresses = append(toAddresses, addr.Address())
				}
			}
			toStr := strings.Join(toAddresses, ", ")
			email.ToAddresses = &toStr
		}
		if msg.Envelope.MessageId != "" {
			email.MessageID = msg.Envelope.MessageId
		}
	}

	// Extract internal date
	if !msg.InternalDate.IsZero() {
		email.ReceivedAt = &msg.InternalDate
	}

	// Check for \Seen flag
	email.IsRead = false
	for _, flag := range msg.Flags {
		if flag == imap.SeenFlag {
			email.IsRead = true
			break
		}
	}

	// Set body content from extractBodyText results
	if textBody != "" {
		email.BodyText = &textBody
	}
	if htmlBody != "" {
		email.BodyHTML = &htmlBody
	}

	// Store attachments in email (convert services.EmailAttachment to models.EmailAttachment)
	if len(attachments) > 0 {
		modelAttachments := make([]models.EmailAttachment, len(attachments))
		for i, att := range attachments {
			var contentType *string
			if att.ContentType != "" {
				contentType = &att.ContentType
			}
			var contentID *string
			if att.ContentID != "" {
				contentID = &att.ContentID
			}
			size := att.Size
			modelAttachments[i] = models.EmailAttachment{
				Filename:    att.Filename,
				ContentType: contentType,
				ContentID:   contentID,
				IsInline:    att.IsInline,
				Size:        &size,
			}
		}
		email.Attachments = modelAttachments
	}

	return EmailWithAttachments{
		Email:       email,
		Attachments: attachments, // Keep the raw attachment data with S3 data
	}
}
