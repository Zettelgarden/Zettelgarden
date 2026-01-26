package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/mail"
	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/server"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

var S *server.Server

var setupOnce sync.Once
var db *sql.DB

// setTestEnvironmentVariables sets required environment variables for testing
func setTestEnvironmentVariables() {
	// Database config - these will already be set in real test environment
	setEnvIfNotSet("DB_HOST", "localhost")
	setEnvIfNotSet("DB_PORT", "5432")
	setEnvIfNotSet("DB_USER", "postgres")
	setEnvIfNotSet("DB_PASS", "password")
	setEnvIfNotSet("DB_NAME", "zettelgarden")

	// Server config - test defaults
	setEnvIfNotSet("ZETTEL_DEV", "true")
	setEnvIfNotSet("ZETTEL_PORT", "8080")
	setEnvIfNotSet("ZETTEL_URL", "http://localhost:8080")
	setEnvIfNotSet("ZETTEL_ADMIN_EMAIL", "admin@test.com")
	setEnvIfNotSet("SECRET_KEY", "test-secret-key-for-jwt-signing-32-chars-minimum")
	setEnvIfNotSet("ZETTEL_BACKEND_LOG_LOCATION", "")

	// LLM service config - test defaults
	setEnvIfNotSet("ZETTEL_LLM_KEY", "test-openai-api-key")
	setEnvIfNotSet("ZETTEL_LLM_ENDPOINT", "https://api.openai.com/v1")
	setEnvIfNotSet("ZETTEL_LLM_DEFAULT_MODEL", "gpt-3.5-turbo")
	setEnvIfNotSet("ZETTEL_LLM_SUMMARIZE_MODEL", "gpt-3.5-turbo")

	// Mail service config - test defaults
	setEnvIfNotSet("MAIL_HOST", "smtp.gmail.com")
	setEnvIfNotSet("MAIL_PASSWORD", "test-mail-password")

	// Stripe config - test defaults
	setEnvIfNotSet("STRIPE_SECRET_KEY", "test-stripe-secret-key")
	setEnvIfNotSet("STRIPE_PUBLISHABLE_KEY", "test-stripe-publishable-key")
	setEnvIfNotSet("STRIPE_WEBHOOK_SECRET", "test-stripe-webhook-secret")
	setEnvIfNotSet("STRIPE_MONTH_PRICE", "price_monthly_test_id")
	setEnvIfNotSet("STRIPE_YEAR_PRICE", "price_yearly_test_id")
	setEnvIfNotSet("STRIPE_BILLING_URL", "https://billing.stripe.com/test")

	// S3/B2 config - test defaults
	setEnvIfNotSet("B2_ACCESS_KEY_ID", "test-access-key-id")
	setEnvIfNotSet("B2_SECRET_ACCESS_KEY", "test-secret-access-key")
	setEnvIfNotSet("B2_BUCKET_NAME", "test-bucket-name")

	// GitHub OAuth config - test defaults
	setEnvIfNotSet("GITHUB_CLIENT_ID", "test-github-client-id")
	setEnvIfNotSet("GITHUB_CLIENT_SECRET", "test-github-client-secret")
	setEnvIfNotSet("GITHUB_REDIRECT_URI", "http://localhost:8080/auth/github/callback")

	// Search/Typesense config - test defaults
	setEnvIfNotSet("TYPESENSE_HOST", "http://localhost:8108")
	setEnvIfNotSet("TYPESENSE_PASSWORD", "test-typesense-password")
	setEnvIfNotSet("TYPESENSE_COLLECTION", "zettelgarden_test")

	// Optional features
	setEnvIfNotSet("ZETTEL_RUN_CHUNKING_EMBEDDING", "false")
}

