package bootstrap

import (
	"database/sql"
	"log"

	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/server"
)

// InitServer connects to the configured database (postgres or sqlite), runs
// migrations, and returns a ready *server.Server.
func InitServer(dbConfig config.DatabaseConfig) *server.Server {
	var (
		db  *sql.DB
		err error
	)
	if dbConfig.Driver == "sqlite" {
		db, err = server.OpenSQLite(dbConfig.SQLitePath)
	} else {
		// Convert config.DatabaseConfig to models.DatabaseConfig for the
		// PostgreSQL path (lib/pq retained through cutover — see Decision D6).
		db, err = server.ConnectToDatabase(models.DatabaseConfig{
			Host:         dbConfig.Host,
			Port:         dbConfig.Port,
			User:         dbConfig.User,
			Password:     dbConfig.Password,
			DatabaseName: dbConfig.DatabaseName,
		})
	}
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}

	schemaDir := "./schema"
	if dbConfig.Driver == "sqlite" {
		schemaDir = "./schema/sqlite"
	}

	s := &server.Server{
		DB:        db,
		SchemaDir: schemaDir,
		Driver:    dbConfig.Driver,
	}

	server.RunMigrations(s)
	return s
}
