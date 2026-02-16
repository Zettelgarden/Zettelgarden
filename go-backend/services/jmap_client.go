package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go-backend/models"
)

const (
	// JMAPCoreCapability is the URN for the JMAP Core capability
	JMAPCoreCapability = "urn:ietf:params:jmap:core"
	// JMAPMailCapability is the URN for the JMAP Mail capability
	JMAPMailCapability = "urn:ietf:params:jmap:mail"
)

// JMAPClient handles communication with a JMAP server (e.g., Fastmail)
type JMAPClient struct {
	httpClient *http.Client
	serverURL  string
	authToken  string // Bearer token for authentication
	accountID  string // from session
	apiURL     string // from session
	uploadURL  string
	downloadURL string
}

// JMAPSessionResponse represents the JMAP session response
type JMAPSessionResponse struct {
	Capabilities    map[string]interface{} `json:"capabilities"`
	APIURL          string                 `json:"apiUrl"`
	UploadURL       string                 `json:"uploadUrl"`
	DownloadURL     string                 `json:"downloadUrl"`
	EventSourceURL  string                 `json:"eventSourceUrl"`
	PrimaryAccounts map[string]string      `json:"primaryAccounts"`
	Accounts        map[string]JMAPAccount  `json:"accounts"`
}

// JMAPAccount represents a JMAP account
type JMAPAccount struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsPrimary bool   `json:"isPrimary"`
}

// JMAPRequest represents a JMAP request
type JMAPRequest struct {
	Using       []string        `json:"using"`
	MethodCalls [][]interface{} `json:"methodCalls"`
	CreatedIDs  map[string]string `json:"createdIds,omitempty"`
}

// JMAPResponse represents a JMAP response
type JMAPResponse struct {
	MethodResponses [][]interface{} `json:"methodResponses"`
	SessionState    string          `json:"sessionState,omitempty"`
	CreatedIDs      map[string]string `json:"createdIds,omitempty"`
}

// JMAPMailbox represents a JMAP mailbox
type JMAPMailbox struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Role     *string `json:"role,omitempty"`
	ParentID *string `json:"parentId,omitempty"`
}

// JMAPEmail represents a JMAP email
type JMAPEmail struct {
	ID          string                   `json:"id"`
	MessageID   []string                 `json:"messageId"`
	ThreadID    string                   `json:"threadId"`
	Subject     *string                  `json:"subject"`
	ReceivedAt  time.Time                `json:"receivedAt"`
	SentAt      time.Time                `json:"sentAt"`
	From        JMAPEmailAddress         `json:"from"`
	To          []JMAPEmailAddress       `json:"to"`
	CC          []JMAPEmailAddress       `json:"cc"`
	BCC         []JMAPEmailAddress       `json:"bcc"`
	ReplyTo     []JMAPEmailAddress       `json:"replyTo"`
	BodyValues  map[string]JMAPBodyValue `json:"bodyValues"`
	HTMLBody    []interface{}            `json:"htmlBody"`
	TextBody    []interface{}            `json:"textBody"`
	Attachments []interface{}            `json:"attachments"`
	Keywords    map[string]bool          `json:"keywords"`
	MailboxIDs  map[string]bool          `json:"mailboxIds"`
}

// JMAPEmailAddress represents an email address in JMAP
type JMAPEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// JMAPBodyValue represents a body value in JMAP
type JMAPBodyValue struct {
	Value      *string `json:"value"`
	IsEncoding bool    `json:"isEncoding"`
	Charset    string  `json:"charset,omitempty"`
}

// JMAPQueryResponse represents the response from an Email/query call
type JMAPQueryResponse struct {
	AccountID          string   `json:"accountId"`
	QueryState         string   `json:"queryState"`
	IDs                []string `json:"ids"`
	Limit              int      `json:"limit,omitempty"`
	Position           int      `json:"position,omitempty"`
	CanCalculateChanges bool    `json:"canCalculateChanges"`
}

// NewJMAPClient creates a new JMAP client
func NewJMAPClient(serverURL, authToken string) *JMAPClient {
	return &JMAPClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		serverURL: serverURL,
		authToken: authToken,
	}
}

