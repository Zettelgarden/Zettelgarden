package main

import (
	"log"

	"go-backend/bootstrap"
	"go-backend/mail"
	"go-backend/pkg/config"
	"go-backend/services"
	"go-backend/settings"
)

func main() {
	// 1. Initialize DB and Mailer
	cfg := config.LoadConfig()

	s := bootstrap.InitServer(cfg.Database)

	// Respect the admin-managed mail_enabled toggle (config.yaml) the same
	// way the main server does (6er.16).
	sm, err := settings.New(settings.DefaultPath(cfg.Database.SQLitePath))
	if err != nil {
		log.Fatalf("Failed to load settings file: %v", err)
	}
	smtpConfigured := cfg.Services.Mail.SMTPHost != "" && cfg.Services.Mail.SMTPFrom != ""

	s.Mail = &mail.MailClient{
		SMTPHost:     cfg.Services.Mail.SMTPHost,
		SMTPPort:     cfg.Services.Mail.SMTPPort,
		SMTPUsername: cfg.Services.Mail.SMTPUsername,
		SMTPPassword: cfg.Services.Mail.SMTPPassword,
		SMTPFrom:     cfg.Services.Mail.SMTPFrom,
		StartTLS:     cfg.Services.Mail.StartTLS,
		Queue:        mail.NewEmailQueue(),
		DB:           s.DB,
		Disabled:     !smtpConfigured,
		EnabledFn:    func() bool { return sm.GetBool("mail_enabled") },
	}

	log.Println("Running scheduled reminder check...")

	// 2. Perform the task once
	err = services.SendTaskReminders(s.DB, s.Mail)
	if err != nil {
		log.Fatalf("Error sending reminders: %v", err)
	}

	log.Println("Reminder check complete.")
	// 3. Program exits naturally, releasing resources
}
