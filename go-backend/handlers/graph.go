package handlers

import (
	"encoding/json"
	"go-backend/services"
	"log"
	"net/http"
)

// GetGraphRoute returns the current user's knowledge graph (card/entity/tag
// nodes plus reference/parent/entity/tag edges), optionally filtered by the
// ?types=card,entity,tag query parameter.
func (s *Handler) GetGraphRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	types := services.ParseGraphTypes(r.URL.Query().Get("types"))

	data, err := services.GetGraphData(s.GetDB(), userID, types)
	if err != nil {
		log.Printf("Failed to get graph data: %v", err)
		http.Error(w, "Failed to get graph data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode graph data: %v", err)
		return
	}
}
