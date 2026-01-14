package today

import (
	"time"

	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/model"
)

// Repository handles completion database operations
type Repository struct {
	db *db.DB
}

// NewRepository creates a new completion repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// IsCompletedOn checks if a habit is completed on a specific date
func (r *Repository) IsCompletedOn(habitID int64, date time.Time) (bool, error) {
	query := `SELECT COUNT(*) FROM completions WHERE habit_id = ? AND completed_at = ?`
	dateStr := date.Format("2006-01-02")
	var count int
	err := r.db.QueryRow(query, habitID, dateStr).Scan(&count)
	return count > 0, err
}

// Complete marks a habit as completed for a date
func (r *Repository) Complete(habitID int64, date time.Time, notes string) error {
	query := `INSERT OR REPLACE INTO completions (habit_id, completed_at, notes) VALUES (?, ?, ?)`
	dateStr := date.Format("2006-01-02")
	_, err := r.db.Exec(query, habitID, dateStr, notes)
	return err
}

// Uncomplete removes a completion for a date
func (r *Repository) Uncomplete(habitID int64, date time.Time) error {
	query := `DELETE FROM completions WHERE habit_id = ? AND completed_at = ?`
	dateStr := date.Format("2006-01-02")
	_, err := r.db.Exec(query, habitID, dateStr)
	return err
}

// GetCompletionsInRange returns all completions for a habit in a date range
func (r *Repository) GetCompletionsInRange(habitID int64, start, end time.Time) ([]model.Completion, error) {
	query := `
		SELECT id, habit_id, completed_at, notes
		FROM completions
		WHERE habit_id = ? AND completed_at >= ? AND completed_at <= ?
		ORDER BY completed_at DESC
	`
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	rows, err := r.db.Query(query, habitID, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var completions []model.Completion
	for rows.Next() {
		var c model.Completion
		var dateStr string
		if err := rows.Scan(&c.ID, &c.HabitID, &dateStr, &c.Notes); err != nil {
			return nil, err
		}
		c.CompletedAt, _ = time.Parse("2006-01-02", dateStr)
		completions = append(completions, c)
	}
	return completions, rows.Err()
}

// CountCompletionsThisWeek returns the number of completions this week
func (r *Repository) CountCompletionsThisWeek(habitID int64) (int, error) {
	// Get start of this week (Monday)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is day 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, time.Local)

	query := `SELECT COUNT(*) FROM completions WHERE habit_id = ? AND completed_at >= ?`
	startStr := startOfWeek.Format("2006-01-02")
	var count int
	err := r.db.QueryRow(query, habitID, startStr).Scan(&count)
	return count, err
}

// CalculateCurrentStreak calculates the current streak for a daily habit
func (r *Repository) CalculateCurrentStreak(habitID int64) (int, error) {
	query := `
		SELECT completed_at FROM completions
		WHERE habit_id = ?
		ORDER BY completed_at DESC
	`
	rows, err := r.db.Query(query, habitID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var streak int
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	expectedDate := today

	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, err
		}

		// First completion must be today or yesterday
		if streak == 0 {
			if dateStr != today && dateStr != yesterday {
				return 0, nil
			}
			expectedDate = dateStr
		}

		if dateStr == expectedDate {
			streak++
			// Calculate previous day
			d, _ := time.Parse("2006-01-02", expectedDate)
			expectedDate = d.AddDate(0, 0, -1).Format("2006-01-02")
		} else if dateStr < expectedDate {
			// Gap found, streak broken
			break
		}
		// Skip duplicate dates
	}

	return streak, rows.Err()
}

// CalculateBestStreak calculates the best streak ever for a habit
func (r *Repository) CalculateBestStreak(habitID int64) (int, error) {
	query := `
		SELECT completed_at FROM completions
		WHERE habit_id = ?
		ORDER BY completed_at ASC
	`
	rows, err := r.db.Query(query, habitID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var bestStreak, currentStreak int
	var lastDate string

	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, err
		}

		if lastDate == "" {
			currentStreak = 1
		} else {
			last, _ := time.Parse("2006-01-02", lastDate)
			curr, _ := time.Parse("2006-01-02", dateStr)

			daysDiff := int(curr.Sub(last).Hours() / 24)

			if daysDiff == 1 {
				currentStreak++
			} else if daysDiff > 1 {
				if currentStreak > bestStreak {
					bestStreak = currentStreak
				}
				currentStreak = 1
			}
			// daysDiff == 0 means duplicate, ignore
		}

		lastDate = dateStr
	}

	if currentStreak > bestStreak {
		bestStreak = currentStreak
	}

	return bestStreak, rows.Err()
}
