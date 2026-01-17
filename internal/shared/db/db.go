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
