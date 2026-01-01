package main

import (
	"log"

	"go-backend/bootstrap"
	"go-backend/services"
)

func main() {
	// 1. Initialize DB and Mailer
	s := bootstrap.InitServer()

	log.Println("Running scheduled reminder check...")

	// 2. Perform the task once
	err := services.SendTaskReminders(s.DB, s.Mail)
	if err != nil {
		log.Fatalf("Error sending reminders: %v", err)
	}

	log.Println("Reminder check complete.")
	// 3. Program exits naturally, releasing resources
}
