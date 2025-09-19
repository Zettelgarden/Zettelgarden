package handlers

import (
	"go-backend/models"
	"log"
	"net/http"
	"os"
)

func OpenLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}

func LogRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("current_user").(int)
		if !ok {
			log.Printf("- %s %s", r.Method, r.URL.Path)
		} else {
			log.Printf("- %s %s - user %v", r.Method, r.URL.Path, userID)
		}
		next.ServeHTTP(w, r)

	}
}

func (s *Handler) logCardView(cardPK int, userID int) {
	_, err := s.DB.Exec(`
   INSERT INTO 
   card_views 
   (card_pk, user_id, created_at) 
   VALUES ($1, $2, CURRENT_TIMESTAMP);`, cardPK, userID)

	if err != nil {
		// Log the error
		log.Printf("Error logging card view for cardPK %d and userID %d: %v", cardPK, userID, err)
	}
}

func (s *Handler) LogLastLogin(user models.User) {
	log.Printf("Successfully logged in for userID %v, username %v", user.ID, user.Username)
	_, err := s.DB.Exec(`UPDATE users SET last_login = NOW() WHERE id = $1`, user.ID)
	if err != nil {
		// Log the error
		log.Printf("Error logging card view for userID %v: %v", user.ID, err)
	}

}
