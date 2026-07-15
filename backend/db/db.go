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
// the schema. Foreign keys are enabled since the DDL relies on ON DELETE
// CASCADE. WAL journal mode lets readers proceed without waiting on a
// writer (the default rollback-journal mode locks the whole database file
// for the duration of a write transaction); busy_timeout makes a writer
// that does have to wait for another writer retry for up to 5s instead of
// failing (or, without a timeout at all, appearing to hang indefinitely
// from the caller's perspective) the instant it hits SQLITE_BUSY. Without
// both of these, concurrent requests — this app's own empty-queue refresh
// timer landing at the same moment as a page navigation, for example — can
// stall each other out long enough to trip the frontend's request timeout.
func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// database/sql pools connections and, left at its default, will happily
	// open as many as concurrent requests demand. Each one is a real
	// os-thread-blocking cgo call into SQLite (mattn/go-sqlite3 links the
	// real C library), so a burst of concurrent requests — WithActiveUser's
	// per-request profile lookup running in front of every single handler,
	// stacked with whatever that handler itself queries, stacked with the
	// drill pages' background refresh timers — can end up wanting several
	// simultaneous connections to the same SQLite file. Capping this at 1
	// forces every query in the process through a single real connection;
	// database/sql then queues the rest in-process (cheap, no OS/network
	// cost) instead of racing multiple cgo threads against SQLite's own
	// file locking, which is what busy_timeout above is a safety net for
	// but shouldn't need to be exercised at all under normal use.
	database.SetMaxOpenConns(1)
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

	defaultUserID, err := ensureDefaultUser(database)
	if err != nil {
		return fmt.Errorf("ensure default profile: %w", err)
	}
	if err := migrateProgressTableToProfiles(database, "vocab_progress",
		"song_id, vocab_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen",
		defaultUserID); err != nil {
		return err
	}
	if err := migrateProgressTableToProfiles(database, "line_progress",
		"line_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen",
		defaultUserID); err != nil {
		return err
	}

	columnMigrations := []struct {
		table  string
		column string
		ddl    string
	}{
		{"lines", "section", `ALTER TABLE lines ADD COLUMN section TEXT`},
		{"vocab_progress", "introduced_at", `ALTER TABLE vocab_progress ADD COLUMN introduced_at TEXT`},
		{"line_progress", "introduced_at", `ALTER TABLE line_progress ADD COLUMN introduced_at TEXT`},
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

// ensureDefaultUser guarantees at least one profile exists, returning its
// id — used both as the fallback "active profile" when no cookie names one,
// and here to backfill any pre-profiles progress rows found below.
func ensureDefaultUser(database *sql.DB) (int64, error) {
	var id int64
	err := database.QueryRow(`SELECT id FROM users ORDER BY id ASC LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := database.Exec(`INSERT INTO users (display_name) VALUES (?)`, "Player 1")
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// migrateProgressTableToProfiles adds user_id to a pre-profiles
// vocab_progress/line_progress table. SQLite can't add a table-level UNIQUE
// constraint via ALTER TABLE, and the old constraint (song_id/vocab_id, or
// line_id alone) is no longer correct once more than one profile exists — so
// this rebuilds the table: rename the existing one aside, let schema.sql
// create the new (profile-scoped) shape, copy every row across tagged with
// defaultUserID, then drop the old table. Unlike the state-column migration
// above, this is real progress history (not disposable test data), so it
// must be preserved, not dropped.
func migrateProgressTableToProfiles(database *sql.DB, table, sharedColumns string, defaultUserID int64) error {
	hasUserID, err := hasColumn(database, table, "user_id")
	if err != nil {
		return fmt.Errorf("check %s.user_id: %w", table, err)
	}
	if hasUserID {
		return nil
	}

	oldTable := table + "_pre_profiles"
	if _, err := database.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, table, oldTable)); err != nil {
		return fmt.Errorf("rename %s for profile migration: %w", table, err)
	}
	if _, err := database.Exec(schemaSQL); err != nil {
		return fmt.Errorf("recreate %s with profile support: %w", table, err)
	}
	copySQL := fmt.Sprintf(
		`INSERT INTO %s (user_id, %s) SELECT ?, %s FROM %s`,
		table, sharedColumns, sharedColumns, oldTable,
	)
	if _, err := database.Exec(copySQL, defaultUserID); err != nil {
		return fmt.Errorf("copy %s rows to profile-scoped table: %w", table, err)
	}
	if _, err := database.Exec(fmt.Sprintf(`DROP TABLE %s`, oldTable)); err != nil {
		return fmt.Errorf("drop %s: %w", oldTable, err)
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
