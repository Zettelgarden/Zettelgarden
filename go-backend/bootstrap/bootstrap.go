package bootstrap

import (
	"log"

	"go-backend/pkg/config"
	"go-backend/server"
)

// InitServer opens the SQLite database, runs migrations, and returns a ready
// *server.Server.
//
// The Postgres backend was retired after the cutover (epic Zettelgarden-c7j,
// Phase 7b): the backend is unconditionally SQLite now, so there is no
// driver branch. lib/pq, ConnectToDatabase, and the numbered Postgres
// migrations under schema/ have been removed/archived.
func InitServer(dbConfig config.DatabaseConfig) *server.Server {
	db, err := server.OpenSQLite(dbConfig.SQLitePath)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}

	s := &server.Server{
		DB:        db,
		SchemaDir: "./schema/sqlite",
	}

	server.RunMigrations(s)
	return s
}
