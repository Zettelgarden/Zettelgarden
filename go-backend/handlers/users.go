package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Handler) GetUserAdminRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	user, err := s.QueryUser(userID)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if user.IsAdmin {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}

}

// admin protected
func (s *Handler) GetUserRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	user, err := s.QueryUser(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Handler) GetUsersRoute(w http.ResponseWriter, r *http.Request) {
	users, err := s.QueryUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)

}

func (s *Handler) UpdateUserRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	user, _ := s.QueryUser(userID)
	if !user.IsAdmin {
		if user.ID != id {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var params models.EditUserParams
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("err? %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	user, err = s.UpdateUser(id, user, params)
	if err != nil {
		log.Printf("?")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)

}

func (s *Handler) CreateUserRoute(w http.ResponseWriter, r *http.Request) {
	var params models.CreateUserParams
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("err? %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	response := models.CreateUserResponse{
		NewID:   0,
		Message: "",
		Error:   false,
	}
	w.Header().Set("Content-Type", "application/json")

	if params.ConfirmPassword != params.Password {
		response.Error = true
		response.Message = "Passwords do not match"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	newID, err := s.CreateUser(params)
	if err != nil {
		response.Error = true
		response.Message = "Passwords do not match"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := s.QueryUser(newID)
	if err != nil {
		response.Error = true
		response.Message = "Unable to fetch created used"
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return

	}
	response.NewID = newID
	response.Message = "Check your email for a validation email"
	response.User = user

	json.NewEncoder(w).Encode(response)
}

func (s *Handler) GetCurrentUserRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)

	user, err := s.QueryUser(userID)
	if err != nil {
		log.Printf("user %v", userID)
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Handler) GetUserSubscriptionRoute(w http.ResponseWriter, r *http.Request) {

	var userSub models.UserSubscription

	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	user, err := s.QueryUser(userID)
	if user.ID != id {
		if !user.IsAdmin {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
	}

	err = s.DB.QueryRow(`
	SELECT 
	id, stripe_customer_id, stripe_subscription_id, 
	stripe_subscription_status,
	stripe_subscription_frequency, stripe_current_plan
	FROM users WHERE id = $1
	`, id).Scan(
		&userSub.ID,
		&userSub.StripeCustomerID,
		&userSub.StripeSubscriptionID,
		&userSub.StripeSubscriptionStatus,
		&userSub.StripeSubscriptionFrequency,
		&userSub.StripeCurrentPlan,
	)
	if err != nil {
		log.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing" {
		userSub.IsActive = true
	} else {
		userSub.IsActive = false
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userSub)
}

func (s *Handler) ResendEmailValidationRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var response models.GenericResponse
	w.Header().Set("Content-Type", "application/json")
	log.Printf("test")

	user, err := s.QueryUser(userID)
	if err != nil {
		response.Message = err.Error()
		response.Error = true
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	if user.EmailValidated {
		response.Message = "email already validated"
		response.Error = true
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	s.sendEmailValidation(user)

	response.Message = "Email sent, check your inbox."
	response.Error = false
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) ValidateEmailRoute(w http.ResponseWriter, r *http.Request) {
	var response models.GenericResponse
	var params models.ValidateEmailParams

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("error email validation json %v", err)
		response.Error = true
		response.Message = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	claims, err := s.decodeToken(params.Token)
	if err != nil {
		log.Printf("error email validation token %v", err)
		response.Error = true
		response.Message = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := s.QueryUser(claims.Sub)
	if err != nil {
		log.Printf("error email validation user %v", err)
		response.Error = true
		response.Message = "Invalid token"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	_, err = s.DB.Exec(`UPDATE users SET email_validated = TRUE WHERE id = $1`, user.ID)
	if err != nil {
		log.Printf("error email validation db %v", err)
		response.Error = true
		response.Message = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	response.Message = "Your email has been validated."
	response.Error = false
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Handler) QueryUsers() ([]models.User, error) {

	users := []models.User{}
	rows, err := s.DB.Query(`
		SELECT
	u.id, u.username, u.email, u.created_at, u.updated_at,
	u.is_admin, u.email_validated, u.can_upload_files,
	u.stripe_subscription_status, u.max_file_storage, u.last_login,
	u.last_seen, u.dashboard_card_pk, u.has_seen_getting_started,
	(SELECT COUNT(*) FROM cards c WHERE c.user_id = u.id) as cards,
	(SELECT COUNT(*) FROM tasks t WHERE t.user_id = u.id) as tasks,
	(SELECT COUNT(*) FROM files f WHERE f.created_by = u.id) as files,
	COALESCE((SELECT SUM(l.cost_usd) FROM llm_query_log l WHERE l.user_id = u.id), 0) as cost,
	COALESCE((SELECT SUM(r.amount_cents) FROM revenue r WHERE r.user_id = u.id), 0) / 100.0 as revenue
	FROM users u
	GROUP BY u.id
	ORDER BY u.id
	`)
	if err != nil {
		return users, err
	}
	defer rows.Close()
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.IsAdmin,
			&user.EmailValidated,
			&user.CanUploadFiles,
			&user.StripeSubscriptionStatus,
			&user.MaxFileStorage,
			&user.LastLogin,
			&user.LastSeen,
			&user.DashboardCardPK,
			&user.HasSeenGettingStarted,
			&user.CardCount,
			&user.TaskCount,
			&user.FileCount,
			&user.LLMCost,
			&user.Revenue,
		); err != nil {
			return users, err
		}
		if user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing" {
			user.IsActive = true
		} else {
			user.IsActive = false
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return users, err
	}
	return users, nil
}

func (s *Handler) QueryUserByEmail(email string) (models.User, error) {

	var user models.User
	err := s.DB.QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started
	FROM users WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsAdmin,
		&user.EmailValidated,
		&user.CanUploadFiles,
		&user.StripeSubscriptionStatus,
		&user.MaxFileStorage,
		&user.LastLogin,
		&user.LastSeen,
		&user.DashboardCardPK,
		&user.HasSeenGettingStarted,
	)
	if err != nil {
		log.Printf("err %v", err)
		return models.User{}, fmt.Errorf("something went wrong")
	}
	if user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing" {
		user.IsActive = true
	} else {
		user.IsActive = false
	}
	return user, nil

}

func (s *Handler) QueryUserByStripeID(stripeID string) (models.User, error) {

	var user models.User
	err := s.DB.QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started
	FROM users WHERE stripe_customer_id = $1
	`, stripeID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsAdmin,
		&user.EmailValidated,
		&user.CanUploadFiles,
		&user.StripeSubscriptionStatus,
		&user.MaxFileStorage,
		&user.LastLogin,
		&user.LastSeen,
		&user.DashboardCardPK,
		&user.HasSeenGettingStarted,
	)
	if err != nil {
		log.Printf("err %v", err)
		return models.User{}, fmt.Errorf("something went wrong")
	}
	if user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing" {
		user.IsActive = true
	} else {
		user.IsActive = false
	}
	return user, nil
}

func (s *Handler) QueryUser(id int) (models.User, error) {
	var user models.User
	err := s.DB.QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started
	FROM users WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsAdmin,
		&user.EmailValidated,
		&user.CanUploadFiles,
		&user.StripeSubscriptionStatus,
		&user.MaxFileStorage,
		&user.LastLogin,
		&user.LastSeen,
		&user.DashboardCardPK,
		&user.HasSeenGettingStarted,
	)
	if err != nil {
		log.Printf("errsd %v", err)
		return models.User{}, fmt.Errorf("something went wrong")
	}
	if user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing" {
		user.IsActive = true
	} else {
		user.IsActive = false
	}
	return user, nil
}

func (s *Handler) UpdateUser(id int, user models.User, params models.EditUserParams) (models.User, error) {
	oldEmail := user.Email

	query := `
	UPDATE users SET username = $1, email = $2, is_admin = $3, updated_at = NOW(),
        dashboard_card_pk = $4, has_seen_getting_started = $5
	WHERE
	id = $6
	`
	_, err := s.DB.Exec(
		query,
		params.Username,
		params.Email,
		params.IsAdmin,
		params.DashboardCardPK,
		params.HasSeenGettingStarted,
		id,
	)
	if err != nil {
		log.Printf("updateuser err %v", err)
		return models.User{}, err
	}
	user, err = s.QueryUser(id)
	if user.Email != oldEmail {
		_, err := s.DB.Exec(`UPDATE users SET email_validated = FALSE WHERE id = $1`, id)
		if err != nil {
			return models.User{}, err
		}
		user, _ := s.QueryUser(id)
		s.sendEmailValidation(user)
	}
	return user, err

}

func (s *Handler) getDefaultDashboardBody() string {
	var path string
	if s.Server.Testing {
		path = "../static/dashboard.md"
	} else {
		path = "./static/dashboard.md"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return ""
	}

	result := string(data)
	return result
}

func (s *Handler) createDefaultCards(userID int) error {
	params := models.EditCardParams{
		CardID: "1",
		Title:  "Welcome to Zettelgarden!",
		Body:   s.getDefaultDashboardBody(),
		Link:   "",
	}
	card, err := s.CreateCard(userID, params)
	if err != nil {
		log.Printf("error creating default cards: %v", err)
		return err
	}
	s.ProcessEntitiesAndFacts(userID, card)
	query := `UPDATE users SET dashboard_card_pk = $1 WHERE id = $2`
	_, err = s.DB.Exec(query, card.ID, userID)
	if err != nil {
		log.Printf("error creating default cards: %v", err)
		return err
	}
	return nil

}

func (s *Handler) createDefaultTags(userID int) error {
	defaultTags := []string{"meeting", "reference", "book", "podcast", "people"}

	for _, tagName := range defaultTags {
		params := models.EditTagParams{
			Name:  tagName,
			Color: "black", // default color
		}
		_, err := s.CreateTag(userID, params)
		if err != nil {
			log.Printf("error creating default tag %s: %v", tagName, err)
			return err
		}
	}
	return nil
}

func (s *Handler) CreateUser(params models.CreateUserParams) (int, error) {
	if params.Email == "" {
		return -1, fmt.Errorf("Email is blank.")
	}
	if params.Username == "" {
		return -1, fmt.Errorf("Username is blank.")
	}
	if params.Password == "" {
		return -1, fmt.Errorf("Password is blank.")
	}

	hashedPassword, err := hashPassword(params.Password)
	if err != nil {
		return -1, fmt.Errorf("Unable to hash password")
	}

	_, err = s.QueryUserByEmail(params.Email)
	if err == nil {
		return -1, fmt.Errorf("Email already exists")
	}

	var newID int
	query := `
	INSERT INTO users (username, email, password, created_at, updated_at,
	stripe_customer_id, stripe_subscription_id, stripe_subscription_status, 
	stripe_subscription_frequency, stripe_current_plan, dashboard_card_pk
	)
	VALUES ($1, $2, $3, NOW(), NOW(), '', '', 'free', '', '', 0) RETURNING id
	`

	err = s.DB.QueryRow(query, params.Username, params.Email, hashedPassword).Scan(&newID)

	err = s.createDefaultCards(newID)
	if err != nil {
		log.Printf("error creating default cards %v", err)
	}

	err = s.createDefaultTags(newID)
	if err != nil {
		log.Printf("error creating default tags %v", err)
	}

	user, _ := s.QueryUser(newID)
	s.sendEmailValidation(user)
	go func() {
		subject := "New user registered at Zettelgarden"
		recipient := "nick@nicksavage.ca"
		body := fmt.Sprintf("A new user has registered at Zettelgarden: %v, %v", params.Username, params.Email)
		s.Server.Mail.SendEmail(subject, recipient, body)
		log.Printf("New user registered: %v, %v", params.Username, params.Email)
	}()
	return newID, err
}

func (s *Handler) sendEmailValidation(user models.User) error {
	host := os.Getenv("ZETTEL_URL")
	token, err := s.generateTempToken(user.ID)
	if err != nil {
		return err
	}

	url := host + "/validate?token=" + token
	messageBody := fmt.Sprintf(`
	Welcome to ZettelGarden, %s.

	Please click the following link to confirm your email address: %s.

	Thank you.
	`, user.Username, url)

	s.Server.Mail.SendEmail("Please confirm your Zettelgarden email", user.Email, messageBody)
	return nil
}

func (s *Handler) UserHasSubscription(userID int) bool {
	if s.Server.Testing {
		return true
	}
	user, err := s.QueryUser(userID)
	if err != nil {
		return false
	}
	return user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing"
}

func (s *Handler) UpdateLastSeen(userID int) error {
	query := `
		UPDATE users 
		SET last_seen = NOW() 
		WHERE id = $1
	`
	_, err := s.DB.Exec(query, userID)
	return err
}

func (s *Handler) UpdateLastSeenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if userID, ok := r.Context().Value("current_user").(int); ok {
			go s.UpdateLastSeen(userID) // Run asynchronously to not block the request
		}
		next.ServeHTTP(w, r)
	}
}
