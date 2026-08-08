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
	"go-backend/services/storage"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
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

// Testing Transaction Pattern:
//
// This test framework uses a transaction-per-test pattern for test isolation:
//
// 1. Setup() creates a new transaction for each test
// 2. All database operations use Handler.GetDB() which returns the test transaction
//    during testing (when Server.Testing=true and Server.Tx is set)
// 3. Teardown() rolls back the transaction, discarding all changes
//
// This ensures:
// - Each test starts with a clean database state
// - Tests don't interfere with each other
// - Tests run quickly (no need to drop/recreate tables)
// - The production database is never affected by tests
//
// The Handler.GetDB() method is the key to this pattern:
//   - During testing: returns Server.Tx (the test transaction)
//   - During production: returns h.DB (the actual database connection)
//
// Test Setup Patterns
//
// There are two main test setup patterns:
//
// 1. Handler Tests (HTTP endpoint testing):
//    Use handlers.NewHandler() to set up a Handler with all dependencies.
//    This is the standard way to test HTTP routes and middleware.
//
//    Example:
//      func TestGetCard(t *testing.T) {
//          s := handlers.NewHandler()
//          defer tests.Teardown()
//
//          token, _ := tests.GenerateTestJWT(1)
//          req, _ := http.NewRequest("GET", "/api/cards/1", nil)
//          req.Header.Set("Authorization", "Bearer "+token)
//
//          rr := httptest.NewRecorder()
//          router := mux.NewRouter()
//          router.HandleFunc("/api/cards/{id}", s.JwtMiddleware(s.GetCardRoute))
//          router.ServeHTTP(rr, req)
//
//          if status := rr.Code; status != http.StatusOK {
//              t.Errorf("expected status OK, got %v", status)
//          }
//      }
//
// 2. Service Tests (business logic testing):
//    Use tests.Setup() directly to get the server, then access services.
//    Use this when testing service layer functions without HTTP.
//
//    Example:
//      func TestCreateCard(t *testing.T) {
//          s := tests.Setup()
//          defer tests.Teardown()
//
//          userID := 1
//          params := models.EditCardParams{
//              Title:  "Test Card",
//              Body:   "Test Body",
//              CardID: "test123",
//          }
//
//          card, err := services.CreateCard(s.DB, userID, params)
//          if err != nil {
//              t.Fatalf("CreateCard failed: %v", err)
//          }
//
//          if card.Title != params.Title {
//              t.Errorf("expected title %v, got %v", params.Title, card.Title)
//          }
//      }
//
// Important Notes:
// - Always call tests.Teardown() in a defer statement immediately after Setup/NewHandler
// - Handler.GetDB() returns the test transaction during testing
// - Tests run sequentially (parallel=1) to avoid database conflicts
// - Test fixtures are loaded once via sync.Once in tests.Setup()

