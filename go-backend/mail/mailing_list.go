package mail

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func (m *MailClient) HandleAddToMailingList(email string) error {
	// First check if email already exists
	checkQuery := `
        SELECT id, welcome_email_sent 
        FROM mailing_list 
        WHERE email = $1
    `

	var existingId int
	var sent bool
	err := m.DB.QueryRow(checkQuery, email).Scan(&existingId, &sent)

	// If email is found, return error indicating it already exists
	if err == nil {
		return fmt.Errorf("Email %s is already registered", email)
	}

	// If error is anything other than no rows found, return the error
	if err != sql.ErrNoRows {
		log.Printf("Error checking existing email: %v", err)
		return fmt.Errorf("Error checking existing email: %v", err)
	}

	query := `
            INSERT INTO mailing_list (email)
            VALUES ($1)
            RETURNING id`

	var id int
	err = m.DB.QueryRow(query, email).Scan(&id)
	if err != nil {
		log.Printf("Error adding email to mailing list: %v", err)
		return fmt.Errorf("Error adding email to mailing list: %v", err)
	}

	adminEmail := Email{
		Subject:   "New mailing list registered at Zettelgarden",
		Recipient: "nick@nicksavage.ca",
		Body:      fmt.Sprintf("A new email has registered at for the Zettelgarden mailing list: %v", email),
	}

	m.SendEmail(adminEmail.Subject, adminEmail.Recipient, adminEmail.Body)
	if !sent {
		m.SendWelcomeEmail(email)
	}
	return nil
}
func (m *MailClient) SendWelcomeEmail(email string) error {
	welcomeEmail := Email{
		Subject:   "Welcome to Zettelgarden! 🌱",
		Recipient: email,
		Body: `
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <h2 style="color: #2c5282;">Hi there! 👋</h2>

    <p>Thank you for signing up for updates about Zettelgarden! We're excited to have you join our community of knowledge gardeners.</p>

    <p>Zettelgarden is an open-source project aimed at helping people organize and grow their ideas. If you're interested in checking out the code or contributing, you can find our GitHub repository here: 
        <a href="https://github.com/NickSavage/Zettelgarden" style="color: #4299e1;">https://github.com/NickSavage/Zettelgarden</a>
    </p>

    <p>We'll keep you updated on new features, improvements, and tips for getting the most out of Zettelgarden. Feel free to reply to this email if you have any questions or suggestions.</p>

    <p style="margin-top: 30px;">Happy gardening! 🌳</p>

    <p>Best regards,<br>
    Nick<br>
    Zettelgarden Team</p>

    <p style="font-style: italic; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
        P.S. Want to have a say in Zettelgarden's development? Star us on GitHub ⭐ and join the conversation! 💬
    </p>
</body>
</html>`,
	}

	// Send the welcome email
	err := m.SendHTMLEmail(welcomeEmail.Subject, welcomeEmail.Recipient, welcomeEmail.Body)
	if err != nil {
		log.Printf("Error sending welcome email to %s: %v", email, err)
		return err
	}

	// Update the database to mark welcome email as sent
	updateQuery := `
        UPDATE mailing_list 
        SET welcome_email_sent = true 
        WHERE email = $1
    `

	_, err = m.DB.Exec(updateQuery, email)
	if err != nil {
		log.Printf("Error updating welcome_email_sent status: %v", err)
		return fmt.Errorf("Error updating welcome email status: %v", err)
	}

	return nil
}

func (m *MailClient) SendMailingListMessage(subject, body string, toRecipients, bccRecipients []string) (int, error) {
	// Start a transaction
	tx, err := m.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Validate recipients
	if len(toRecipients) == 0 && len(bccRecipients) == 0 {
		return 0, fmt.Errorf("at least one recipient is required")
	}

	// Convert markdown to HTML
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(body))

	// Create HTML renderer with options
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{
		Flags: htmlFlags,
	}
	renderer := html.NewRenderer(opts)
	htmlBody := string(markdown.Render(doc, renderer))

	// Wrap the HTML in a styled template
	htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    %s
