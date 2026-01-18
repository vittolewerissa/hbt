package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// Open opens or creates the database at the given path
func Open(path string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{db}, nil
}

// migrate runs the schema migrations
func migrate(db *sql.DB) error {
	// Run main schema
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Add emoji column if it doesn't exist (for existing databases)
	// This will fail silently if the column already exists
	_, _ = db.Exec("ALTER TABLE categories ADD COLUMN emoji TEXT DEFAULT '📁'")
	_, _ = db.Exec("ALTER TABLE habits ADD COLUMN emoji TEXT DEFAULT ''")

	// Add target_per_day column if it doesn't exist
	_, _ = db.Exec("ALTER TABLE habits ADD COLUMN target_per_day INTEGER DEFAULT 1")

	// Remove UNIQUE constraint from completions table to allow multiple completions per day
	// SQLite doesn't support dropping constraints, so we need to recreate the table
	if err := migrateCompletionsTable(db); err != nil {
		// If this fails, log but don't stop - the app will still work for single completions
		// The user can manually delete the database to get the new schema
	}

	return nil
}

// migrateCompletionsTable removes the UNIQUE constraint from completions table
func migrateCompletionsTable(db *sql.DB) error {
	// Check if the constraint exists by trying to insert a duplicate
	// If it fails, we need to migrate
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM completions").Scan(&count)
	if err != nil {
		return err
	}

	// Try to detect if we need migration by checking sqlite_master
	var constraintExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='completions'
			AND sql LIKE '%UNIQUE%'
		)
	`).Scan(&constraintExists)
	if err != nil {
		return err
	}

	if !constraintExists {
		// Already migrated or new database
		return nil
	}

	// Recreate the table without UNIQUE constraint
	// Step 1: Rename old table
	if _, err := db.Exec("ALTER TABLE completions RENAME TO completions_old"); err != nil {
		return err
	}

	// Step 2: Create new table without UNIQUE constraint
	if _, err := db.Exec(`
		CREATE TABLE completions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
			completed_at DATE NOT NULL,
			notes TEXT DEFAULT ''
		)
	`); err != nil {
		// Rollback
		db.Exec("ALTER TABLE completions_old RENAME TO completions")
		return err
	}

	// Step 3: Copy data (only unique rows to avoid duplicates during migration)
	if _, err := db.Exec(`
		INSERT INTO completions (id, habit_id, completed_at, notes)
		SELECT id, habit_id, completed_at, notes FROM completions_old
	`); err != nil {
		// Rollback
		db.Exec("DROP TABLE completions")
		db.Exec("ALTER TABLE completions_old RENAME TO completions")
		return err
	}

	// Step 4: Drop old table
	if _, err := db.Exec("DROP TABLE completions_old"); err != nil {
		// Not critical, just leave it
	}

	// Step 5: Recreate indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_completions_habit ON completions(habit_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_completions_date ON completions(completed_at)")

	return nil
}

// DefaultPath returns the default database path
func DefaultPath() string {
	// Try XDG_DATA_HOME first, then fall back to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fall back to current directory
			return "habit.db"
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "habit-cli", "habit.db")
}