// setTestEnvironmentVariables sets required environment variables for testing
func setTestEnvironmentVariables() {
	// The backend is SQLite-only (Postgres retired, epic Zettelgarden-c7j
	// Phase 7b). SQLITE_PATH is intentionally NOT defaulted here: tests use a
	// fresh file-backed temp DB created in Setup so that WAL mode engages --
	// see the comment where it is opened.

	// Server config - test defaults
	setEnvIfNotSet("ZETTEL_DEV", "true")
	setEnvIfNotSet("ZETTEL_PORT", "8080")
	setEnvIfNotSet("ZETTEL_URL", "http://localhost:8080")
	setEnvIfNotSet("ZETTEL_ADMIN_EMAIL", "admin@test.com")
	setEnvIfNotSet("SECRET_KEY", "test-secret-key-for-jwt-signing-32-chars-minimum")
	setEnvIfNotSet("ZETTEL_BACKEND_LOG_LOCATION", "")

	// LLM service config - test defaults
	setEnvIfNotSet("ZETTEL_LLM_KEY", "test-zai-api-key")
	setEnvIfNotSet("ZETTEL_LLM_ENDPOINT", "https://api.z.ai/api/coding/paas/v4")
	setEnvIfNotSet("ZETTEL_LLM_DEFAULT_MODEL", "glm-5.1")
	setEnvIfNotSet("ZETTEL_LLM_SUMMARIZE_MODEL", "glm-5.1")

	// Mail service config - test defaults (direct SMTP, 6er.12)
	setEnvIfNotSet("SMTP_HOST", "smtp.gmail.com")
	setEnvIfNotSet("SMTP_PORT", "587")
	setEnvIfNotSet("SMTP_USERNAME", "test-smtp-user")
	setEnvIfNotSet("SMTP_PASSWORD", "test-mail-password")
	setEnvIfNotSet("SMTP_FROM", "noreply@test.com")

	// Stripe config - test defaults
	setEnvIfNotSet("STRIPE_SECRET_KEY", "test-stripe-secret-key")
	setEnvIfNotSet("STRIPE_PUBLISHABLE_KEY", "test-stripe-publishable-key")
	setEnvIfNotSet("STRIPE_WEBHOOK_SECRET", "test-stripe-webhook-secret")
	setEnvIfNotSet("STRIPE_MONTH_PRICE", "price_monthly_test_id")
	setEnvIfNotSet("STRIPE_YEAR_PRICE", "price_yearly_test_id")
	setEnvIfNotSet("STRIPE_BILLING_URL", "https://billing.stripe.com/test")

	// File storage config - test default. The suite shares one store dir per
	// process (see Setup); keys are server-generated UUIDs so tests don't
	// collide. The real LocalStore is wired onto S.Store in Setup (design D8).
	setEnvIfNotSet("STORAGE_DIR", filepath.Join(os.TempDir(), "zettelgarden-test-storage"))

	// GitHub OAuth config - test defaults
	setEnvIfNotSet("GITHUB_AUTH_ENABLED", "true")
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

// Setup initializes the test environment for a single test.
//
// It runs global setup (database connection, migrations, test data) exactly once
// using sync.Once, then creates a fresh transaction for each test.
//
// The transaction is stored in Server.Tx, which causes Handler.GetDB() to
// return the transaction instead of the database connection.
//
// Call this function at the start of each test, followed by defer Teardown().
func Setup() *server.Server {
	var err error
	setupOnce.Do(func() {
		// Set test environment variables if not already set
		setTestEnvironmentVariables()

		// Load config. The backend is SQLite-only (Postgres retired, epic
		// Zettelgarden-c7j Phase 7b); the suite runs against a file-backed
		// SQLite DB built from the consolidated schema, with no Postgres
		// process required.
		cfg := config.LoadConfig()

		// Tests use a FILE-backed SQLite DB, not :memory:. WAL mode (the D4
		// production setting) is ignored for in-memory databases, which run in
		// rollback-journal mode instead -- and there, an open transaction
		// blocks writers on other connections. Many handlers write via the
		// pool directly (s.DB) while the per-test transaction (S.Tx) is open,
		// which deadlocks the busy-handler under rollback-journal. A
		// file-backed DB lets WAL engage, so the pool and the test tx coexist
		// (same isolation semantics: a read sees prior committed writes).
		tmpDir, err := os.MkdirTemp("", "zettelgarden_test_*")
		if err != nil {
			log.Fatalf("test sqlite temp dir: %v", err)
		}
		db, err = server.OpenSQLite(filepath.Join(tmpDir, "test.db"))
		if err != nil {
			log.Fatalf("Unable to connect to the database: %v\n", err)
		}

		S = &server.Server{}
		S.DB = db
		S.Testing = true
		// Use absolute path for schema directory to work from any test directory.
		schemaDir, err := getSchemaDir()
		if err != nil {
			log.Fatalf("Failed to get schema directory: %v\n", err)
		}
		// SQLite loads its consolidated schema from schema/sqlite/.
		// RunMigrations then applies schema.sqlite.sql via the Phase 1
		// statement splitter.
		schemaDir = filepath.Join(schemaDir, "sqlite")
		S.SchemaDir = schemaDir

		S.Mail = &mail.MailClient{
			Testing:           true,
			TestingEmailsSent: 0,
			DB:                db,
		}
		// Real tempdir-backed local store (design D8): upload/download routes
		// now stream real bytes in tests instead of the old Server.Testing
		// no-op. loadStorageConfig already MkdirAll'd the dir; NewLocalStore
		// re-validates it and returns a usable Store rooted there.
		fileStore, err := storage.NewLocalStore(cfg.Services.Storage.Dir)
		if err != nil {
			log.Fatalf("test storage init: %v", err)
		}
		S.Store = fileStore
		S.LLMClient = &models.LLMClient{Testing: true}

		server.RunMigrations(S)
		err = importTestData(S)
		if err != nil {
			log.Fatal(err)
		}
	})

	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}

	// Create a fresh transaction for this test
	tx, err := S.DB.Begin()
	if err != nil {
		log.Fatalf("Unable to start transaction: %v\n", err)
	}

	S.Tx = tx
	S.Mail.Tx = S.Tx
	log.Printf("S %v", S)
	return S
}

