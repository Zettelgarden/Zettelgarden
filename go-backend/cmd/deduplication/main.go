package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"go-backend/bootstrap"
	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/server"
)

// CLIApp holds the initialized services for deduplication operations
type CLIApp struct {
	Server *server.Server
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]

	// Handle help command immediately without requiring config
	switch subcommand {
	case "help", "-h", "--help":
		usage()
		return
	}

	// Initialize shared configuration and services for other commands
	cfg, err := loadConfigWithErrorHandling()
	if err != nil {
		log.Fatalf("Configuration error: %v\n\nMake sure all required environment variables are set.\nView available variables with: grep -r '^ZETTEL_' .env.example 2>/dev/null || echo 'Check your .env file'", err)
	}

	app, err := initAppWithErrorHandling(cfg)
	if err != nil {
		log.Fatalf("Application initialization error: %v", err)
	}
	defer app.Server.DB.Close()

	switch subcommand {
	case "entities":
		runEntityDeduplication(app, cfg, args)
	case "facts":
		runFactDeduplication(app, cfg, args)
	case "status":
		runStatusReport(app, cfg, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
		usage()
		os.Exit(1)
	}
}

// loadConfigWithErrorHandling wraps config.LoadConfig with better error messages
func loadConfigWithErrorHandling() (config.Config, error) {
	cfg := config.LoadConfig()
	// Config validation happens inside LoadConfig, so if we reach here it's valid
	return cfg, nil
}

// initAppWithErrorHandling initializes the server
func initAppWithErrorHandling(cfg config.Config) (*CLIApp, error) {
	s := bootstrap.InitServer(cfg.Database)
	if s == nil {
		return nil, fmt.Errorf("failed to initialize server")
	}

	return &CLIApp{
		Server: s,
	}, nil
}

func usage() {
	fmt.Println(`Zettelgarden Deduplication CLI

USAGE:
    deduplication <subcommand> [options]

SUBCOMMANDS:
    entities    Find and merge similar entities
    facts       Find and merge similar facts
    status      Show deduplication statistics and progress
    help        Show this help message

GLOBAL FLAGS:
    --dry-run                 Preview merges without executing (default: false)
    --batch-size int          Number of items to process per batch (default: 100)
    --similarity-threshold float  Similarity score cutoff 0.0-1.0 (default: 0.8)
    --time-window int         Process items newer than X days (default: 30)
    --user-id int             Process specific user (optional)
    --confirm                 Require user confirmation for each merge (default: false)

NOTE: Similarity search uses Typesense if available, otherwise falls back to text matching.

EXAMPLES:
    deduplication entities --dry-run --similarity-threshold 0.9
    deduplication facts --user-id 123 --batch-size 50
    deduplication status --user-id 123
`)
}

// Shared flag parsing for all subcommands
type DeduplicationFlags struct {
	DryRun              bool
	BatchSize           int
	SimilarityThreshold float64
	TimeWindowDays      int
	UserID              *int
	Confirm             bool
}

func parseFlags(args []string) (*DeduplicationFlags, []string) {
	fs := flag.NewFlagSet("deduplication", flag.ContinueOnError)

	flags := &DeduplicationFlags{}

	fs.BoolVar(&flags.DryRun, "dry-run", false, "Preview merges without executing")
	fs.IntVar(&flags.BatchSize, "batch-size", 100, "Number of items to process per batch")
	fs.Float64Var(&flags.SimilarityThreshold, "similarity-threshold", 0.8, "Similarity score cutoff 0.0-1.0")
	fs.IntVar(&flags.TimeWindowDays, "time-window", 30, "Process items newer than X days")
	fs.BoolVar(&flags.Confirm, "confirm", false, "Require confirmation for each merge")

	var userIDStr string
	fs.StringVar(&userIDStr, "user-id", "", "Process specific user ID")

	err := fs.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	// Parse user ID if provided
	if userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Fatalf("Invalid user ID: %s", userIDStr)
		}
		flags.UserID = &userID
	}

	return flags, fs.Args()
}