// Connect establishes a JMAP session by fetching the session endpoint
func (c *JMAPClient) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.serverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create JMAP session request: %w", err)
	}

	// Set Bearer token auth
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to JMAP server %s: %w", c.serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("JMAP authentication failed - invalid bearer token")
		}
		return fmt.Errorf("JMAP server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JMAP session response: %w", err)
	}

	var session JMAPSessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		return fmt.Errorf("failed to parse JMAP session response: %w", err)
	}

	// Validate that the server supports mail capability
	if _, ok := session.Capabilities[JMAPMailCapability]; !ok {
		return fmt.Errorf("JMAP server does not support mail capability %s", JMAPMailCapability)
	}

	// Extract the primary mail account ID from primaryAccounts
	accountID, ok := session.PrimaryAccounts[JMAPMailCapability]
	if !ok || accountID == "" {
		return fmt.Errorf("JMAP session did not return a primary mail account ID")
	}

	// Store session data
	c.accountID = accountID
	c.apiURL = session.APIURL
	c.uploadURL = session.UploadURL
	c.downloadURL = session.DownloadURL

	log.Printf("[jmap] connected to %s, api=%s, account=%s", c.serverURL, c.apiURL, c.accountID)

	return nil
}

// GetMailboxes retrieves all mailboxes for the account
func (c *JMAPClient) GetMailboxes(ctx context.Context) ([]JMAPMailbox, error) {
	if c.accountID == "" {
		return nil, fmt.Errorf("not connected - call Connect() first")
	}

	req := JMAPRequest{
		Using: []string{JMAPCoreCapability, JMAPMailCapability},
		MethodCalls: [][]interface{}{
			{
				"Mailbox/get",
				map[string]interface{}{
					"accountId": c.accountID,
				},
				"call1",
			},
		},
	}

	var resp JMAPResponse
	if err := c.call(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get mailboxes: %w", err)
	}

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no response from JMAP server")
	}

	response := resp.MethodResponses[0]
	if len(response) < 2 {
		return nil, fmt.Errorf("invalid JMAP response format from Mailbox/get")
	}

	data, ok := response[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid JMAP response data format from Mailbox/get")
	}

	mailboxesData, ok := data["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid mailboxes data format in JMAP response")
	}

	var mailboxes []JMAPMailbox
	for _, mboxData := range mailboxesData {
		mboxMap, ok := mboxData.(map[string]interface{})
		if !ok {
			continue
		}

		mboxJSON, err := json.Marshal(mboxMap)
		if err != nil {
			log.Printf("[jmap] warning: failed to marshal mailbox data: %v", err)
			continue
		}

		var mbox JMAPMailbox
		if err := json.Unmarshal(mboxJSON, &mbox); err != nil {
			log.Printf("[jmap] warning: failed to unmarshal mailbox: %v", err)
			continue
		}

		mailboxes = append(mailboxes, mbox)
	}

	return mailboxes, nil
}

// FindInboxMailbox finds the Inbox mailbox ID
// First tries to find by role="inbox", then falls back to name="Inbox"
func (c *JMAPClient) FindInboxMailbox(ctx context.Context) (string, error) {
	mailboxes, err := c.GetMailboxes(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to find inbox mailbox: %w", err)
	}

	// First try to find by role
	for _, mbox := range mailboxes {
		if mbox.Role != nil && *mbox.Role == "inbox" {
			return mbox.ID, nil
		}
	}

	// Fallback to name
	for _, mbox := range mailboxes {
		if mbox.Name == "Inbox" {
			return mbox.ID, nil
		}
	}

	return "", fmt.Errorf("inbox mailbox not found in %d mailboxes", len(mailboxes))
}