// setEnvIfNotSet sets an environment variable only if it's not already set
func setEnvIfNotSet(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

func Setup() *server.Server {
	var err error
	setupOnce.Do(func() {
		// Set test environment variables if not already set
		//		setTestEnvironmentVariables()

		dbConfig := models.DatabaseConfig{}
		dbConfig.Host = os.Getenv("DB_HOST")
		dbConfig.Port = os.Getenv("DB_PORT")
		dbConfig.User = os.Getenv("DB_USER")
		dbConfig.Password = os.Getenv("DB_PASS")
		dbConfig.DatabaseName = "zettelkasten_testing"

		config.LoadConfig()
		db, err = server.ConnectToDatabase(dbConfig)
		if err != nil {
			log.Fatalf("Unable to connect to the database: %v\n", err)
		}

		S = &server.Server{}
		S.DB = db
		S.Testing = true
		S.SchemaDir = "../schema"

		S.Mail = &mail.MailClient{
			Testing:           true,
			TestingEmailsSent: 0,
			DB:                db,
		}
		S.TestInspector = &server.TestInspector{}
		S.LLMClient = &models.LLMClient{Testing: true}

		// Reset database and run migrations ONLY once at the beginning of the test suite
		// We explicitly call ResetDatabase here to ensure a clean slate
		if err := server.ResetDatabase(S); err != nil {
			log.Fatalf("Failed to reset database: %v\n", err)
		}
		server.RunMigrations(S)
	})

	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}

	// Import test data for each test (data gets cleared in Teardown)
	err = importTestData(S)
	if err != nil {
		log.Fatal(err)
	}
	return S
}

func Teardown() {
	// Fast cleanup: truncate tables instead of dropping and re-migrating
	// Only truncate if the database has been initialized
	if S != nil && S.DB != nil {
		truncateTestData()
	}
}

// truncateTestData clears all test data but keeps the schema.
// This is much faster than dropping tables and re-migrating.
func truncateTestData() {
	tables := []string{
		"admin_audit_log",
		"api_keys",
		"audit_events",
		"backlinks",
		"card_chunks",
		"card_tags",
		"card_templates",
		"card_views",
		"cards",
		"chat_messages",
		"chat_tool_calls",
		"chat_usage_quotas",
		"chat_conversations",
		"entity_card_junction",
		"entity_fact_junction",
		"entities",
		"fact_card_junction",
		"facts",
		"files",
		"keywords",
		"llm_jobs",
		"llm_models",
		"llm_providers",
		"llm_query_log",
		"mailing_list",
		"revenue",
		"schema_definitions",
		"starred_cards",
		"starred_searches",
		"stripe_plans",
		"summarizations",
		"summary_arguments",
		"summary_sections",
		"summary_theses",
		"tags",
		"task_statuses",
		"task_tags",
		"tasks",
		"user_llm_configurations",
		"user_memories",
		"users",
	}

	// Build a single TRUNCATE statement for all tables to handle foreign keys correctly
	truncateStmt := "TRUNCATE TABLE "
	for i, table := range tables {
		if i > 0 {
			truncateStmt += ", "
		}
		truncateStmt += table
	}
	truncateStmt += " RESTART IDENTITY CASCADE"

	_, err := S.DB.Exec(truncateStmt)
	if err != nil {
		log.Printf("Warning: TRUNCATE failed (%v), falling back to DELETE for all tables", err)
		// Fallback: Use DELETE for all tables if TRUNCATE fails
		// Order matters for DELETE - delete from tables with no dependents first
		for _, table := range tables {
			_, deleteErr := S.DB.Exec("DELETE FROM " + table)
			if deleteErr != nil {
				log.Printf("Warning: failed to DELETE from %s: %v", table, deleteErr)
			}
		}
		// Explicitly reset sequences after DELETE
		for _, table := range tables {
			// Try to reset the sequence for this table if it has one
			_, seqErr := S.DB.Exec("SELECT setval(pg_get_serial_sequence('" + table + "', 'id'), 1, false)")
			if seqErr != nil {
				// Not all tables have sequences, so this is expected to fail for some
				continue
			}
		}
	} else {
		// TRUNCATE succeeded, but sequences might not be reset properly
		// Explicitly reset all sequences to ensure they start at 1
		for _, table := range tables {
			_, seqErr := S.DB.Exec("SELECT setval(pg_get_serial_sequence('" + table + "', 'id'), 1, false)")
			if seqErr != nil {
				// Not all tables have sequences, so this is expected to fail for some
				continue
			}
		}
	}
}

