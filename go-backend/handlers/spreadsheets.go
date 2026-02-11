package handlers

import (
	"encoding/json"
	"go-backend/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetSpreadsheetsRoute handles GET /api/cards/:cardId/spreadsheets
// Returns all spreadsheets attached to a specific card
func (s *Handler) GetSpreadsheetsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["cardId"])
	if err != nil {
		log.Printf("Invalid card ID param: %v", err)
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify user owns the card
	_, err = s.QueryFullCard(userID, cardID)
	if err != nil {
		log.Printf("Card not found for user %d: %v", userID, err)
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	spreadsheets, err := models.GetSpreadsheetsByCardID(s.GetDB(), cardID, userID)
	if err != nil {
		log.Printf("Error querying spreadsheets for card %d: %v", cardID, err)
		http.Error(w, "Error retrieving spreadsheets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spreadsheets)
}

// GetSpreadsheetRoute handles GET /api/spreadsheets/:id
// Returns a single spreadsheet by ID
func (s *Handler) GetSpreadsheetRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid spreadsheet ID param: %v", err)
		http.Error(w, "Invalid spreadsheet ID", http.StatusBadRequest)
		return
	}

	spreadsheet, err := models.GetSpreadsheetByID(s.GetDB(), id, userID)
	if err != nil {
		log.Printf("Spreadsheet not found for id %d: %v", id, err)
		http.Error(w, "Spreadsheet not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spreadsheet)
}

// CreateSpreadsheetRoute handles POST /api/cards/:cardId/spreadsheets
// Creates a new spreadsheet attached to a card
func (s *Handler) CreateSpreadsheetRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardID, err := strconv.Atoi(mux.Vars(r)["cardId"])
	if err != nil {
		log.Printf("Invalid card ID param: %v", err)
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	// Verify user owns the card
	_, err = s.QueryFullCard(userID, cardID)
	if err != nil {
		log.Printf("Card not found for user %d: %v", userID, err)
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	// Parse request body
	var params struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Error decoding create spreadsheet request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Default name to "sheet1" if not provided
	name := params.Name
	if name == "" {
		name = "sheet1"
	}

	// Create spreadsheet with 5x5 default data
	spreadsheet := &models.Spreadsheet{
		UserID: userID,
		CardID: cardID,
		Name:   name,
		Rows:   5,
		Cols:   5,
		Data: models.SpreadsheetData{
			Rows: 5,
			Cols: 5,
			Data: make(map[string]models.SpreadsheetCell),
		},
	}

	if err := models.CreateSpreadsheet(s.GetDB(), spreadsheet); err != nil {
		log.Printf("Error creating spreadsheet: %v", err)
		http.Error(w, "Error creating spreadsheet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(spreadsheet)
}

// UpdateSpreadsheetRoute handles PUT /api/spreadsheets/:id
// Updates an existing spreadsheet's data
func (s *Handler) UpdateSpreadsheetRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid spreadsheet ID param: %v", err)
		http.Error(w, "Invalid spreadsheet ID", http.StatusBadRequest)
		return
	}

	// Verify spreadsheet exists and user owns it
	_, err = models.GetSpreadsheetByID(s.GetDB(), id, userID)
	if err != nil {
		log.Printf("Spreadsheet not found for id %d: %v", id, err)
		http.Error(w, "Spreadsheet not found", http.StatusNotFound)
		return
	}

	// Parse SpreadsheetData from request body
	var data models.SpreadsheetData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("Error decoding spreadsheet data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := models.UpdateSpreadsheet(s.GetDB(), id, userID, data); err != nil {
		log.Printf("Error updating spreadsheet %d: %v", id, err)
		http.Error(w, "Error updating spreadsheet", http.StatusInternalServerError)
		return
	}

	// Return the updated spreadsheet
	spreadsheet, err := models.GetSpreadsheetByID(s.GetDB(), id, userID)
	if err != nil {
		log.Printf("Error retrieving updated spreadsheet %d: %v", id, err)
		http.Error(w, "Error retrieving spreadsheet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spreadsheet)
}

// DeleteSpreadsheetRoute handles DELETE /api/spreadsheets/:id
// Deletes a spreadsheet
func (s *Handler) DeleteSpreadsheetRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		log.Printf("Invalid spreadsheet ID param: %v", err)
		http.Error(w, "Invalid spreadsheet ID", http.StatusBadRequest)
		return
	}

	if err := models.DeleteSpreadsheet(s.GetDB(), id, userID); err != nil {
		log.Printf("Error deleting spreadsheet %d: %v", id, err)
		if err == models.ErrSpreadsheetNotFound {
			http.Error(w, "Spreadsheet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error deleting spreadsheet", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
