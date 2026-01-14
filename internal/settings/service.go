package settings

import (
	"github.com/vittolewerissa/hbt/internal/shared/db"
)

// Service handles settings management
type Service struct {
	db *db.DB
}

// NewService creates a new settings service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Get retrieves a setting value
func (s *Service) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	return value, err
}

// Set stores a setting value
func (s *Service) Set(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

// GetAll retrieves all settings
func (s *Service) GetAll() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

// Settings keys
const (
	KeyDatabasePath = "database_path"
	KeyTheme        = "theme"
	KeyWeekStart    = "week_start" // 0=Sunday, 1=Monday
)

// Defaults
var Defaults = map[string]string{
	KeyTheme:     "default",
	KeyWeekStart: "1", // Monday
}