func ParseJsonResponse(t *testing.T, body []byte, x interface{}) {
	err := json.Unmarshal(body, &x)
	if err != nil {
		log.Printf("body: %v", string(body))
		t.Fatalf("could not unmarshal response: %v", err)
	}
}

func StringToReader(s string) *strings.Reader {
	return strings.NewReader(s)
}

func importTestData(s *server.Server) error {
	data := generateData()
	users := data["users"].([]models.User)
	cards := data["cards"].([]models.Card)
	files := data["files"].([]models.File)
	backlinks := data["backlinks"].([]models.Backlink)
	tasks := data["tasks"].([]models.Task)
	keywords := data["keywords"].([]models.Keyword)
	tags := data["tags"].([]models.Tag)
	card_tags := data["card_tags"].([]models.CardTag)

	tx, _ := s.DB.Begin()
	var userIDs []int
	for _, user := range users {
		var id int
		err := tx.QueryRow(`
			INSERT INTO users 
			(username, email, password, created_at, updated_at, can_upload_files, 
			stripe_subscription_status, stripe_customer_id, stripe_current_plan, stripe_subscription_frequency, stripe_subscription_id,
			email_validated, dashboard_card_pk) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 0) 
			RETURNING id`,
			user.Username, user.Email, user.Password, user.CreatedAt,
			user.UpdatedAt, user.CanUploadFiles, user.StripeSubscriptionStatus,
			user.StripeCustomerID, user.StripeCurrentPlan, user.StripeSubscriptionFrequency,
			user.StripeSubscriptionID, user.EmailValidated,
		).Scan(&id)
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
		userIDs = append(userIDs, id)
	}

	for _, card := range cards {
		_, err := tx.Exec(
			"INSERT INTO cards (card_id, user_id, title, body, link, created_at, updated_at, parent_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			card.CardID, card.UserID, card.Title, card.Body, card.Link, card.CreatedAt, card.UpdatedAt, card.ParentID,
		)
		if err != nil {
			log.Printf("something went wrong inserting rows: %v", err)
			return err
		}
	}

	for _, file := range files {
		_, err := tx.Exec(
			"INSERT INTO files (name, user_id, type, path, filename, size, created_by, updated_by, card_pk, is_deleted, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)",
			file.Name, file.UserID, file.Filetype, file.Path, file.Filename, file.Size, file.CreatedBy, file.UpdatedBy, file.CardPK, file.IsDeleted, file.CreatedAt, file.UpdatedAt,
		)
		if err != nil {
			log.Printf("error %v", err)
			return err
		}
	}

	_, err := tx.Exec("UPDATE users SET is_admin = TRUE WHERE id = 1")
	if err != nil {
		return err
	}

	for _, backlink := range backlinks {
		_, err := tx.Exec("INSERT INTO backlinks (source_id_int, target_id_int, created_at, updated_at) VALUES ($1, $2, $3, $4)", backlink.SourceIDInt, backlink.TargetIDInt, backlink.CreatedAt, backlink.UpdatedAt)
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
	}

	for _, task := range tasks {
		_, err := tx.Exec(
			"INSERT INTO tasks (card_pk, user_id, created_at, updated_at, due_date, scheduled_date, title, is_complete) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			task.CardPK,
			task.UserID,
			task.CreatedAt,
			task.UpdatedAt,
			task.DueDate,
			task.ScheduledDate,
			task.Title,
			task.IsComplete,
		)
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
	}

	for _, keyword := range keywords {
		_, err := tx.Exec(
			"INSERT INTO keywords (card_pk, user_id, keyword) VALUES ($1, $2, $3)",
			keyword.CardPK,
			keyword.UserID,
			keyword.Keyword,
		)
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
	}

	for _, tag := range tags {
		_, err := tx.Exec(
			"INSERT INTO tags (name, color, user_id) VALUES ($1, $2, $3)",
			tag.Name,
			tag.Color,
			tag.UserID,
		)
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
	}

	for _, card_tag := range card_tags {
		_, err := tx.Exec(
			"INSERT INTO card_tags (card_pk, tag_id) VALUES ($1, $2)",
			card_tag.CardPK,
			card_tag.TagID,
		)
		if err != nil {
			log.Printf("err %v", err)
			return err
		}
	}

	for _, entity := range data["entities"].([]models.Entity) {
		_, err := tx.Exec(`
			INSERT INTO entities 
			(id, user_id, name, description, type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			entity.ID, entity.UserID, entity.Name, entity.Description, entity.Type,
			entity.CreatedAt, entity.UpdatedAt,
		)
		if err != nil {
			log.Printf("error inserting entity: %v", err)
			return err
		}
	}

	_, err = tx.Exec(`SELECT setval('entities_id_seq', (SELECT MAX(id) FROM entities) + 1);`)
	if err != nil {
		log.Printf("error resetting entities sequence after inserts: %v", err)
		return err
	}

	for _, junction := range data["entity_cards"].([]models.EntityCardJunction) {
		_, err := tx.Exec(`
			INSERT INTO entity_card_junction 
			(user_id, entity_id, card_pk)
			VALUES ($1, $2, $3)`,
			junction.UserID, junction.EntityID, junction.CardPK,
		)
		if err != nil {
			log.Printf("error inserting entity_card_junction: %v", err)
			return err
		}
	}

	tx.Commit()
	return nil
}

func generateData() map[string]interface{} {
	rand.Seed(time.Now().UnixNano())

	keywords := []models.Keyword{}
	idCount := 0
	for i := 1; i <= 20; i++ {
		for x := 1; x < 10; x++ {
			idCount += 1
			keyword := models.Keyword{
				ID:      idCount,
				CardPK:  i,
				UserID:  1,
				Keyword: randomString(10),
			}
			keywords = append(keywords, keyword)
		}
	}

	tags := []models.Tag{}
	for i := 1; i <= 3; i++ {
		tag := models.Tag{
			Name:   randomString(10),
			Color:  randomString(10),
			UserID: 1,
		}
		if i == 1 {
			tag.Name = "test"
		}
		tags = append(tags, tag)

	}

	card_tags := []models.CardTag{}
	card_tag := models.CardTag{
		CardPK: 2,
		TagID:  1,
	}
	card_tags = append(card_tags, card_tag)

	users := []models.User{}
	for i := 1; i <= 10; i++ {
		user := models.User{
			ID:                          i,
			Username:                    randomString(10),
			Email:                       randomEmail(),
			Password:                    randomString(15),
			CreatedAt:                   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt:                   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			CanUploadFiles:              true,
			StripeSubscriptionStatus:    "",
			StripeCustomerID:            "",
			StripeCurrentPlan:           "",
			StripeSubscriptionFrequency: "",
			StripeSubscriptionID:        "",
			EmailValidated:              true,
		}
		if i == 2 {
			user.CanUploadFiles = false
			user.Email = "test@test.com"
		}
		users = append(users, user)
	}

	cards := []models.Card{}
	for i := 1; i <= 20; i++ {
		card := models.Card{
			ID:        i,
			CardID:    strconv.Itoa(i),
			UserID:    1,
			Title:     randomString(20),
			Body:      randomString(100),
			Link:      fmt.Sprintf("https://%s.com", randomString(10)),
			CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			ParentID:  i,
		}
		if i == 1 {
			card.Body = card.Body + "\n[" + strconv.Itoa(i+1) + "]"
		}
		if i == 4 {
			card.CardID = "REF001"
			card.Body = card.Body + "\n[3]"
		}
		if i == 5 {
			card.CardID = "MM001"
		}
		cards = append(cards, card)
	}
	cards = append(cards, models.Card{
		ID:        21,
		CardID:    "1/A",
		UserID:    1,
		Title:     randomString(20),
		Body:      randomString(20) + " uniquebodycontent",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  1,
	})
	cards = append(cards, models.Card{
		ID:        22,
		CardID:    "2/A",
		UserID:    1,
		Title:     "test card",
		Body:      randomString(20) + "[1]",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  2,
	})
	cards = append(cards, models.Card{
		ID:        23,
		CardID:    "1",
		UserID:    2,
		Title:     "test card",
		Body:      "hello world #to-read",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  23,
	})
	cards = append(cards, models.Card{
		ID:        24,
		CardID:    "2/A.1",
		UserID:    1,
		Title:     "another test card",
		Body:      randomString(20) + "[1]",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  22,
	})

	backlinks := []models.Backlink{}
	for i := 1; i <= 30; i++ {
		backlinks = append(backlinks, models.Backlink{
			SourceIDInt: i,
			TargetIDInt: i,
			CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		})
	}
	backlinks = append(backlinks, models.Backlink{
		SourceIDInt: 22,
		TargetIDInt: 1,
		CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
	})
	backlinks = append(backlinks, models.Backlink{
		SourceIDInt: 3,
		TargetIDInt: 4,
		CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
	})

	files := []models.File{}
	for i := 1; i <= 20; i++ {
		files = append(files, models.File{
			ID:        i,
			UserID:    1,
			Name:      randomString(20),
			Filetype:  randomString(20),
			Path:      randomString(20),
			Filename:  randomString(20),
			Size:      rand.Intn(1000),
			CreatedBy: rand.Intn(10) + 1,
			UpdatedBy: rand.Intn(10) + 1,
			CardPK:    i,
			IsDeleted: false,
			CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		})
	}

	tasks := []models.Task{}
	for i := 1; i <= 20; i++ {
		task := models.Task{
			ID:            i,
			CardPK:        i,
			UserID:        1,
			CreatedAt:     randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt:     randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			DueDate:       nil,
			ScheduledDate: randomMaybeNullDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			Title:         randomString(20),
			IsComplete:    false,
			CompletedAt:   nil,
		}
		if i == 2 {
			task.IsComplete = true
			task.CompletedAt = randomMaybeNullDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))
		}

		if i == 3 {
			task.Title = "hello world #to-read"
		}
		tasks = append(tasks, task)
	}

	entities := []models.Entity{
		{
			ID:          1,
			UserID:      1,
			Name:        "Test Entity 1",
			Description: "Original entity",
			Type:        "person",
			CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		{
			ID:          2,
			UserID:      1,
			Name:        "Test Entity 2",
			Description: "Duplicate entity",
			Type:        "person",
			CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		{
			ID:          3,
			UserID:      2,
			Name:        "Other User Entity",
			Description: "Entity for different user",
			Type:        "person",
			CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
	}

	entity_cards := []models.EntityCardJunction{
		{
			UserID:   1,
			EntityID: 1,
			CardPK:   1,
		},
		{
			UserID:   1,
			EntityID: 1,
			CardPK:   2,
		},
		{
			UserID:   1,
			EntityID: 2,
			CardPK:   1,
		},
		{
			UserID:   1,
			EntityID: 2,
			CardPK:   2,
		},
	}

	results := map[string]interface{}{
		"users":        users,
		"cards":        cards,
		"backlinks":    backlinks,
		"files":        files,
		"tasks":        tasks,
		"keywords":     keywords,
		"tags":         tags,
		"card_tags":    card_tags,
		"entities":     entities,
		"entity_cards": entity_cards,
	}
	return results
}

func randomString(length int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, length)
	for i := range s {
		s[i] = chars[rand.Intn(len(chars))]
	}
	return string(s)
}

func randomEmail() string {
	return fmt.Sprintf("%s@%s.com", randomString(5), randomString(5))
}

func randomDate(start, end time.Time) time.Time {
	delta := end.Unix() - start.Unix()
	sec := rand.Int63n(delta) + start.Unix()
	return time.Unix(sec, 0)
}
func randomMaybeNullDate(start, end time.Time) *time.Time {
	date := randomDate(start, end)
	return &date
}

func GenerateTestJWT(userID int) (string, error) {
	var jwtKey = []byte("")
	now := time.Now()

	claims := &models.Claims{
		Sub:   userID,
		Fresh: false, // Assuming 'fresh' is always false for test token
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func CreateJsonBody(t *testing.T, v interface{}) *bytes.Reader {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal to JSON: %v", err)
	}
	return bytes.NewReader(jsonBytes)
}
