package server

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"io/ioutil"
	"log"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

func ConnectToDatabase(dbConfig models.DatabaseConfig) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%v port=%v user=%v "+
		"password=%v dbname=%v sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DatabaseName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Configure connection pool for better performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, err
}

func ResetDatabase(S *Server) error {
	_, err := S.DB.Exec(`
			DROP TABLE IF EXISTS users CASCADE;
			DROP TABLE IF EXISTS cards CASCADE;
			DROP TABLE IF EXISTS backlinks CASCADE;
			DROP TABLE IF EXISTS card_views CASCADE;
			DROP TABLE IF EXISTS files CASCADE;
			DROP TABLE IF EXISTS migrations CASCADE;
			DROP TABLE IF EXISTS stripe_plans CASCADE;
			DROP TABLE IF EXISTS tasks CASCADE;
            DROP TABLE IF EXISTS keywords CASCADE;
			DROP TABLE IF EXISTS card_tags CASCADE;
			DROP TABLE IF EXISTS tags CASCADE;
			DROP TABLE IF EXISTS task_tags CASCADE;
			DROP TABLE IF EXISTS card_embeddings CASCADE;
			DROP TABLE IF EXISTS card_chunks CASCADE;
			DROP TABLE IF EXISTS mailing_list CASCADE;
			DROP TABLE IF EXISTS chat_completions CASCADE;
			DROP TABLE IF EXISTS chat_conversations CASCADE;
			DROP TABLE IF EXISTS entities CASCADE;
			DROP TABLE IF EXISTS entity_card_junction CASCADE;
			DROP TABLE IF EXISTS audit_events CASCADE;
			DROP TABLE IF EXISTS llm_providers CASCADE;
			DROP TABLE IF EXISTS llm_models CASCADE;
			DROP TABLE IF EXISTS user_llm_configurations CASCADE;
			DROP TABLE IF EXISTS pinned_cards CASCADE;
			DROP TABLE IF EXISTS card_templates CASCADE;
			DROP TABLE IF EXISTS starred_searches CASCADE;
			DROP TABLE IF EXISTS user_memories CASCADE;
			DROP TABLE IF EXISTS summarizations CASCADE;
			DROP TABLE IF EXISTS facts CASCADE;
			DROP TABLE IF EXISTS entity_fact_junction CASCADE;
			DROP TABLE IF EXISTS fact_card_junction CASCADE;
			DROP TABLE IF EXISTS llm_query_log CASCADE;
			DROP TABLE IF EXISTS revenue CASCADE;
			DROP TABLE IF EXISTS summary_sections CASCADE;
			DROP TABLE IF EXISTS summary_theses CASCADE;
			DROP TABLE IF EXISTS summary_arguments CASCADE;
			DROP TABLE IF EXISTS chat_messages CASCADE;
			DROP TABLE IF EXISTS chat_tool_calls CASCADE;
			DROP TABLE IF EXISTS chat_usage_quotas CASCADE;
			DROP TABLE IF EXISTS starred_cards CASCADE;
			DROP TABLE IF EXISTS pinned_searches CASCADE;
			DROP TABLE IF EXISTS task_statuses CASCADE;
			DROP TABLE IF EXISTS api_keys CASCADE;
			DROP TABLE IF EXISTS admin_audit_log CASCADE;
			DROP TABLE IF EXISTS llm_jobs CASCADE;
			DROP TABLE IF EXISTS user_stats CASCADE;
			DROP TABLE IF EXISTS schema_definitions CASCADE;
			DROP TABLE IF EXISTS external_calendars CASCADE;
			DROP TABLE IF EXISTS external_events CASCADE;
			DROP TABLE IF EXISTS scheduled_job_runs CASCADE;

			DROP INDEX IF EXISTS idx_task_statuses_user;
			DROP INDEX IF EXISTS idx_task_statuses_position;
			DROP INDEX IF EXISTS idx_task_dependencies_task_id;
			DROP INDEX IF EXISTS idx_task_dependencies_blocking_task_id;
			DROP INDEX IF EXISTS idx_api_keys_user_id;
			DROP INDEX IF EXISTS idx_unique_active_key_name_per_user;
			DROP INDEX IF EXISTS idx_schema_definitions_owner_name;
			DROP INDEX IF EXISTS idx_users_last_memory_job_id;
			DROP INDEX IF EXISTS idx_llm_jobs_updated_at;
			DROP INDEX IF EXISTS idx_summarizations_llm_job_id;
			DROP INDEX IF EXISTS idx_user_stats_card_count;
			DROP INDEX IF EXISTS idx_user_stats_revenue;
			DROP INDEX IF EXISTS idx_summarizations_user_card_created;
			DROP INDEX IF EXISTS idx_summarizations_user_created;
			DROP INDEX IF EXISTS idx_admin_audit_log_admin_user_id;
			DROP INDEX IF EXISTS idx_admin_audit_log_action;
			DROP INDEX IF EXISTS idx_admin_audit_log_target;
			DROP INDEX IF EXISTS idx_admin_audit_log_created_at;
			DROP INDEX IF EXISTS idx_llm_jobs_user_status;
			DROP INDEX IF EXISTS idx_llm_jobs_created_at;
			DROP INDEX IF EXISTS idx_llm_jobs_priority;
			DROP INDEX IF EXISTS idx_llm_jobs_status;
			DROP INDEX IF EXISTS idx_schema_definitions_owner_name;
			DROP INDEX IF EXISTS idx_schema_definitions_owner_slug;
			DROP INDEX IF EXISTS idx_schema_definitions_owner_id;
			DROP INDEX IF EXISTS idx_cards_card_schema_id;
			DROP INDEX IF EXISTS idx_tasks_status;
			DROP INDEX IF EXISTS idx_cards_user_created;
			DROP INDEX IF EXISTS idx_tasks_user_created;
			DROP INDEX IF EXISTS idx_tasks_user_completed;

			CREATE TABLE IF NOT EXISTS migrations (
				id SERIAL PRIMARY KEY,
				migration_name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
	if err != nil {
		return err
	}

	return nil
}

func RunMigrations(S *Server) {
	queryString := "SELECT applied_at FROM migrations WHERE migration_name = $1"
	insertString := "INSERT INTO migrations (migration_name) VALUES ($1)"

	files, err := ioutil.ReadDir(S.SchemaDir)
	if err != nil {
		log.Fatal(err)
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		var result time.Time
		err = S.DB.QueryRow(queryString, fileName).Scan(&result)

		if err == sql.ErrNoRows {
			content, err := ioutil.ReadFile(S.SchemaDir + "/" + fileName)
			if err != nil {
				log.Fatal(err)
			}

			tx, err := S.DB.Begin()
			if err != nil {
				log.Fatal(err)
			}

			_, err = tx.Exec(string(content))
			if err != nil {
				tx.Rollback()
				log.Fatal(err)
			}

			_, err = tx.Exec(insertString, fileName)
			if err != nil {
				tx.Rollback()
				log.Fatal(err)
			}

			err = tx.Commit()
			if err != nil {
				log.Fatal(err)
			}

			//	fmt.Println("Running migration:", fileName)
		} else if err != nil {
			log.Fatal(err)
		}
	}
}
