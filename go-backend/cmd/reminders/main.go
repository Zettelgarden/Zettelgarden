package main

import (
	"log"
	"os"

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
		Host:     os.Getenv("MAIL_HOST"),
		Password: os.Getenv("MAIL_PASSWORD"),
		Queue:    mail.NewEmailQueue(),
		DB:       s.DB,
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