func runEntityDeduplication(app *CLIApp, cfg config.Config, args []string) {
	flags, remaining := parseFlags(args)
	if len(remaining) > 0 {
		log.Fatalf("Unexpected arguments for 'entities': %v", remaining)
	}

	fmt.Printf("Starting entity deduplication with flags:\n")
	fmt.Printf("  Dry run: %t\n", flags.DryRun)
	fmt.Printf("  Batch size: %d\n", flags.BatchSize)
	fmt.Printf("  Similarity threshold: %.2f\n", flags.SimilarityThreshold)
	fmt.Printf("  Time window: %d days\n", flags.TimeWindowDays)
	fmt.Printf("  Confirm merges: %t\n", flags.Confirm)
	if flags.UserID != nil {
		fmt.Printf("  User ID: %d\n", *flags.UserID)
	}

	err := deduplicateEntities(app, cfg, flags)
	if err != nil {
		log.Fatalf("Entity deduplication failed: %v", err)
	}

	log.Println("Entity deduplication completed successfully")
}

// deduplicateEntities performs entity deduplication for the given user(s)
func deduplicateEntities(app *CLIApp, cfg config.Config, flags *DeduplicationFlags) error {
	ctx := context.Background()

	// Get list of users to process
	var userIDs []int
	if flags.UserID != nil {
		userIDs = []int{*flags.UserID}
	} else {
		// Get all users with entities
		rows, err := app.Server.DB.Query("SELECT DISTINCT user_id FROM entities ORDER BY user_id")
		if err != nil {
			return fmt.Errorf("failed to get users: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var userID int
			if err := rows.Scan(&userID); err != nil {
				return fmt.Errorf("failed to scan user ID: %v", err)
			}
			userIDs = append(userIDs, userID)
		}
	}

	totalMerged := 0

	for _, userID := range userIDs {
		fmt.Printf("Processing entities for user %d...\n", userID)

		merged, err := deduplicateEntitiesForUser(app, cfg, flags, ctx, userID)
		if err != nil {
			log.Printf("Error processing user %d: %v", userID, err)
			continue
		}

		totalMerged += merged
	}

	fmt.Printf("Entity deduplication complete. Merged %d duplicates.\n", totalMerged)
	return nil
}

// deduplicateEntitiesForUser processes entities for a single user
func deduplicateEntitiesForUser(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, userID int) (int, error) {
	// Query entities for this user, ordered by creation date (newest first)
	timeCondition := ""
	if flags.TimeWindowDays > 0 {
		cutoffDate := time.Now().AddDate(0, 0, -flags.TimeWindowDays)
		timeCondition = fmt.Sprintf(" AND created_at > '%s'", cutoffDate.Format("2006-01-02"))
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, name, description, type, created_at, updated_at, card_pk
		FROM entities
		WHERE user_id = $1%s
		ORDER BY created_at DESC
	`, timeCondition)

	rows, err := app.Server.DB.Query(query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to query entities: %v", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var entity models.Entity
		var cardPK sql.NullInt64

		err := rows.Scan(
			&entity.ID, &entity.UserID, &entity.Name, &entity.Description, &entity.Type,
			&entity.CreatedAt, &entity.UpdatedAt, &cardPK,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to scan entity: %v", err)
		}
		if cardPK.Valid {
			cpk := int(cardPK.Int64)
			entity.CardPK = &cpk
		}

		entities = append(entities, entity)
	}

	if len(entities) == 0 {
		fmt.Printf("  No entities found for user %d\n", userID)
		return 0, nil
	}

	fmt.Printf("  Found %d entities for user %d\n", len(entities), userID)

	if flags.DryRun {
		fmt.Printf("  DRY RUN: Would process %d entities\n", len(entities))
		return 0, nil
	}

	// Find and merge duplicates
	merged := 0
	for _, entity := range entities {
		// Find similar entities
		similarIDs, err := app.Server.FindSimilarEntities(ctx, entity, 100)
		if err != nil {
			log.Printf("Error finding similar entities for %d: %v", entity.ID, err)
			continue
		}

		// For each similar entity with higher ID (newer than current), merge it into current
		for _, similarID := range similarIDs {
			if similarID > entity.ID { // Only merge newer entities into older ones
				if flags.Confirm {
					fmt.Printf("    Merge entity %d into %d? (y/N): ", similarID, entity.ID)
					var response string
					fmt.Scanln(&response)
					if response != "y" && response != "Y" {
						continue
					}
				}

				if err := app.Server.MergeEntities(ctx, userID, entity.ID, similarID); err != nil {
					log.Printf("Failed to merge entities %d -> %d: %v", similarID, entity.ID, err)
				} else {
					merged++
					fmt.Printf("    Merged entity %d into %d\n", similarID, entity.ID)
				}
			}
		}
	}

	return merged, nil
}

// deduplicateFacts performs fact deduplication for the given user(s)
func deduplicateFacts(app *CLIApp, cfg config.Config, flags *DeduplicationFlags) error {
	ctx := context.Background()

	// Get list of users to process
	var userIDs []int
	if flags.UserID != nil {
		userIDs = []int{*flags.UserID}
	} else {
		// Get all users with facts
		rows, err := app.Server.DB.Query("SELECT DISTINCT user_id FROM facts ORDER BY user_id")
		if err != nil {
			return fmt.Errorf("failed to get users: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var userID int
			if err := rows.Scan(&userID); err != nil {
				return fmt.Errorf("failed to scan user ID: %v", err)
			}
			userIDs = append(userIDs, userID)
		}
	}

	totalMerged := 0

	for _, userID := range userIDs {
		fmt.Printf("Processing facts for user %d...\n", userID)

		merged, err := deduplicateFactsForUser(app, cfg, flags, ctx, userID)
		if err != nil {
			log.Printf("Error processing user %d: %v", userID, err)
			continue
		}

		totalMerged += merged
	}

	fmt.Printf("Fact deduplication complete. Merged %d duplicates.\n", totalMerged)
	return nil
}

// deduplicateFactsForUser processes facts for a single user
func deduplicateFactsForUser(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, userID int) (int, error) {
	// Query facts for this user, ordered by creation date (newest first)
	timeCondition := ""
	if flags.TimeWindowDays > 0 {
		cutoffDate := time.Now().AddDate(0, 0, -flags.TimeWindowDays)
		timeCondition = fmt.Sprintf(" AND created_at > '%s'", cutoffDate.Format("2006-01-02"))
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, card_pk, fact, created_at, updated_at
		FROM facts
		WHERE user_id = $1%s
		ORDER BY created_at DESC
	`, timeCondition)

	rows, err := app.Server.DB.Query(query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to query facts: %v", err)
	}
	defer rows.Close()

	var facts []models.Fact
	for rows.Next() {
		var fact models.Fact

		err := rows.Scan(
			&fact.ID, &fact.UserID, &fact.CardPK, &fact.Fact,
			&fact.CreatedAt, &fact.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to scan fact: %v", err)
		}

		facts = append(facts, fact)
	}

	if len(facts) == 0 {
		fmt.Printf("  No facts found for user %d\n", userID)
		return 0, nil
	}

	fmt.Printf("  Found %d facts for user %d\n", len(facts), userID)

	if flags.DryRun {
		fmt.Printf("  DRY RUN: Would process %d facts\n", len(facts))
		return 0, nil
	}

	// Find and merge duplicates
	merged := 0
	for _, fact := range facts {
		// Find similar facts
		similarIDs, err := app.Server.FindSimilarFacts(ctx, fact, 100)
		if err != nil {
			log.Printf("Error finding similar facts for %d: %v", fact.ID, err)
			continue
		}

		// For each similar fact with higher ID (newer than current), merge it into current
		for _, similarID := range similarIDs {
			if similarID > fact.ID { // Only merge newer facts into older ones
				if flags.Confirm {
					fmt.Printf("    Merge fact %d into %d? (y/N): ", similarID, fact.ID)
					var response string
					fmt.Scanln(&response)
					if response != "y" && response != "Y" {
						continue
					}
				}

				if err := app.Server.MergeFacts(ctx, userID, fact.ID, similarID); err != nil {
					log.Printf("Failed to merge facts %d -> %d: %v", similarID, fact.ID, err)
				} else {
					merged++
					fmt.Printf("    Merged fact %d into %d\n", similarID, fact.ID)
				}
			}
		}
	}

	return merged, nil
}

func runFactDeduplication(app *CLIApp, cfg config.Config, args []string) {
	flags, remaining := parseFlags(args)
	if len(remaining) > 0 {
		log.Fatalf("Unexpected arguments for 'facts': %v", remaining)
	}

	fmt.Printf("Starting fact deduplication with flags:\n")
	fmt.Printf("  Dry run: %t\n", flags.DryRun)
	fmt.Printf("  Batch size: %d\n", flags.BatchSize)
	fmt.Printf("  Similarity threshold: %.2f\n", flags.SimilarityThreshold)
	fmt.Printf("  Time window: %d days\n", flags.TimeWindowDays)
	fmt.Printf("  Confirm merges: %t\n", flags.Confirm)
	if flags.UserID != nil {
		fmt.Printf("  User ID: %d\n", *flags.UserID)
	}

	err := deduplicateFacts(app, cfg, flags)
	if err != nil {
		log.Fatalf("Fact deduplication failed: %v", err)
	}

	log.Println("Fact deduplication completed successfully")
}

func runStatusReport(app *CLIApp, cfg config.Config, args []string) {
	flags, remaining := parseFlags(args)
	if len(remaining) > 0 {
		log.Fatalf("Unexpected arguments for 'status': %v", remaining)
	}

	var userCondition string
	var params []interface{}
	if flags.UserID != nil {
		fmt.Printf("Getting status for user ID: %d\n", *flags.UserID)
		userCondition = "WHERE user_id = $1"
		params = []interface{}{*flags.UserID}
	} else {
		fmt.Println("Getting global deduplication status")
		userCondition = ""
	}

	// Count entities
	entityQuery := fmt.Sprintf("SELECT COUNT(*) FROM entities %s", userCondition)
	var entityCount int
	err := app.Server.DB.QueryRow(entityQuery, params...).Scan(&entityCount)
	if err != nil {
		log.Fatalf("Failed to count entities: %v", err)
	}

	// Count facts
	factQuery := fmt.Sprintf("SELECT COUNT(*) FROM facts %s", userCondition)
	var factCount int
	err = app.Server.DB.QueryRow(factQuery, params...).Scan(&factCount)
	if err != nil {
		log.Fatalf("Failed to count facts: %v", err)
	}

	// Count junction table entries
	var entityCardJunctionCount, entityFactJunctionCount, factCardJunctionCount int
	junctionQuery1 := fmt.Sprintf("SELECT COUNT(*) FROM entity_card_junction %s", userCondition)
	err = app.Server.DB.QueryRow(junctionQuery1, params...).Scan(&entityCardJunctionCount)
	if err != nil {
		log.Fatalf("Failed to count entity_card_junction: %v", err)
	}

	junctionQuery2 := fmt.Sprintf("SELECT COUNT(*) FROM entity_fact_junction %s", userCondition)
	err = app.Server.DB.QueryRow(junctionQuery2, params...).Scan(&entityFactJunctionCount)
	if err != nil {
		log.Fatalf("Failed to count entity_fact_junction: %v", err)
	}

	junctionQuery3 := fmt.Sprintf("SELECT COUNT(*) FROM fact_card_junction %s", userCondition)
	err = app.Server.DB.QueryRow(junctionQuery3, params...).Scan(&factCardJunctionCount)
	if err != nil {
		log.Fatalf("Failed to count fact_card_junction: %v", err)
	}

	fmt.Printf("=== Deduplication Status ===\n")
	fmt.Printf("Entities: %d\n", entityCount)
	fmt.Printf("Facts: %d\n", factCount)
	fmt.Printf("Entity-Card Links: %d\n", entityCardJunctionCount)
	fmt.Printf("Entity-Fact Links: %d\n", entityFactJunctionCount)
	fmt.Printf("Fact-Card Links: %d\n", factCardJunctionCount)
}
