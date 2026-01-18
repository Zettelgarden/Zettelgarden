package bootstrap

import (
	"log"

	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/server"
)

func InitServer(dbConfig config.DatabaseConfig) *server.Server {
	// Convert config.DatabaseConfig to models.DatabaseConfig
	modelsDBConfig := models.DatabaseConfig{
		Host:         dbConfig.Host,
		Port:         dbConfig.Port,
		User:         dbConfig.User,
		Password:     dbConfig.Password,
		DatabaseName: dbConfig.DatabaseName,
	}

	db, err := server.ConnectToDatabase(modelsDBConfig)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}

	s := &server.Server{
		DB:        db,
		SchemaDir: "./schema",
	}

	server.RunMigrations(s)
	return s
}