</body>
</html>`, htmlBody)

	// Log recipient counts
	log.Printf("Sending message '%s' to %d TO recipients and %d BCC recipients",
		subject, len(toRecipients), len(bccRecipients))

	// Insert the message (store original markdown)
	var messageID int
	err = tx.QueryRow(`
		INSERT INTO mailing_list_messages (subject, body, total_recipients)
		VALUES ($1, $2, $3)
		RETURNING id
	`, subject, body, len(toRecipients)+len(bccRecipients)).Scan(&messageID)
	if err != nil {
		return 0, fmt.Errorf("error creating message record: %v", err)
	}

	// Insert recipient records and send emails
	for _, email := range toRecipients {
		log.Printf("Sending to TO recipient: %s", email)
		_, err = tx.Exec(`
			INSERT INTO mailing_list_recipients (message_id, recipient_email, recipient_type)
			VALUES ($1, $2, 'to')
		`, messageID, email)
		if err != nil {
			return 0, fmt.Errorf("error recording TO recipient %s: %v", email, err)
		}

		err = m.SendHTMLEmail(subject, email, htmlBody)
		if err != nil {
			return 0, fmt.Errorf("error sending to TO recipient %s: %v", email, err)
		}
	}

	for _, email := range bccRecipients {
		log.Printf("Sending to BCC recipient: %s", email)
		_, err = tx.Exec(`
			INSERT INTO mailing_list_recipients (message_id, recipient_email, recipient_type)
			VALUES ($1, $2, 'bcc')
		`, messageID, email)
		if err != nil {
			return 0, fmt.Errorf("error recording BCC recipient %s: %v", email, err)
		}

		err = m.SendHTMLEmail(subject, email, htmlBody)
		if err != nil {
			return 0, fmt.Errorf("error sending to BCC recipient %s: %v", email, err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("error committing transaction: %v", err)
	}

	log.Printf("Successfully sent message ID %d to %d recipients",
		messageID, len(toRecipients)+len(bccRecipients))

	return messageID, nil
}

func (m *MailClient) GetMailingListMessages(limit, offset int) ([]models.MailingListMessage, error) {
	query := `
		SELECT id, subject, body, sent_at, total_recipients
		FROM mailing_list_messages
		ORDER BY sent_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := m.DB.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error querying messages: %v", err)
	}
	defer rows.Close()

	var messages []models.MailingListMessage
	for rows.Next() {
		var msg models.MailingListMessage
		err := rows.Scan(&msg.ID, &msg.Subject, &msg.Body, &msg.SentAt, &msg.TotalRecipients)
		if err != nil {
			return nil, fmt.Errorf("error scanning message: %v", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (m *MailClient) GetMessageRecipients(messageID int) ([]models.MailingListRecipient, error) {
	query := `
		SELECT id, message_id, recipient_email, recipient_type, sent_at
		FROM mailing_list_recipients
		WHERE message_id = $1
		ORDER BY sent_at ASC
	`

	rows, err := m.DB.Query(query, messageID)
	if err != nil {
		return nil, fmt.Errorf("error querying recipients: %v", err)
	}
	defer rows.Close()

	var recipients []models.MailingListRecipient
	for rows.Next() {
		var r models.MailingListRecipient
		err := rows.Scan(&r.ID, &r.MessageID, &r.RecipientEmail, &r.RecipientType, &r.SentAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning recipient: %v", err)
		}
		recipients = append(recipients, r)
	}

	return recipients, nil
}

func (m *MailClient) GetMailingListSubscribers() ([]models.MailingList, error) {
	query := `
		SELECT ml.id, ml.email, ml.created_at, ml.updated_at, 
		       ml.welcome_email_sent, ml.subscribed,
		       CASE WHEN u.id IS NOT NULL THEN true ELSE false END as has_account
		FROM mailing_list ml
		LEFT JOIN users u ON LOWER(ml.email) = LOWER(u.email)
		ORDER BY ml.created_at DESC
	`

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying mailing list: %v", err)
	}
	defer rows.Close()

	var subscribers []models.MailingList
	for rows.Next() {
		var s models.MailingList
		err := rows.Scan(&s.ID, &s.Email, &s.CreatedAt, &s.UpdatedAt,
			&s.WelcomeEmailSent, &s.Subscribed, &s.HasAccount)
		if err != nil {
			return nil, fmt.Errorf("error scanning subscriber: %v", err)
		}
		subscribers = append(subscribers, s)
	}

	return subscribers, nil
}
