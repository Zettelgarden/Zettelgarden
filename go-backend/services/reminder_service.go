package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

	frontendURL := os.Getenv("ZETTEL_URL")
	if frontendURL == "" {
		frontendURL = "https://zettelgarden.com"
	}

	for _, task := range tasks {
		// Get user email
		var userEmail string
		err := db.QueryRow("SELECT email FROM users WHERE id = $1", task.UserID).Scan(&userEmail)
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

		// Build email body
		subject := fmt.Sprintf("Reminder: %s", task.Title)
		body := buildReminderEmailBody(task, frontendURL)

		// Send email
		err = mailClient.SendHTMLEmail(subject, userEmail, body)
		if err != nil {
			log.Printf("Failed to send reminder email for task %d: %v", task.ID, err)
			failCount++
			continue
		}

		// Mark reminder as sent
		_, err = db.Exec("UPDATE tasks SET reminder_sent = TRUE WHERE id = $1", task.ID)
		if err != nil {
			log.Printf("Failed to mark reminder as sent for task %d: %v", task.ID, err)
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
func buildReminderEmailBody(task models.Task, frontendURL string) string {
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
		body += fmt.Sprintf(`            <p><span class="info-label">Due:</span> %s</p>`, task.DueDate.Format("Monday, January 2, 2006 at 3:04 PM"))
	}

	// Add scheduled date if it exists
	if task.ScheduledDate != nil {
		body += fmt.Sprintf(`            <p><span class="info-label">Scheduled:</span> %s</p>`, task.ScheduledDate.Format("Monday, January 2, 2006 at 3:04 PM"))
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