// FetchEmails fetches recent emails from the Inbox
// Returns emails, query state, and error
func (c *JMAPClient) FetchEmails(ctx context.Context, limit int) ([]models.Email, string, error) {
	if c.accountID == "" {
		return nil, "", fmt.Errorf("not connected - call Connect() first")
	}

	inboxID, err := c.FindInboxMailbox(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find inbox for email fetch: %w", err)
	}

	req := JMAPRequest{
		Using: []string{JMAPCoreCapability, JMAPMailCapability},
		MethodCalls: [][]interface{}{
			{
				"Email/query",
				map[string]interface{}{
					"accountId": c.accountID,
					"filter": map[string]interface{}{
						"inMailbox": inboxID,
					},
					"sort": []map[string]interface{}{
						{
							"property":    "receivedAt",
							"isAscending": false,
						},
					},
					"limit": limit,
				},
				"call2",
			},
		},
	}

	var resp JMAPResponse
	if err := c.call(ctx, req, &resp); err != nil {
		return nil, "", fmt.Errorf("failed to query emails: %w", err)
	}

	if len(resp.MethodResponses) == 0 {
		return nil, "", fmt.Errorf("no response from JMAP server for Email/query")
	}

	queryResponse := resp.MethodResponses[0]
	if len(queryResponse) < 2 {
		return nil, "", fmt.Errorf("invalid JMAP response format from Email/query")
	}

	queryData, ok := queryResponse[1].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP query data format from Email/query")
	}

	queryState, _ := queryData["queryState"].(string)
	idsInterface, ok := queryData["ids"].([]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP email IDs format from Email/query")
	}

	var ids []string
	for _, id := range idsInterface {
		if idStr, ok := id.(string); ok {
			ids = append(ids, idStr)
		}
	}

	if len(ids) == 0 {
		return []models.Email{}, queryState, nil
	}

	// Fetch email details
	req = JMAPRequest{
		Using: []string{JMAPCoreCapability, JMAPMailCapability},
		MethodCalls: [][]interface{}{
			{
				"Email/get",
				map[string]interface{}{
					"accountId":  c.accountID,
					"ids":        ids,
					"properties": []string{
						"id", "messageId", "threadId", "subject",
						"receivedAt", "from", "to", "bodyValues",
						"mailboxIds",
					},
				},
				"call3",
			},
		},
	}

	var getEmailResp JMAPResponse
	if err := c.call(ctx, req, &getEmailResp); err != nil {
		return nil, "", fmt.Errorf("failed to get email details: %w", err)
	}

	if len(getEmailResp.MethodResponses) == 0 {
		return nil, "", fmt.Errorf("no response from JMAP server for Email/get")
	}

	getEmailResponse := getEmailResp.MethodResponses[0]
	if len(getEmailResponse) < 2 {
		return nil, "", fmt.Errorf("invalid JMAP response format from Email/get")
	}

	emailData, ok := getEmailResponse[1].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP email data format from Email/get")
	}

	emailsData, ok := emailData["data"].([]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP emails data format from Email/get")
	}

	var emails []models.Email
	for _, emailData := range emailsData {
		emailMap, ok := emailData.(map[string]interface{})
		if !ok {
			continue
		}

		emailJSON, err := json.Marshal(emailMap)
		if err != nil {
			log.Printf("[jmap] warning: failed to marshal email data: %v", err)
			continue
		}

		var jmapEmail JMAPEmail
		if err := json.Unmarshal(emailJSON, &jmapEmail); err != nil {
			log.Printf("[jmap] warning: failed to unmarshal email: %v", err)
			continue
		}

		email := convertJMAPToEmail(jmapEmail, "Inbox")
		emails = append(emails, email)
	}

	return emails, queryState, nil
}

