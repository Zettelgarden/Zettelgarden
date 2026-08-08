package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"go-backend/mail"
	"go-backend/models"
)

// SendTaskReminders sends email reminders for all tasks that need them
func SendTaskReminders(db *sql.DB, mailClient *mail.MailClient) error {
	// Get all tasks needing reminders
	tasks, err := GetTasksNeedingReminders(db)
	if err != nil {
		return fmt.Errorf("failed to get tasks needing reminders: %v", err)
	}

	if len(tasks) == 0 {
		log.Println("No tasks need reminders at this time")
		return nil
	}

	log.Printf("Found %d task(s) needing reminders", len(tasks))
	successCount := 0
	failCount := 0

	// ZETTEL_URL is required config, so the reminder links should always be
	// buildable; never fall back to a SaaS host (6.7).
	frontendURL := os.Getenv("ZETTEL_URL")
	if frontendURL == "" {
		log.Printf("WARNING: ZETTEL_URL is empty; task reminder email links will be incomplete")
	}

	for _, task := range tasks {
		// Get user email and timezone
		var userEmail string
		var userTimezone sql.NullString
		err := db.QueryRow("SELECT email, timezone FROM users WHERE id = $1", task.UserID).Scan(&userEmail, &userTimezone)
		if err != nil {
			log.Printf("Failed to get email for user %d: %v", task.UserID, err)
			failCount++
			continue
		}

		if userEmail == "" {
			log.Printf("User %d has no email address, skipping reminder for task %d", task.UserID, task.ID)
			failCount++
			continue
		}

		// Get user's timezone, default to UTC if not set
		timezone := "UTC"
		if userTimezone.Valid && userTimezone.String != "" {
			timezone = userTimezone.String
		}

		// Build email body
		subject := fmt.Sprintf("Reminder: %s", task.Title)
		body := buildReminderEmailBody(task, frontendURL, timezone)

		// Send email
		err = mailClient.SendHTMLEmail(subject, userEmail, body)
		if err != nil {
			log.Printf("Failed to send reminder email for task %d: %v", task.ID, err)
			failCount++
			continue
		}

		// Mark reminder as sent
		_, err = db.Exec("UPDATE tasks SET reminder_sent = TRUE, version = version + 1 WHERE id = $1", task.ID)
		if err != nil {
			log.Printf("Failed to mark reminder as sent for task %d: %v", task.ID, err)
			failCount++
			continue
		}
		if err := emitRowChange(db, task.UserID, SyncCollectionTasks, task.ID, SyncOpUpsert); err != nil {
			log.Printf("Failed to record sync change for task %d: %v", task.ID, err)
			failCount++
			continue
		}

		log.Printf("Sent reminder for task %d ('%s') to %s", task.ID, task.Title, userEmail)
		successCount++
	}

	log.Printf("Reminder processing complete: %d sent, %d failed", successCount, failCount)
	return nil
}

// buildReminderEmailBody creates the HTML email body for a task reminder
func buildReminderEmailBody(task models.Task, frontendURL string, timezone string) string {
	// Load user's timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("Failed to load timezone %s, falling back to UTC: %v", timezone, err)
		loc = time.UTC
	}
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .task-title { font-size: 18px; font-weight: bold; color: #2c3e50; margin-bottom: 10px; }
        .task-info { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 15px 0; }
        .info-label { font-weight: bold; color: #555; }
        .button { display: inline-block; padding: 10px 20px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px; margin-top: 15px; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #777; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Task Reminder</h2>
        <p>Hi,</p>
        <p>This is a reminder for your task:</p>

        <div class="task-info">
            <div class="task-title">%s</div>
`, task.Title)

	// Add due date if it exists
	if task.DueDate != nil {
		localTime := task.DueDate.In(loc)
		body += fmt.Sprintf(`            <p><span class="info-label">Due:</span> %s (%s)</p>`, localTime.Format("Monday, January 2, 2006 at 3:04 PM"), timezone)
	}

	// Add scheduled date if it exists
	if task.ScheduledDate != nil {
		localTime := task.ScheduledDate.In(loc)
		body += fmt.Sprintf(`            <p><span class="info-label">Scheduled:</span> %s (%s)</p>`, localTime.Format("Monday, January 2, 2006 at 3:04 PM"), timezone)
	}

	// Add priority if it exists
	if task.Priority != nil && *task.Priority != "" {
		body += fmt.Sprintf(`            <p><span class="info-label">Priority:</span> %s</p>`, *task.Priority)
	}

	body += `        </div>

        <a href="` + frontendURL + `/app/tasks?taskId=` + fmt.Sprintf("%d", task.ID) + `" class="button">View Task</a>

        <div class="footer">
            <p>You're receiving this because you set a reminder for this task in Zettelgarden.</p>
        </div>
    </div>
</body>
</html>
`

	return body
}
