package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
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
	s.TypesenseClient = bootstrap.GetTypesenseClient(cfg.Services.Search)
	if s == nil {
		return nil, fmt.Errorf("failed to initialize server")
	}

	return &CLIApp{
		Server: s,
	}, nil
}

func usage() {
	fmt.Print(`Zettelgarden Deduplication CLI

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

// deduplicateEntitiesForUser processes entities for a single user using cluster-based merging
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

	// Build similarity clusters using union-find
	// Only create links between entities above the similarity threshold
	type clusterInfo struct {
		rootID       int
		memberIDs    []int
		repEntity    *models.Entity
		similarities map[int]float64 // score for each member to root
	}

	clusters := make(map[int]*clusterInfo)    // root ID -> cluster info
	entityMap := make(map[int]*models.Entity) // ID -> entity

	// Build entity map for quick lookup
	for i := range entities {
		entityMap[entities[i].ID] = &entities[i]
		clusters[entities[i].ID] = &clusterInfo{
			rootID:       entities[i].ID,
			memberIDs:    []int{entities[i].ID},
			repEntity:    &entities[i],
			similarities: make(map[int]float64),
		}
	}

	// Build clusters by finding similar entities and linking them
	// We only create links between entities that are above threshold
	for _, entity := range entities {
		similarEntities, err := app.Server.FindSimilarEntities(ctx, entity, 100)
		if err != nil {
			log.Printf("Error finding similar entities for %d: %v", entity.ID, err)
			continue
		}

		for _, similarEntity := range similarEntities {
			// Only link if above threshold and similarEntity exists in our map
			if similarEntity.Score < flags.SimilarityThreshold {
				continue
			}
			if _, exists := entityMap[similarEntity.ID]; !exists {
				continue
			}

			// Find roots of both entities (with nil checks in case cluster was deleted)
			cluster1, ok1 := clusters[entity.ID]
			if !ok1 {
				continue
			}
			root1 := entity.ID
			for cluster1.rootID != root1 {
				cluster1 = clusters[cluster1.rootID]
				if cluster1 == nil {
					break
				}
				root1 = cluster1.rootID
			}
			if cluster1 == nil {
				continue
			}

			cluster2, ok2 := clusters[similarEntity.ID]
			if !ok2 {
				continue
			}
			root2 := similarEntity.ID
			for cluster2.rootID != root2 {
				cluster2 = clusters[cluster2.rootID]
				if cluster2 == nil {
					break
				}
				root2 = cluster2.rootID
			}
			if cluster2 == nil {
				continue
			}

			// Link clusters if they're different and above threshold
			if root1 != root2 {
				// Merge smaller cluster into larger one
				if root1 < root2 {
					clusters[root2].rootID = root1
					clusters[root1].memberIDs = append(clusters[root1].memberIDs, clusters[root2].memberIDs...)
					// Copy all similarities from the absorbed cluster
					for k, v := range clusters[root2].similarities {
						clusters[root1].similarities[k] = v
					}
					// Record this specific link's score
					clusters[root1].similarities[similarEntity.ID] = similarEntity.Score
					delete(clusters, root2)
				} else {
					clusters[root1].rootID = root2
					clusters[root2].memberIDs = append(clusters[root2].memberIDs, clusters[root1].memberIDs...)
					// Copy all similarities from the absorbed cluster
					for k, v := range clusters[root1].similarities {
						clusters[root2].similarities[k] = v
					}
					// Record this specific link's score
					clusters[root2].similarities[entity.ID] = similarEntity.Score
					delete(clusters, root1)
				}
			}
		}
	}

	// Now merge each cluster into its representative (lowest ID)
	merged := 0
	var potentialMerges []struct {
		sourceEntity models.Entity
		targetEntity models.Entity
		score        float64
	}

	processed := make(map[int]bool)

	for _, cluster := range clusters {
		if processed[cluster.rootID] {
			continue
		}

		// Only process clusters with more than one member
		if len(cluster.memberIDs) <= 1 {
			processed[cluster.rootID] = true
			continue
		}

		// Sort members by ID (ascending) - lowest ID is the target
		sort.Ints(cluster.memberIDs)

		// Skip if target is already processed
		targetID := cluster.memberIDs[0]
		if processed[targetID] {
			continue
		}
		processed[targetID] = true

		targetEntity := entityMap[targetID]

		// Merge all other members into the target
		for _, sourceID := range cluster.memberIDs[1:] {
			if processed[sourceID] {
				continue
			}
			processed[sourceID] = true

			sourceEntity := entityMap[sourceID]
			score := cluster.similarities[sourceID]
			if score == 0 {
				// If no direct score recorded, use threshold as default
				score = flags.SimilarityThreshold
			}

			if flags.Confirm && !flags.DryRun {
				fmt.Printf("    Merge entity %d (score: %.2f) into %d? (y/N): ", sourceID, score, targetID)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					continue
				}
			}

			if flags.DryRun {
				potentialMerges = append(potentialMerges, struct {
					sourceEntity models.Entity
					targetEntity models.Entity
					score        float64
				}{*sourceEntity, *targetEntity, score})
			} else {
				if err := app.Server.MergeEntities(ctx, userID, targetID, sourceID); err != nil {
					log.Printf("Failed to merge entities %d -> %d: %v", sourceID, targetID, err)
				} else {
					merged++
					fmt.Printf("    Merged entity %d (score: %.2f) into %d\n", sourceID, score, targetID)
				}
			}
		}
	}

	// Report dry run results
	if flags.DryRun {
		fmt.Printf("  DRY RUN: Found %d potential merges:\n", len(potentialMerges))
		for _, merge := range potentialMerges {
			fmt.Printf("    Would merge entity %d ('%s') [score: %.2f] into %d ('%s')\n",
				merge.sourceEntity.ID, merge.sourceEntity.Name, merge.score,
				merge.targetEntity.ID, merge.targetEntity.Name)
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

// deduplicateFactsForUser processes facts for a single user using cluster-based merging
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

	// Build similarity clusters using union-find
	// Only create links between facts above the similarity threshold
	type clusterInfo struct {
		rootID       int
		memberIDs    []int
		repFact      *models.Fact
		similarities map[int]float64 // score for each member to root
	}

	clusters := make(map[int]*clusterInfo) // root ID -> cluster info
	factMap := make(map[int]*models.Fact)  // ID -> fact

	// Build fact map for quick lookup
	for i := range facts {
		factMap[facts[i].ID] = &facts[i]
		clusters[facts[i].ID] = &clusterInfo{
			rootID:       facts[i].ID,
			memberIDs:    []int{facts[i].ID},
			repFact:      &facts[i],
			similarities: make(map[int]float64),
		}
	}

	// Build clusters by finding similar facts and linking them
	// We only create links between facts that are above threshold
	for _, fact := range facts {
		similarFacts, err := app.Server.FindSimilarFacts(ctx, fact, 100)
		if err != nil {
			log.Printf("Error finding similar facts for %d: %v", fact.ID, err)
			continue
		}

		for _, similarFact := range similarFacts {
			// Only link if above threshold and similarFact exists in our map
			if similarFact.Score < flags.SimilarityThreshold {
				continue
			}
			if _, exists := factMap[similarFact.ID]; !exists {
				continue
			}

			// Find roots of both facts (with nil checks in case cluster was deleted)
			cluster1, ok1 := clusters[fact.ID]
			if !ok1 {
				continue
			}
			root1 := fact.ID
			for cluster1.rootID != root1 {
				cluster1 = clusters[cluster1.rootID]
				if cluster1 == nil {
					break
				}
				root1 = cluster1.rootID
			}
			if cluster1 == nil {
				continue
			}

			cluster2, ok2 := clusters[similarFact.ID]
			if !ok2 {
				continue
			}
			root2 := similarFact.ID
			for cluster2.rootID != root2 {
				cluster2 = clusters[cluster2.rootID]
				if cluster2 == nil {
					break
				}
				root2 = cluster2.rootID
			}
			if cluster2 == nil {
				continue
			}

			// Link clusters if they're different and above threshold
			if root1 != root2 {
				// Merge smaller cluster into larger one (by lower ID)
				if root1 < root2 {
					clusters[root2].rootID = root1
					clusters[root1].memberIDs = append(clusters[root1].memberIDs, clusters[root2].memberIDs...)
					// Copy all similarities from the absorbed cluster
					for k, v := range clusters[root2].similarities {
						clusters[root1].similarities[k] = v
					}
					// Record this specific link's score
					clusters[root1].similarities[similarFact.ID] = similarFact.Score
					delete(clusters, root2)
				} else {
					clusters[root1].rootID = root2
					clusters[root2].memberIDs = append(clusters[root2].memberIDs, clusters[root1].memberIDs...)
					// Copy all similarities from the absorbed cluster
					for k, v := range clusters[root1].similarities {
						clusters[root2].similarities[k] = v
					}
					// Record this specific link's score
					clusters[root2].similarities[fact.ID] = similarFact.Score
					delete(clusters, root1)
				}
			}
		}
	}

	// Now merge each cluster into its representative (lowest ID)
	merged := 0
	var potentialMerges []struct {
		sourceFact models.Fact
		targetFact models.Fact
		score      float64
	}

	processed := make(map[int]bool)

	log.Printf("DEBUG: Total clusters after building: %d", len(clusters))
	multiMemberClusters := 0
	for _, cluster := range clusters {
		if len(cluster.memberIDs) > 1 {
			multiMemberClusters++
			log.Printf("DEBUG: Cluster root %d has %d members: %v", cluster.rootID, len(cluster.memberIDs), cluster.memberIDs)
		}
	}
	log.Printf("DEBUG: Clusters with >1 member: %d", multiMemberClusters)

	for _, cluster := range clusters {
		if processed[cluster.rootID] {
			continue
		}

		// Only process clusters with more than one member
		if len(cluster.memberIDs) <= 1 {
			processed[cluster.rootID] = true
			continue
		}

		// Sort members by ID (ascending) - lowest ID is the target
		sort.Ints(cluster.memberIDs)

		// Skip if target is already processed
		targetID := cluster.memberIDs[0]
		if processed[targetID] {
			continue
		}
		processed[targetID] = true

		targetFact := factMap[targetID]

		// Merge all other members into the target
		for _, sourceID := range cluster.memberIDs[1:] {
			if processed[sourceID] {
				continue
			}
			processed[sourceID] = true

			sourceFact := factMap[sourceID]
			score := cluster.similarities[sourceID]
			if score == 0 {
				// If no direct score recorded, use threshold as default
				score = flags.SimilarityThreshold
			}

			if flags.Confirm && !flags.DryRun {
				fmt.Printf("    Merge fact %d (score: %.2f) into %d? (y/N): ", sourceID, score, targetID)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					continue
				}
			}

			if flags.DryRun {
				potentialMerges = append(potentialMerges, struct {
					sourceFact models.Fact
					targetFact models.Fact
					score      float64
				}{*sourceFact, *targetFact, score})
			} else {
				if err := app.Server.MergeFacts(ctx, userID, targetID, sourceID); err != nil {
					log.Printf("Failed to merge facts %d -> %d: %v", sourceID, targetID, err)
				} else {
					merged++
					fmt.Printf("    Merged fact %d (score: %.2f) into %d\n", sourceID, score, targetID)
				}
			}
		}
	}

	// Report dry run results
	if flags.DryRun {
		fmt.Printf("  DRY RUN: Found %d potential merges:\n", len(potentialMerges))
		for _, merge := range potentialMerges {
			// Truncate long facts for readability
			sourceText := merge.sourceFact.Fact
			if len(sourceText) > 80 {
				sourceText = sourceText[:77] + "..."
			}
			targetText := merge.targetFact.Fact
			if len(targetText) > 80 {
				targetText = targetText[:77] + "..."
			}

			fmt.Printf("    Would merge fact %d ('%s') [score: %.2f] into %d ('%s')\n",
				merge.sourceFact.ID, sourceText, merge.score,
				merge.targetFact.ID, targetText)
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
