package models

import "database/sql"

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	DatabaseName string
}

type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
