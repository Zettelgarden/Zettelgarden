package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
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
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
			imap.FetchRFC822,
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
		items := []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
			imap.FetchRFC822,
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

// extractBodyText extracts the text/plain body from an IMAP message
func (c *IMAPClient) extractBodyText(msg *imap.Message) (string, string, error) {
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
			return "", "", fmt.Errorf("failed to read message body: %w", err)
		}

		if len(data) == 0 {
			continue
		}

		// Parse the email message using go-message
		reader := bytes.NewReader(data)
		mr, err := mail.CreateReader(reader)
		if err != nil {
			// If parsing fails, try to extract text directly
			return c.extractPlainText(data), "", nil
		}

		// Iterate through message parts to find text/plain and text/html
		var textBody, htmlBody strings.Builder

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				// Failed to parse multipart, try to get body from entire message
				return c.extractPlainText(data), "", nil
			}

			contentType := part.Header.Get("Content-Type")
			log.Printf("[imap] processing part with Content-Type: %s", contentType)

			switch {
			case strings.HasPrefix(contentType, "text/plain"):
				data, err := io.ReadAll(part.Body)
				if err != nil {
					continue
				}
				textBody.WriteString(string(data))

			case strings.HasPrefix(contentType, "text/html"):
				data, err := io.ReadAll(part.Body)
				if err != nil {
					continue
				}
				htmlBody.WriteString(string(data))

			case strings.HasPrefix(contentType, "multipart/"):
				// multipart sections are handled by NextPart()
				continue

			default:
				log.Printf("[imap] skipping attachment with Content-Type: %s", contentType)
			}
		}

		// Return the extracted bodies
		return textBody.String(), htmlBody.String(), nil
	}

	return "", "", fmt.Errorf("no body literal found in message")
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

	// Extract body content (text/plain and text/html)
	textBody, htmlBody, err := c.extractBodyText(msg)
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
	}

	return email
}
