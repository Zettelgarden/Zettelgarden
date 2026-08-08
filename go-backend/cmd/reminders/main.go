package main

import (
	"log"

	"go-backend/bootstrap"
	"go-backend/mail"
	"go-backend/pkg/config"
	"go-backend/services"
)

func main() {
	// 1. Initialize DB and Mailer
	cfg := config.LoadConfig()

	s := bootstrap.InitServer(cfg.Database)

	s.Mail = &mail.MailClient{
		SMTPHost:     cfg.Services.Mail.SMTPHost,
		SMTPPort:     cfg.Services.Mail.SMTPPort,
		SMTPUsername: cfg.Services.Mail.SMTPUsername,
		SMTPPassword: cfg.Services.Mail.SMTPPassword,
		SMTPFrom:     cfg.Services.Mail.SMTPFrom,
		StartTLS:     cfg.Services.Mail.StartTLS,
		Queue:        mail.NewEmailQueue(),
		DB:           s.DB,
	}

	log.Println("Running scheduled reminder check...")

	// 2. Perform the task once
	err := services.SendTaskReminders(s.DB, s.Mail)
	if err != nil {
		log.Fatalf("Error sending reminders: %v", err)
	}

	log.Println("Reminder check complete.")
	// 3. Program exits naturally, releasing resources
}
