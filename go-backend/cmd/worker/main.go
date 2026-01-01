package main

import (
	"log"
	"time"

	"go-backend/bootstrap"
	"go-backend/services"
)

func main() {
	s := bootstrap.InitServer()
	log.Println("Worker service started")
	log.Println("Task reminder service running - checking every 15 minutes")

	// Create a ticker that fires every 15 minutes
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	// Send reminders immediately on startup
	log.Println("Running initial reminder check...")
	err := services.SendTaskReminders(s.DB, s.Mail)
	if err != nil {
		log.Printf("Error sending reminders: %v", err)
	}

	// Then check every 15 minutes
	for range ticker.C {
		log.Println("Running scheduled reminder check...")
		err := services.SendTaskReminders(s.DB, s.Mail)
		if err != nil {
			log.Printf("Error sending reminders: %v", err)
		}
	}
}
