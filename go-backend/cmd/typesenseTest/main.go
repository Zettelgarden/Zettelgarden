package main

import (
	"context"
	"fmt"
	"go-backend/bootstrap"
	"go-backend/handlers"
	"log"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

func main() {
	s := bootstrap.InitServer()

	h := &handlers.Handler{
		Server: s,
		DB:     s.DB,
	}
	client := typesense.NewClient(
		typesense.WithServer("http://192.168.10.3:8108"),
		typesense.WithAPIKey("chamber"),
	)
	//	client.Collection("search").Delete(context.Background())
	sortField := "updated_at"
	schema := &api.CollectionSchema{
		Name: "search",
		Fields: []api.Field{
			{
				Name: "card_id",
				Type: "string",
			},
			{
				Name: "card_pk",
				Type: "int32",
			},
			{
				Name: "fact_pk",
				Type: "int32",
			},
			{
				Name: "entity_pk",
				Type: "int32",
			},
			{
				Name: "type",
				Type: "string",
			},
			{
				Name: "title",
				Type: "string",
			},
			{
				Name: "preview",
				Type: "string",
			},
			{
				Name: "score",
				Type: "float",
			},
			{
				Name: "created_at",
				Type: "int64",
			},
			{
				Name: "updated_at",
				Type: "int64",
			},
			{
				Name: "embedding",
				Type: "float[]",
				Embed: &struct {
					From        []string `json:"from"`
					ModelConfig struct {
						AccessToken  *string `json:"access_token,omitempty"`
						ApiKey       *string `json:"api_key,omitempty"`
						ClientId     *string `json:"client_id,omitempty"`
						ClientSecret *string `json:"client_secret,omitempty"`
						ModelName    string  `json:"model_name"`
						ProjectId    *string `json:"project_id,omitempty"`
					} `json:"model_config"`
				}{
					From: []string{"title", "preview"},
					ModelConfig: struct {
						AccessToken  *string `json:"access_token,omitempty"`
						ApiKey       *string `json:"api_key,omitempty"`
						ClientId     *string `json:"client_id,omitempty"`
						ClientSecret *string `json:"client_secret,omitempty"`
						ModelName    string  `json:"model_name"`
						ProjectId    *string `json:"project_id,omitempty"`
					}{
						ModelName: "ts/all-MiniLM-L12-v2",
					},
				},
			},
		},
		DefaultSortingField: &sortField,
	}
	_, err := client.Collections().Create(context.Background(), schema)
	if err != nil {
		log.Printf("Could not create collection: %v", err)
	}

	params := handlers.SearchRequestParams{
		SearchTerm:   "",
		FullText:     false,
		ShowEntities: true,
		ShowFacts:    true,
	}
	cards, err := h.ClassicCardSearch(1, params)
	log.Printf("%v cards", len(cards))

	// // Start timer and counters
	start := time.Now()
	// cardsIndexed := 0
	factsIndexed := 0
	// entitiesIndexed := 0

	// for _, card := range cards {
	// 	doc := map[string]interface{}{
	// 		"card_pk":    fmt.Sprintf("%d", card.ID), // ID must be string
	// 		"card_id":    card.CardID,
	// 		"type":       "card",                // static type since schema expects it
	// 		"title":      card.Title,            // your title field
	// 		"preview":    card.Body,             // use Body, or maybe a truncated version
	// 		"score":      0.0,                   // you can define logic to compute this
	// 		"created_at": card.CreatedAt.Unix(), // convert time.Time to Unix (int64)
	// 		"updated_at": card.UpdatedAt.Unix(),
	// 	}

	// 	// Upsert (insert or overwrite if exists)
	// 	_, err := client.Collection("search").
	// 		Documents().
	// 		Upsert(context.Background(), doc)

	// 	if err != nil {
	// 		log.Printf("failed to upsert card ID %d: %v", card.ID, err)
	// 	} else {
	// 		log.Printf("indexed card ID %d successfully", card.ID)
	// 	}
	// }
	// // Index all facts
	// rows, err := h.DB.Query(`
	// 	SELECT f.id, f.fact, f.created_at, f.updated_at,
	// 	       c.id, c.card_id, c.user_id, c.title, c.parent_id,
	// 	       c.created_at, c.updated_at
	// 	FROM facts f
	// 	JOIN cards c ON f.card_pk = c.id
	// 	WHERE f.user_id = $1
	// 	ORDER BY f.created_at DESC
	// `, 1) // assuming user_id=1 here
	// if err != nil {
	// 	log.Printf("error querying facts: %v", err)
	// } else {
	// 	defer rows.Close()
	// 	for rows.Next() {
	// 		var factID int
	// 		var factText string
	// 		var createdAtTime, updatedAtTime time.Time
	// 		var cardID int
	// 		var cardCardID string
	// 		var userID, parentID int
	// 		var cardTitle string
	// 		var cardCreatedAt, cardUpdatedAt time.Time
	// 		err := rows.Scan(
	// 			&factID, &factText, &createdAtTime, &updatedAtTime,
	// 			&cardID, &cardCardID, &userID, &cardTitle, &parentID, &cardCreatedAt, &cardUpdatedAt,
	// 		)
	// 		if err != nil {
	// 			log.Printf("error scanning fact: %v", err)
	// 			continue
	// 		}
	// 		doc := map[string]interface{}{
	// 			"fact_pk":    factID,
	// 			"card_id":    "",
	// 			"card_pk":    -1,
	// 			"entity_pk":  -1,
	// 			"type":       "fact",
	// 			"title":      factText,
	// 			"preview":    cardTitle,
	// 			"score":      0.0,
	// 			"created_at": createdAtTime.Unix(),
	// 			"updated_at": updatedAtTime.Unix(),
	// 		}
	// 		_, err = client.Collection("search").Documents().Upsert(context.Background(), doc)
	// 		if err != nil {
	// 			log.Printf("failed to upsert fact ID %d: %v", factID, err)
	// 		} else {
	// 			factsIndexed++
	// 			log.Printf("indexed fact ID %d successfully", factID)
	// 		}
	// 	}
	// }

	// // // Index all entities
	// rows2, err := h.DB.Query(`
	// 	SELECT id, name, description, type, created_at, updated_at
	// 	FROM entities
	// 	WHERE user_id = $1
	// 	ORDER BY created_at DESC
	// `, 1) // assuming user_id=1 here
	// if err != nil {
	// 	log.Printf("error querying entities: %v", err)
	// } else {
	// 	defer rows2.Close()
	// 	for rows2.Next() {
	// 		var entityID int
	// 		var name, description, etype string
	// 		var createdAtTime, updatedAtTime time.Time
	// 		err := rows2.Scan(&entityID, &name, &description, &etype, &createdAtTime, &updatedAtTime)
	// 		if err != nil {
	// 			log.Printf("error scanning entity: %v", err)
	// 			continue
	// 		}
	// 		doc := map[string]interface{}{
	// 			"entity_pk":  entityID,
	// 			"card_id":    "",
	// 			"card_pk":    -1,
	// 			"fact_pk":    -1,
	// 			"type":       "entity",
	// 			"title":      name,
	// 			"preview":    description,
	// 			"score":      0.0,
	// 			"created_at": createdAtTime.Unix(),
	// 			"updated_at": updatedAtTime.Unix(),
	// 		}
	// 		_, err = client.Collection("search").Documents().Upsert(context.Background(), doc)
	// 		if err != nil {
	// 			log.Printf("failed to upsert entity ID %d: %v", entityID, err)
	// 		} else {
	// 			log.Printf("indexed entity ID %d successfully", entityID)
	// 		}
	// 	}
	// }

	// Log summary
	elapsed := time.Since(start)
	log.Printf("time elapsed: %v, %v", elapsed, factsIndexed)
	// log.Printf("Indexed %d cards, %d facts, %d entities (total %d) in %s",
	// 	cardsIndexed, factsIndexed, entitiesIndexed, cardsIndexed+factsIndexed+entitiesIndexed, elapsed)

	perPage := 200
	sortBy := "_text_match:desc"
	searchParams := &api.SearchCollectionParams{
		Q:       "leib",            // the query text
		QueryBy: "title,embedding", // field(s) to search; can be comma-separated list
		SortBy:  &sortBy,

		PerPage: &perPage,
	}

	result, err := client.Collection("search").
		Documents().
		Search(context.Background(), searchParams)

	if err != nil {
		log.Fatalf("Search error: %v", err)
	}

	fmt.Printf("Found %d docs\n", *result.Found)
	for i, hit := range *result.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			title := doc["title"]
			log.Printf("[%d] type=%v, card_pk=%v, fact_pk=%v, entity_pk=%v, card_id=%v, Title=%v",
				i, doc["type"], doc["card_pk"], doc["fact_pk"], doc["entity_pk"], doc["card_id"], title)
		} else {
			log.Printf("[%d] unexpected document format: %v", i, hit.Document)
		}
	}
}
