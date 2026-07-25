package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"
)

func (s *Handler) AddToMailingListRoute(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var request struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	err := s.Server.Mail.HandleAddToMailingList(request.Email)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, "Internal server error: %v", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := struct {
		Email string `json:"email"`
	}{
		Email: request.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) GetAllSubscribers() ([]models.MailingList, error) {
	return s.Server.Mail.GetMailingListSubscribers()
}

// admin protected (via middleware)
func (s *Handler) GetMailingListSubscribersRoute(w http.ResponseWriter, r *http.Request) {
	subscribers, err := s.GetAllSubscribers()
	if err != nil {
		log.Printf("Error getting subscribers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscribers)
}

type SendMailingListRequest struct {
	Subject       string   `json:"subject"`
	Body          string   `json:"body"`
	ToRecipients  []string `json:"to_recipients"`
	BccRecipients []string `json:"bcc_recipients"`
}

// admin protected (via middleware)
func (s *Handler) SendMailingListMessageRoute(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req SendMailingListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Subject == "" || req.Body == "" {
		http.Error(w, "Subject and body are required", http.StatusBadRequest)
		return
	}
	if len(req.ToRecipients) == 0 && len(req.BccRecipients) == 0 {
		http.Error(w, "At least one recipient is required", http.StatusBadRequest)
		return
	}

	// Send the message
	messageID, err := s.Server.Mail.SendMailingListMessage(req.Subject, req.Body, req.ToRecipients, req.BccRecipients)
	if err != nil {
		log.Printf("Error sending mailing list message: %v", err)
		http.Error(w, "Error sending message", http.StatusInternalServerError)
		return
	}

	// Log admin action
	s.LogAdminActionAsync(r, "mailing_list.send_message", "mailing_list_message", messageID, map[string]interface{}{
		"subject":        req.Subject,
		"to_count":       len(req.ToRecipients),
		"bcc_count":      len(req.BccRecipients),
		"to_recipients":  req.ToRecipients,
		"bcc_recipients": req.BccRecipients,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Mailing list message sent successfully",
	})
}

// admin protected (via middleware)
func (s *Handler) GetMailingListMessagesRoute(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 10 // default limit
	}

	messages, err := s.Server.Mail.GetMailingListMessages(limit, offset)
	if err != nil {
		log.Printf("Error getting mailing list messages: %v", err)
		http.Error(w, "Error retrieving messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// admin protected (via middleware)
func (s *Handler) GetMessageRecipientsRoute(w http.ResponseWriter, r *http.Request) {
	// Get message ID from URL parameters
	messageID, err := strconv.Atoi(r.URL.Query().Get("message_id"))
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	recipients, err := s.Server.Mail.GetMessageRecipients(messageID)
	if err != nil {
		log.Printf("Error getting message recipients: %v", err)
		http.Error(w, "Error retrieving recipients", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipients)
}

// admin protected (via middleware)
func (s *Handler) UnsubscribeMailingListRoute(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var request struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the subscriber's status in the database
	query := `
		UPDATE mailing_list
		SET subscribed = false, updated_at = CURRENT_TIMESTAMP
		WHERE email = $1 AND subscribed = true
		RETURNING id
	`
	var id int
	err := s.GetDB().QueryRow(query, request.Email).Scan(&id)
	if err != nil {
		log.Printf("Error unsubscribing email %s: %v", request.Email, err)
		http.Error(w, "Failed to unsubscribe email", http.StatusInternalServerError)
		return
	}

	// Log admin action
	s.LogAdminActionAsync(r, "mailing_list.unsubscribe", "mailing_list", id, map[string]interface{}{
		"email": request.Email,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Successfully unsubscribed %s", request.Email),
	})
}
