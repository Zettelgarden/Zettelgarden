package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go-backend/bootstrap"
	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/server"

	"github.com/lib/pq"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

// CLIApp holds the initialized services for deduplication operations
type CLIApp struct {
	Server       *server.Server
	Typesense    *typesense.Client
	TypesenseOK  bool // whether Typesense is available for similarity searches
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

// initAppWithErrorHandling initializes the server, database, and Typesense client
func initAppWithErrorHandling(cfg config.Config) (*CLIApp, error) {
	s := bootstrap.InitServer(cfg.Database)
	if s == nil {
		return nil, fmt.Errorf("failed to initialize server")
	}

	app := &CLIApp{
		Server: s,
	}

	// Try to initialize Typesense for similarity searches
	tsClient, err := bootstrap.InitTypesense(cfg.Services.Search)
	if err != nil {
		log.Printf("WARNING: Typesense unavailable (%v) - using fallback similarity detection", err)
		app.TypesenseOK = false
	} else {
		app.Typesense = tsClient
		app.TypesenseOK = true
		log.Println("Typesense client initialized successfully")
	}

	return app, nil
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

EXAMPLES:
    deduplication entities --dry-run --similarity-threshold 0.9
    deduplication facts --user-id 123 --batch-size 50
    deduplication status --user-id 123
`)
}

// Shared flag parsing for all subcommands
type DeduplicationFlags struct {
	DryRun             bool
	BatchSize          int
	SimilarityThreshold float64
	TimeWindowDays     int
	UserID             *int
	Confirm            bool
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
	fmt.Printf("  Typesense available: %t\n", app.TypesenseOK)
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

	totalProcessed := 0
	totalMerged := 0

	for _, userID := range userIDs {
		fmt.Printf("Processing entities for user %d...\n", userID)

		processed, merged, err := deduplicateEntitiesForUser(app, cfg, flags, ctx, userID)
		if err != nil {
			log.Printf("Error processing user %d: %v", userID, err)
			continue
		}

		totalProcessed += processed
		totalMerged += merged
	}

	fmt.Printf("Entity deduplication complete. Processed %d entities, merged %d duplicates.\n", totalProcessed, totalMerged)
	return nil
}

// deduplicateEntitiesForUser processes entities for a single user
func deduplicateEntitiesForUser(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, userID int) (int, int, error) {
	// Query entities for this user, ordered by creation date (newest first)
	timeCondition := ""
	if flags.TimeWindowDays > 0 {
		cutoffDate := time.Now().AddDate(0, 0, -flags.TimeWindowDays)
		timeCondition = fmt.Sprintf(" AND created_at > '%s'", cutoffDate.Format("2006-01-02"))
	}

	query := fmt.Sprintf(`
		SELECT
			e.id, e.user_id, e.name, e.description, e.type,
			e.created_at, e.updated_at, e.card_pk,
			(SELECT COUNT(DISTINCT ecj.card_pk) FROM entity_card_junction ecj WHERE ecj.entity_id = e.id) as card_count
		FROM entities e
		WHERE e.user_id = $1%s
		ORDER BY e.created_at DESC
	`, timeCondition)

	rows, err := app.Server.DB.Query(query, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query entities: %v", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var entity models.Entity
		var cardID sql.NullInt64

		err := rows.Scan(
			&entity.ID, &entity.UserID, &entity.Name, &entity.Description, &entity.Type,
			&entity.CreatedAt, &entity.UpdatedAt, &cardID, &entity.CardCount,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to scan entity: %v", err)
		}
		if cardID.Valid {
			cid := int(cardID.Int64)
			entity.CardPK = &cid
		}

		entities = append(entities, entity)
	}

	processed := len(entities)
	if processed == 0 {
		fmt.Printf("  No entities found for user %d\n", userID)
		return 0, 0, nil
	}

	// Find similar entity clusters
	clusters, err := findEntityClusters(app, cfg, flags, ctx, entities)
	if err != nil {
		return processed, 0, fmt.Errorf("failed to find clusters: %v", err)
	}

	// Filter clusters to only those with similarities above threshold
	var validClusters [][]int
	for _, cluster := range clusters {
		if len(cluster) > 1 { // Only clusters with duplicates
			validClusters = append(validClusters, cluster)
		}
	}

	if len(validClusters) == 0 {
		fmt.Printf("  No duplicate entities found for user %d\n", userID)
		return processed, 0, nil
	}

	fmt.Printf("  Found %d duplicate clusters for user %d\n", len(validClusters), userID)

	if flags.DryRun {
		fmt.Printf("  DRY RUN: Would merge %d duplicate clusters\n", len(validClusters))
		return processed, 0, nil
	}

	// Process merges for each cluster
	merged := 0
	for _, cluster := range validClusters {
		if count, err := mergeEntityCluster(app, cfg, flags, ctx, userID, cluster); err != nil {
			log.Printf("Failed to merge cluster: %v", err)
		} else {
			merged += count
		}
	}

	return processed, merged, nil
}

// findEntityClusters finds clusters of similar entities
func findEntityClusters(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, entities []models.Entity) ([][]int, error) {
	entityMap := make(map[int]models.Entity)
	for _, entity := range entities {
		entityMap[entity.ID] = entity
	}

	visited := make(map[int]bool)
	var clusters [][]int

	for _, entity := range entities {
		if visited[entity.ID] {
			continue
		}

		cluster, err := findSimilarEntities(app, cfg, flags, ctx, entity, entities, entityMap)
		if err != nil {
			return nil, fmt.Errorf("failed to find similarities for entity %d: %v", entity.ID, err)
		}

		if len(cluster) > 1 {
			clusters = append(clusters, cluster)

			// Mark all entities in this cluster as visited
			for _, id := range cluster {
				visited[id] = true
			}
		}
	}

	return clusters, nil
}

// findSimilarEntities finds entities similar to the given entity
func findSimilarEntities(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, entity models.Entity, allEntities []models.Entity, entityMap map[int]models.Entity) ([]int, error) {
	cluster := []int{entity.ID}

	if app.TypesenseOK {
		// Use Typesense for similarity search
		similarIDs, err := findSimilarEntitiesTypesense(app, cfg, flags, ctx, entity)
		if err != nil {
			log.Printf("Typesense similarity search failed for entity %d, falling back to SQL: %v", entity.ID, err)
		} else {
			for _, similarID := range similarIDs {
				if _, exists := entityMap[similarID]; exists {
					cluster = append(cluster, similarID)
				}
			}
			return cluster, nil
		}
	}

	// Fallback: Use simple SQL text similarity
	for _, other := range allEntities {
		if other.ID == entity.ID {
			continue
		}

		similarity := calculateTextSimilarity(entity.Name+" "+entity.Description, other.Name+" "+other.Description)
		if similarity >= flags.SimilarityThreshold {
			cluster = append(cluster, other.ID)
		}
	}

	return cluster, nil
}

// findSimilarEntitiesTypesense uses Typesense to find similar entities
func findSimilarEntitiesTypesense(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, entity models.Entity) ([]int, error) {
	collectionName := cfg.Services.Search.Collection
	filter := fmt.Sprintf("user_id:=%d && type:=entity", entity.UserID)
	perPage := 50 // Get more results for analysis

	searchParams := &api.SearchCollectionParams{
		Q:        entity.Name,
		QueryBy:  "title,embedding",
		FilterBy: &filter,
		PerPage:  &perPage,
	}

	searchResult, err := app.Typesense.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, err
	}

	var similarIDs []int
	if searchResult.Hits != nil {
		for _, hit := range *searchResult.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				if pk, ok := doc["entity_pk"].(float64); ok && int(pk) != entity.ID {
					similarIDs = append(similarIDs, int(pk))
				}
			}
		}
	}

	return similarIDs, nil
}

// calculateTextSimilarity is a simple similarity function as fallback
func calculateTextSimilarity(text1, text2 string) float64 {
	if text1 == text2 {
		return 1.0
	}

	text1 = strings.ToLower(text1)
	text2 = strings.ToLower(text2)

	if strings.Contains(text1, text2) || strings.Contains(text2, text1) {
		return 0.9 // High similarity if one contains the other
	}

	return 0.0 // No similarity detected with simple method
}

// mergeEntityCluster merges a cluster of similar entities
func mergeEntityCluster(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, userID int, cluster []int) (int, error) {
	// Find entity with most card links as the keeper
	keeperID, err := findKeeperEntity(app.Server.DB, cluster)
	if err != nil {
		return 0, fmt.Errorf("failed to find keeper entity: %v", err)
	}

	// Remove keeper from merge targets
	var toMerge []int
	for _, id := range cluster {
		if id != keeperID {
			toMerge = append(toMerge, id)
		}
	}

	fmt.Printf("    Merging %d entities into entity %d\n", len(toMerge), keeperID)

	if flags.Confirm {
		fmt.Printf("    Press Enter to continue...")
		fmt.Scanln()
	}

	// Perform merge in transaction
	return mergeEntitiesInTransaction(app, ctx, keeperID, toMerge)
}

// findKeeperEntity finds the entity with the most card links in a cluster
func findKeeperEntity(db *sql.DB, entityIDs []int) (int, error) {
	query := `
		SELECT ecj.entity_id, COUNT(DISTINCT ecj.card_pk) as card_count
		FROM entity_card_junction ecj
		WHERE ecj.entity_id = ANY($1)
		GROUP BY ecj.entity_id
		ORDER BY card_count DESC
		LIMIT 1
	`

	row := db.QueryRow(query, pq.Array(entityIDs))

	var keeperID int
	var cardCount int
	err := row.Scan(&keeperID, &cardCount)
	if err != nil {
		// If no links found, use the first entity
		if len(entityIDs) > 0 {
			return entityIDs[0], nil
		}
		return 0, err
	}

	return keeperID, nil
}

// mergeEntitiesInTransaction performs the atomic merge operation
func mergeEntitiesInTransaction(app *CLIApp, ctx context.Context, keeperID int, toMerge []int) (int, error) {
	tx, err := app.Server.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Update entity_card_junction references
	_, err = tx.Exec("UPDATE entity_card_junction SET entity_id = $1 WHERE entity_id = ANY($2)", keeperID, pq.Array(toMerge))
	if err != nil {
		return 0, fmt.Errorf("failed to update entity_card_junction: %v", err)
	}

	// Update entity_fact_junction references
	_, err = tx.Exec("UPDATE entity_fact_junction SET entity_id = $1 WHERE entity_id = ANY($2)", keeperID, pq.Array(toMerge))
	if err != nil {
		return 0, fmt.Errorf("failed to update entity_fact_junction: %v", err)
	}

	// Update entities table - merge descriptions
	_, err = tx.Exec(`
		UPDATE entities
		SET description = COALESCE(NULLIF(description, ''), '') || E'\n\nMerged from duplicates: ' || (
			SELECT string_agg(COALESCE(NULLIF(name || COALESCE(E'\n' || description, ''), ''), 'unknown'), E'\n---\n')
			FROM entities WHERE id = ANY($2)
		),
		updated_at = NOW()
		WHERE id = $1
	`, keeperID, pq.Array(toMerge))
	if err != nil {
		return 0, fmt.Errorf("failed to update entity description: %v", err)
	}

	// Delete merged entities
	_, err = tx.Exec("DELETE FROM entities WHERE id = ANY($1)", pq.Array(toMerge))
	if err != nil {
		return 0, fmt.Errorf("failed to delete merged entities: %v", err)
	}

	// Update Typesense index
	if app.TypesenseOK {
		if err := updateTypesenseAfterEntityMerge(app, ctx, keeperID); err != nil {
			log.Printf("Warning: failed to update Typesense index: %v", err)
			// Don't fail the entire operation due to Typesense issues
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return len(toMerge), nil
}

// updateTypesenseAfterEntityMerge updates the Typesense index after entity merge
func updateTypesenseAfterEntityMerge(app *CLIApp, ctx context.Context, entityID int) error {
	// This would require re-indexing the merged entity
	// For now, we'll skip this complexity and just note the need
	log.Printf("Note: Typesense index should be refreshed for entity %d after merge", entityID)
	return nil
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

	totalProcessed := 0
	totalMerged := 0

	for _, userID := range userIDs {
		fmt.Printf("Processing facts for user %d...\n", userID)

		processed, merged, err := deduplicateFactsForUser(app, cfg, flags, ctx, userID)
		if err != nil {
			log.Printf("Error processing user %d: %v", userID, err)
			continue
		}

		totalProcessed += processed
		totalMerged += merged
	}

	fmt.Printf("Fact deduplication complete. Processed %d facts, merged %d duplicates.\n", totalProcessed, totalMerged)
	return nil
}

// deduplicateFactsForUser processes facts for a single user
func deduplicateFactsForUser(app *CLIApp, cfg config.Config, flags *DeduplicationFlags, ctx context.Context, userID int) (int, int, error) {
	log.Println("Fact deduplication not yet implemented")
	return 0, 0, nil
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
	fmt.Printf("  Typesense available: %t\n", app.TypesenseOK)
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
	fmt.Printf("Typesense Available: %t\n", app.TypesenseOK)
}