// FetchEmailsSince fetches emails incrementally since a given query state
// Returns emails, new query state, and error
func (c *JMAPClient) FetchEmailsSince(ctx context.Context, state string, limit int) ([]models.Email, string, error) {
	if c.accountID == "" {
		return nil, "", fmt.Errorf("not connected - call Connect() first")
	}

	inboxID, err := c.FindInboxMailbox(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find inbox for incremental email fetch: %w", err)
	}

	req := JMAPRequest{
		Using: []string{JMAPCoreCapability, JMAPMailCapability},
		MethodCalls: [][]interface{}{
			{
				"Email/query",
				map[string]interface{}{
					"accountId": c.accountID,
					"filter": map[string]interface{}{
						"inMailbox":       inboxID,
						"afterQueryState": state,
					},
					"sort": []map[string]interface{}{
						{
							"property":    "receivedAt",
							"isAscending": false,
						},
					},
					"limit": limit,
				},
				"call2",
			},
		},
	}

	var resp JMAPResponse
	if err := c.call(ctx, req, &resp); err != nil {
		return nil, "", fmt.Errorf("failed to query emails since state %s: %w", state, err)
	}

	if len(resp.MethodResponses) == 0 {
		return nil, "", fmt.Errorf("no response from JMAP server for Email/query")
	}

	queryResponse := resp.MethodResponses[0]
	if len(queryResponse) < 2 {
		return nil, "", fmt.Errorf("invalid JMAP response format from Email/query")
	}

	queryData, ok := queryResponse[1].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP query data format from Email/query")
	}

	newState, _ := queryData["queryState"].(string)
	idsInterface, ok := queryData["ids"].([]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP email IDs format from Email/query")
	}

	var ids []string
	for _, id := range idsInterface {
		if idStr, ok := id.(string); ok {
			ids = append(ids, idStr)
		}
	}

	if len(ids) == 0 {
		return []models.Email{}, newState, nil
	}

	// Fetch email details
	req = JMAPRequest{
		Using: []string{JMAPCoreCapability, JMAPMailCapability},
		MethodCalls: [][]interface{}{
			{
				"Email/get",
				map[string]interface{}{
					"accountId":  c.accountID,
					"ids":        ids,
					"properties": []string{
						"id", "messageId", "threadId", "subject",
						"receivedAt", "from", "to", "bodyValues",
						"mailboxIds",
					},
				},
				"call3",
			},
		},
	}

	var getEmailResp JMAPResponse
	if err := c.call(ctx, req, &getEmailResp); err != nil {
		return nil, "", fmt.Errorf("failed to get email details: %w", err)
	}

	if len(getEmailResp.MethodResponses) == 0 {
		return nil, "", fmt.Errorf("no response from JMAP server for Email/get")
	}

	getEmailResponse := getEmailResp.MethodResponses[0]
	if len(getEmailResponse) < 2 {
		return nil, "", fmt.Errorf("invalid JMAP response format from Email/get")
	}

	emailData, ok := getEmailResponse[1].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP email data format from Email/get")
	}

	emailsData, ok := emailData["data"].([]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid JMAP emails data format from Email/get")
	}

	var emails []models.Email
	for _, emailData := range emailsData {
		emailMap, ok := emailData.(map[string]interface{})
		if !ok {
			continue
		}

		emailJSON, err := json.Marshal(emailMap)
		if err != nil {
			log.Printf("[jmap] warning: failed to marshal email data: %v", err)
			continue
		}

		var jmapEmail JMAPEmail
		if err := json.Unmarshal(emailJSON, &jmapEmail); err != nil {
			log.Printf("[jmap] warning: failed to unmarshal email: %v", err)
			continue
		}

		email := convertJMAPToEmail(jmapEmail, "Inbox")
		emails = append(emails, email)
	}

	return emails, newState, nil
}

// call makes a JMAP API call
func (c *JMAPClient) call(ctx context.Context, req JMAPRequest, resp *JMAPResponse) error {
	if c.apiURL == "" {
		return fmt.Errorf("JMAP client not connected - call Connect() first")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal JMAP request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create JMAP HTTP request: %w", err)
	}

	// Set Bearer token auth
	httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("JMAP HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		if httpResp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("JMAP authentication failed - invalid bearer token")
		}
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("JMAP API returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JMAP response body: %w", err)
	}

	if err := json.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("failed to parse JMAP response: %w", err)
	}

	return nil
}

// convertJMAPToEmail converts a JMAP email to models.Email
func convertJMAPToEmail(jmapEmail JMAPEmail, folder string) models.Email {
	email := models.Email{
		MessageID: "",
		Folder:    &folder,
		Status:    "unprocessed",
	}

	// Extract message ID
	if len(jmapEmail.MessageID) > 0 {
		email.MessageID = jmapEmail.MessageID[0]
	}

	// Extract thread ID
	if jmapEmail.ThreadID != "" {
		email.ThreadID = &jmapEmail.ThreadID
	}

	// Extract subject
	if jmapEmail.Subject != nil {
		email.Subject = jmapEmail.Subject
	}

	// Extract from address
	if jmapEmail.From.Email != "" {
		email.FromAddress = &jmapEmail.From.Email
	}

	// Extract from name
	if jmapEmail.From.Name != "" {
		email.FromName = &jmapEmail.From.Name
	}

	// Extract to addresses
	if len(jmapEmail.To) > 0 {
		var toAddresses []string
		for _, addr := range jmapEmail.To {
			if addr.Name != "" {
				toAddresses = append(toAddresses, fmt.Sprintf("%s <%s>", addr.Name, addr.Email))
			} else {
				toAddresses = append(toAddresses, addr.Email)
			}
		}
		toStr := strings.Join(toAddresses, ", ")
		email.ToAddresses = &toStr
	}

	// Extract body text
	if bodyValue, ok := jmapEmail.BodyValues["text"]; ok && bodyValue.Value != nil {
		email.BodyText = bodyValue.Value
	}

	// Extract received at
	if !jmapEmail.ReceivedAt.IsZero() {
		email.ReceivedAt = &jmapEmail.ReceivedAt
	}

	return email
}
