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
	if err := migrate(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return database, nil
}

// migrate adds columns introduced after a database's tables already existed.
// "CREATE TABLE IF NOT EXISTS" in schema.sql only affects brand new databases
// — an existing database keeps whatever columns it had when first created, so
// later additions need to be applied explicitly here.
func migrate(database *sql.DB) error {
	// vocab_progress / line_progress moved from a fixed streak+date model to
	// a full Anki-style state machine (state, step_index, ease_factor,
	// interval_days, lapses, due-as-datetime instead of next_review-as-date).
	// The only rows that ever existed under the old shape were zero-progress
	// rows from early development testing, so migrating them column-by-column
	// isn't worth the complexity — dropping and letting schema.sql recreate
	// the tables fresh is equivalent and much simpler.
	for _, table := range []string{"vocab_progress", "line_progress"} {
		hasNewShape, err := hasColumn(database, table, "state")
		if err != nil {
			return fmt.Errorf("check %s.state: %w", table, err)
		}
		if hasNewShape {
			continue
		}
		if _, err := database.Exec(fmt.Sprintf(`DROP TABLE %s`, table)); err != nil {
			return fmt.Errorf("drop old-shape %s: %w", table, err)
		}
	}
	if _, err := database.Exec(schemaSQL); err != nil {
		return fmt.Errorf("recreate dropped tables: %w", err)
	}

	columnMigrations := []struct {
		table  string
		column string
		ddl    string
	}{
		{"lines", "section", `ALTER TABLE lines ADD COLUMN section TEXT`},
	}

	for _, m := range columnMigrations {
		exists, err := hasColumn(database, m.table, m.column)
		if err != nil {
			return fmt.Errorf("check %s.%s: %w", m.table, m.column, err)
		}
		if exists {
			continue
		}
		if _, err := database.Exec(m.ddl); err != nil {
			return fmt.Errorf("add %s.%s: %w", m.table, m.column, err)
		}
	}
	return nil
}

func hasColumn(database *sql.DB, table, column string) (bool, error) {
	rows, err := database.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dfltValue any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