// Teardown rolls back the test transaction, discarding all changes made during the test.
//
// This ensures that each test starts with a clean database state and that
// tests don't interfere with each other.
func Teardown() {
	// Rollback the transaction if it exists
	if S.Tx != nil {
		S.Tx.Rollback()
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
		"email_accounts",
		"email_card_links",
		"emails",
		"email_triage_decisions",
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
		"notifications",
		"notification_preferences",
		"revenue",
		"schema_definitions",
		"scheduled_job_runs",
		"starred_cards",
		"starred_searches",
		"stripe_plans",
		"summarizations",
		"tags",
		"task_statuses",
		"task_tags",
		"tasks",
		"user_llm_configurations",
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

	tx, err := s.DB.Begin()
	if err != nil {
		log.Fatalf("importTestData: failed to begin transaction: %v", err)
	}
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
			"INSERT INTO cards (card_id, user_id, title, body, link, created_at, updated_at, parent_id, sync_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, lower(hex(randomblob(16))))",
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

	_, err = tx.Exec("UPDATE users SET is_admin = TRUE WHERE id = 1")
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
			"INSERT INTO tasks (card_pk, user_id, created_at, updated_at, due_date, scheduled_date, title, is_complete, sync_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, lower(hex(randomblob(16))))",
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
			"INSERT INTO tags (name, color, user_id, sync_uuid) VALUES ($1, $2, $3, lower(hex(randomblob(16))))",
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

	// The entities id sequence does not need resetting on SQLite: explicit-id
	// inserts into an INTEGER PRIMARY KEY AUTOINCREMENT column auto-advance
	// sqlite_sequence / the rowid max. (This was a Postgres setval; that path
	// is gone with the Postgres retirement.)

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

	if err = tx.Commit(); err != nil {
		log.Printf("importTestData: failed to commit transaction: %v", err)
		return err
	}
	return nil
}

func generateData() map[string]interface{} {
	rand.Seed(time.Now().UnixNano())

	// Reduced keywords: 3 cards × 6 keywords = 18 (was 180)
	keywords := []models.Keyword{}
	idCount := 0
	for i := 1; i <= 3; i++ {
		for x := 1; x < 6; x++ {
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

	// Tags: keep 3 (needed for tag tests)
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

	// Card_tags: keep 1 (needed for tag tests)
	card_tags := []models.CardTag{}
	card_tag := models.CardTag{
		CardPK: 2,
		TagID:  1,
	}
	card_tags = append(card_tags, card_tag)

	// Users: reduced to 3 (was 10) - keep admin user, test@test.com user, and one more
	users := []models.User{}
	for i := 1; i <= 3; i++ {
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
		if i == 1 {
			// User 1 has unvalidated email for TestValidateEmail
			user.EmailValidated = false
		}
		if i == 2 {
			user.CanUploadFiles = false
			user.Email = "test@test.com"
		}
		users = append(users, user)
	}

	// Cards: reduced to 10 basic + 4 special = 14 (was 27)
	cards := []models.Card{}
	for i := 1; i <= 10; i++ {
		card := models.Card{
			ID:        i,
			CardID:    strconv.Itoa(i),
			UserID:    1,
			Title:     randomString(20),
			Body:      randomString(100),
			Link:      fmt.Sprintf("https://%s.com", randomString(10)),
			CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			ParentID:  &[]int{i}[0],
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
	// Special cards (kept for specific tests)
	cards = append(cards, models.Card{
		ID:        11,
		CardID:    "1/A",
		UserID:    1,
		Title:     randomString(20),
		Body:      randomString(20) + " uniquebodycontent",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  &[]int{1}[0],
	})
	cards = append(cards, models.Card{
		ID:        12,
		CardID:    "2/A",
		UserID:    1,
		Title:     "test card",
		Body:      randomString(20) + "[1]",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  &[]int{2}[0],
	})
	cards = append(cards, models.Card{
		ID:        13,
		CardID:    "1",
		UserID:    2,
		Title:     "test card",
		Body:      "hello world #to-read",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  &[]int{13}[0],
	})
	cards = append(cards, models.Card{
		ID:        14,
		CardID:    "2/A.1",
		UserID:    1,
		Title:     "another test card",
		Body:      randomString(20) + "[1]",
		Link:      fmt.Sprintf("https://%s.com", randomString(10)),
		CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		ParentID:  &[]int{12}[0],
	})

	// Backlinks: reduced to 5 (was 32)
	backlinks := []models.Backlink{}
	backlinks = append(backlinks, models.Backlink{
		SourceIDInt: 1,
		TargetIDInt: 1,
		CreatedAt:   randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdatedAt:   randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
	})
	backlinks = append(backlinks, models.Backlink{
		SourceIDInt: 12,
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

	// Files: reduced to 5 (was 20)
	files := []models.File{}
	for i := 1; i <= 5; i++ {
		files = append(files, models.File{
			ID:        i,
			UserID:    1,
			Name:      randomString(20),
			Filetype:  randomString(20),
			Path:      randomString(20),
			Filename:  randomString(20),
			Size:      rand.Intn(1000),
			CreatedBy: rand.Intn(3) + 1,
			UpdatedBy: rand.Intn(3) + 1,
			CardPK:    getIntPtr(i),
			IsDeleted: false,
			CreatedAt: randomDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			UpdatedAt: randomDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		})
	}

	// Tasks: reduced to 5 (was 20)
	tasks := []models.Task{}
	for i := 1; i <= 5; i++ {
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
		{
			ID:          4,
			UserID:      1,
			Name:        "Test Entity 4",
			Description: "Fourth entity for related card testing",
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
		// Entity 4 is shared between card 1 and card 3
		// Card 3 is not a reference of card 1, so it should appear in related cards
		{
			UserID:   1,
			EntityID: 4,
			CardPK:   1,
		},
		{
			UserID:   1,
			EntityID: 4,
			CardPK:   3,
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

// getSchemaDir returns the absolute path to the schema directory
func getSchemaDir() (string, error) {
	// Get the directory containing this file (tests/)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not get caller info")
	}
	// Get the tests directory
	testsDir := filepath.Dir(filename)
	// Go up one level to get go-backend directory
	backendDir := filepath.Dir(testsDir)
	// Return the schema directory path
	return filepath.Join(backendDir, "schema"), nil
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

func getIntPtr(i int) *int {
	return &i
}
