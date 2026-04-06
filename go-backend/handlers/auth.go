package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *Handler) Admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("current_user").(int)
		user, err := s.QueryUser(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}
		if !user.IsAdmin {
			http.Error(w, "Access denied", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (s *Handler) JwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("Authorization")

		if tokenStr == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}

		tokenStr = tokenStr[len("Bearer "):]

		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return s.Server.JwtSecretKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "Invalid token signature", http.StatusUnauthorized)
				return
			}

			log.Printf("err 3: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			log.Printf("err 4: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add the claims to the request context
		ctx := context.WithValue(r.Context(), "current_user", claims.Sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
func (s *Handler) generateResetToken(id int) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)

	claims := models.Claims{
		Sub:   id,
		Fresh: true,
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create the token with the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign the token with the secret key
	tokenString, err := token.SignedString(s.Server.JwtSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
func (s *Handler) decodeToken(tokenStr string) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return s.Server.JwtSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (s *Handler) generateAccessToken(userID int) (string, error) {
	expirationTime := time.Now().Add(15 * 24 * time.Hour)

	claims := &models.Claims{
		Sub:   userID,
		Fresh: true,
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.Server.JwtSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *Handler) generateTempToken(userID int) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)

	claims := &models.Claims{
		Sub:   userID,
		Fresh: true,
		Type:  "temp",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.Server.JwtSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *Handler) ResetPasswordRoute(w http.ResponseWriter, r *http.Request) {

	var params models.ResetPasswordParams

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	claims, err := s.decodeToken(params.Token)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, err := s.QueryUser(claims.Sub)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hashedPassword, err := hashPassword(params.NewPassword)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = s.DB.Exec("UPDATE users SET password = $1 WHERE id = $2", hashedPassword, user.ID)
	if err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}

	// Send confirmation email
	messageBody := fmt.Sprintf("Your password has been successfully reset. If you did not request this change, please contact info@zettelgarden.com immediately.")
	err = s.Server.Mail.SendEmail("Password Reset Confirmation", user.Email, messageBody)
	if err != nil {
		// Log the error but don't return it to the user since the password was successfully reset
		log.Printf("Error sending password reset confirmation email: %v", err)
	}

	response := models.ResetPasswordResponse{
		Message: "Your password has been updated",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusOK)
}

func (s *Handler) LoginRoute(w http.ResponseWriter, r *http.Request) {

	var params models.LoginParams
	var response models.LoginResponse
	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, err := s.QueryUserByEmail(params.Email)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !checkPasswordHash(params.Password, user.Password) {
		response.Message = "Invalid credentials"
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	user.Password = "" // Remove password from user data
	response.User = user
	response.AccessToken = accessToken

	json.NewEncoder(w).Encode(response)

	s.LogLastLogin(user)
}

func (s *Handler) CheckTokenRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	user, err := s.QueryUser(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	user.Password = ""

	response := models.LoginResponse{
		User:        user,
		AccessToken: "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) RequestPasswordResetRoute(w http.ResponseWriter, r *http.Request) {
	var params models.RequestPasswordResetParams
	var response models.GenericResponse

	w.Header().Set("Content-Type", "application/json")
	response.Error = false
	response.Message = "If your email is in our system, you will receive a password reset link."

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := s.QueryUserByEmail(params.Email)
	if err != nil {
		log.Printf("user not found")
		json.NewEncoder(w).Encode(response)
		return
	}
	token, err := s.generateTempToken(user.ID)
	if err != nil {
		log.Printf("err %v", err.Error())
		response.Error = true
		response.Message = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	url := fmt.Sprintf("%s/reset?token=%s", os.Getenv("ZETTEL_URL"), token)
	messageBody := fmt.Sprintf("Please go to this link to reset your password: %s", url)

	s.Server.Mail.SendEmail("Please confirm your Zettelgarden email", user.Email, messageBody)
	json.NewEncoder(w).Encode(response)
}

// Dual authentication middleware that supports both JWT tokens and API keys
func (s *Handler) APIKeyOrJWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("Authorization")
		if tokenStr == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		// Try JWT first (existing tokens)
		if userID, err := s.validateJWTToken(tokenStr); err == nil {
			// Valid JWT - proceed with existing flow
			ctx := context.WithValue(r.Context(), "current_user", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If JWT failed, try API key validation
		if userID, apiKeyID, err := s.validateAPIKey(tokenStr); err == nil {
			// Valid API key - proceed with user context
			ctx := context.WithValue(r.Context(), "current_user", userID)
			ctx = context.WithValue(ctx, "api_key_id", apiKeyID) // For usage tracking
			go s.updateAPIKeyLastUsed(apiKeyID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Debug: log authentication failure details without exposing sensitive token data
		log.Printf("DEBUG: Authentication failed for token (length %d)", len(tokenStr))
		if len(tokenStr) == 32 {
			// Looks like an API key, check how many keys exist in DB
			var count int
			s.DB.QueryRow("SELECT COUNT(*) FROM api_keys WHERE is_active = true").Scan(&count)
			log.Printf("DEBUG: Found %d active API keys in database", count)
		}

		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
	}
}

// validateJWTToken validates a JWT token and returns the user ID if valid
func (s *Handler) validateJWTToken(tokenStr string) (int, error) {
	claims := &models.Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return s.Server.JwtSecretKey, nil
	})

	if err != nil {
		return 0, err
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	return claims.Sub, nil
}

// validateAPIKey validates an API key and returns user ID and API key ID if valid
// For agents, the API key ID will be negative to distinguish from regular API keys
func (s *Handler) validateAPIKey(apiKey string) (int, int, error) {
	// First check regular API keys from api_keys table
	rows, err := s.DB.Query(`
		SELECT id, user_id, key_hash
		FROM api_keys
		WHERE is_active = true
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var apiKeyID, userID int
		var keyHash string

		err := rows.Scan(&apiKeyID, &userID, &keyHash)
		if err != nil {
			continue
		}

		// Compare the provided key against the stored hash
		if checkPasswordHash(apiKey, keyHash) {
			return userID, apiKeyID, nil
		}
	}

	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	// If not found in regular API keys, check agent API keys from users table
	agentRows, err := s.DB.Query(`
		SELECT id, api_key_hash, owner_user_id
		FROM users
		WHERE is_agent = TRUE AND api_key_hash IS NOT NULL
	`)
	if err != nil {
		return 0, 0, err
	}
	defer agentRows.Close()

	for agentRows.Next() {
		var agentID int
		var keyHash string
		var ownerUserID sql.NullInt64

		err := agentRows.Scan(&agentID, &keyHash, &ownerUserID)
		if err != nil {
			continue
		}

		// Compare the provided key against the stored hash
		if checkPasswordHash(apiKey, keyHash) {
			// For agents, return agent ID as userID and negative API key ID
			return agentID, -agentID, nil
		}
	}

	if err := agentRows.Err(); err != nil {
		return 0, 0, err
	}

	return 0, 0, fmt.Errorf("invalid api key")
}

// updateAPIKeyLastUsed updates the last_used_at timestamp for an API key
func (s *Handler) updateAPIKeyLastUsed(apiKeyID int) {
	_, err := s.DB.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", apiKeyID)
	if err != nil {
		log.Printf("Error updating api key last_used_at: %v", err)
	}
}

// generateAPIKey creates a cryptographically secure random API key
func generateAPIKey() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	const length = 32

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

// updateAgentLastSeen updates the last_seen timestamp for an agent
func (s *Handler) updateAgentLastSeen(agentID int) {
	_, err := s.DB.Exec("UPDATE users SET last_seen = NOW() WHERE id = $1", agentID)
	if err != nil {
		log.Printf("Error updating agent last_seen: %v", err)
	}
}

