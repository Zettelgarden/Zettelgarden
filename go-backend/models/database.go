package models

import (
	"context"
	"database/sql"
)

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	DatabaseName string
}

// Database is a unified interface that both *sql.DB and *sql.Tx implement.
// It allows code to work with either a database connection or a transaction
// interchangeably, which is essential for testability.
//
// During testing, a transaction is used and rolled back after each test.
// In production, the database connection is used directly.
type Database interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
