package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// Open opens (creating if necessary) the SQLite database at path and applies
// the schema. Foreign keys are enabled since the DDL relies on ON DELETE CASCADE.
func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if _, err := database.Exec(schemaSQL); err != nil {
		database.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return database, nil
}
