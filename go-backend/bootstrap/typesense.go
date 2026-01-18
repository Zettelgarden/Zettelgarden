// Package bootstrap provides initialization functions for optional services
package bootstrap

import (
	"context"
	"fmt"
	"log"

	"go-backend/pkg/config"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

func GetTypesenseClient(searchConfig config.SearchConfig) *typesense.Client {
	log.Printf("Initializing Typesense client with host: %s", searchConfig.Host)
	client := typesense.NewClient(
		typesense.WithServer(searchConfig.Host),
		typesense.WithAPIKey(searchConfig.Password),
	)
	return client
}

// InitTypesense initializes Typesense search service.
// This is an optional service - if initialization fails, search will still work
// but will use slower SQL-based full-text search instead of vector search.
func InitTypesense(searchConfig config.SearchConfig) (*typesense.Client, error) {

	ctx := context.Background()

	client := GetTypesenseClient(searchConfig)

	collectionName := searchConfig.Collection

	_, err := client.Collection(collectionName).Retrieve(ctx)
	if err == nil {
		// Collection exists
		fmt.Println("Collection already exists:", collectionName)
		return client, nil
	}
	// log.Printf("delete")
	// client.Collection(collectionName).Delete(context.Background())

	sortField := "updated_at"
	schema := &api.CollectionSchema{
		Name: collectionName,
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
				Name: "user_id",
				Type: "int32",
			},
			{
				Name: "parent_id",
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
				Name: "created_at",
				Type: "int64",
			},
			{
				Name: "updated_at",
				Type: "int64",
			},
			{
				Name: "linked_card_pk",
				Type: "int32",
			},
			{
				Name: "linked_card_id",
				Type: "string",
			},
			{
				Name: "linked_card_title",
				Type: "string",
			},
			{
				Name: "linked_card_parent_id",
				Type: "int32",
			},
			{
				Name: "tags",
				Type: "string[]",
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
	_, err = client.Collections().Create(context.Background(), schema)
	return client, err
}
