package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
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
		return
	}

	if user.IsAdmin {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}

}

// admin protected (via middleware)
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
	// Parse pagination parameters
	page := 1
	perPage := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	users, total, err := s.QueryUsers(page, perPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": (total + perPage - 1) / perPage,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func (s *Handler) UpdateUserRoute(w http.ResponseWriter, r *http.Request) {
	// Authorization handled by AdminOrSelfMiddleware
	// - Admins can update any user
	// - Non-admins can only update themselves
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	user, err := s.QueryUser(userID)
	if err != nil {
		http.Error(w, "Error loading user", http.StatusBadRequest)
		return
	}

	var params models.EditUserParams
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("err? %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Non-admins cannot change admin status
	// Get the target user's current admin status to check if it's being changed
	targetUser, err := s.QueryUser(id)
	if err != nil {
		http.Error(w, "Error loading target user", http.StatusBadRequest)
		return
	}
	if !user.IsAdmin && targetUser.IsAdmin != params.IsAdmin {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Track changes for audit logging
	oldValues := map[string]interface{}{
		"username": targetUser.Username,
		"email":    targetUser.Email,
		"is_admin": targetUser.IsAdmin,
	}

	user, err = s.UpdateUser(id, user, params)
	if err != nil {
		log.Printf("error updating user: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log admin action if the current user is an admin
	if user.IsAdmin {
		details := map[string]interface{}{
			"old": oldValues,
			"new": map[string]interface{}{
				"username": params.Username,
				"email":    params.Email,
				"is_admin": params.IsAdmin,
			},
		}
		s.LogAdminActionAsync(r, "user.update", "user", id, details)
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
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// admin-or-self protected (via middleware)
func (s *Handler) GetUserSubscriptionRoute(w http.ResponseWriter, r *http.Request) {
	var userSub models.UserSubscription

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	err = s.GetDB().QueryRow(`
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
	if userSub.StripeSubscriptionStatus == "active" || userSub.StripeSubscriptionStatus == "trialing" {
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

	_, err = s.GetDB().Exec(`UPDATE users SET email_validated = TRUE WHERE id = $1`, user.ID)
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

func (s *Handler) QueryUsers(page, perPage int) ([]models.User, int, error) {
	users := []models.User{}
	offset := (page - 1) * perPage

	// Get total count for pagination
	var total int
	if err := s.GetDB().QueryRow("SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		return users, 0, err
	}

	rows, err := s.GetDB().Query(`
		SELECT
	u.id, u.username, u.email, u.created_at, u.updated_at,
	u.is_admin, u.email_validated, u.can_upload_files,
	u.stripe_subscription_status, u.max_file_storage, u.last_login,
	u.last_seen, u.dashboard_card_pk, u.has_seen_getting_started,
	COALESCE(u.timezone, 'UTC') as timezone,
	COALESCE(us.card_count, 0) as cards,
	COALESCE(us.task_count, 0) as tasks,
	COALESCE(us.file_count, 0) as files,
	COALESCE(us.chat_message_count, 0) as chat_messages,
	COALESCE(us.llm_cost_usd, 0) as cost,
	COALESCE(us.revenue_cents, 0) / 100.0 as revenue
	FROM users u
	LEFT JOIN user_stats us ON us.user_id = u.id
	ORDER BY u.id
	LIMIT $1 OFFSET $2
	`, perPage, offset)
	if err != nil {
		return users, 0, err
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
			&user.Timezone,
			&user.CardCount,
			&user.TaskCount,
			&user.FileCount,
			&user.ChatMessageCount,
			&user.LLMCost,
			&user.Revenue,
		); err != nil {
			return users, 0, err
		}
		if user.StripeSubscriptionStatus == "active" || user.StripeSubscriptionStatus == "trialing" {
			user.IsActive = true
		} else {
			user.IsActive = false
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return users, 0, err
	}
	return users, total, nil
}

func (s *Handler) QueryUserByEmail(email string) (models.User, error) {

	var user models.User
	err := s.GetDB().QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started, COALESCE(timezone, 'UTC'),
	COALESCE(is_agent, FALSE) as is_agent,
	owner_user_id
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
		&user.Timezone,
		&user.IsAgent,
		&user.OwnerUserID,
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
	err := s.GetDB().QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started, COALESCE(timezone, 'UTC'),
	COALESCE(is_agent, FALSE) as is_agent,
	owner_user_id
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
		&user.Timezone,
		&user.IsAgent,
		&user.OwnerUserID,
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
	err := s.GetDB().QueryRow(`
	SELECT
	id, username, email, password, created_at, updated_at,
	is_admin, email_validated, can_upload_files,
	stripe_subscription_status, max_file_storage, last_login,
	last_seen, dashboard_card_pk, has_seen_getting_started, COALESCE(timezone, 'UTC'),
	COALESCE(is_agent, FALSE) as is_agent,
	owner_user_id
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
		&user.Timezone,
		&user.IsAgent,
		&user.OwnerUserID,
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
	UPDATE users SET username = $1, email = $2, is_admin = $3, updated_at = CURRENT_TIMESTAMP,
        dashboard_card_pk = $4, has_seen_getting_started = $5, timezone = $6
	WHERE
	id = $7
	`
	_, err := s.GetDB().Exec(
		query,
		params.Username,
		params.Email,
		params.IsAdmin,
		params.DashboardCardPK,
		params.HasSeenGettingStarted,
		params.Timezone,
		id,
	)
	if err != nil {
		log.Printf("updateuser err %v", err)
		return models.User{}, err
	}
	user, err = s.QueryUser(id)
	if user.Email != oldEmail {
		_, err := s.GetDB().Exec(`UPDATE users SET email_validated = FALSE WHERE id = $1`, id)
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
	card, err := services.CreateCard(s.GetDB(), userID, params)
	if err != nil {
		log.Printf("error creating default cards: %v", err)
		return err
	}
	s.ProcessEntitiesAndFacts(userID, card)
	query := `UPDATE users SET dashboard_card_pk = $1 WHERE id = $2`
	_, err = s.GetDB().Exec(query, card.ID, userID)
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
		_, err := services.CreateTag(s.GetDB(), userID, params)
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
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, '', '', 'free', '', '', 0) RETURNING id
	`

	err = s.GetDB().QueryRow(query, params.Username, params.Email, hashedPassword).Scan(&newID)
	if err != nil {
		return -1, fmt.Errorf("failed to create user: %w", err)
	}

	err = s.createDefaultCards(newID)
	if err != nil {
		log.Printf("error creating default cards %v", err)
		// Continue anyway - not critical
	}

	err = s.createDefaultTags(newID)
	if err != nil {
		log.Printf("error creating default tags %v", err)
		// Continue anyway - not critical
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
	return services.UserHasSubscription(s.GetDB(), userID)
}

func (s *Handler) UpdateLastSeen(userID int) error {
	query := `
		UPDATE users 
		SET last_seen = CURRENT_TIMESTAMP 
		WHERE id = $1
	`
	_, err := s.GetDB().Exec(query, userID)
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
