package stats

import (
	"time"

	"github.com/vittolewerissa/habit-cli/internal/shared/db"
)

// Repository handles statistics database operations
type Repository struct {
	db *db.DB
}

// NewRepository creates a new stats repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// DailyStats represents completion stats for a single day
type DailyStats struct {
	Date      time.Time
	Completed int
	Total     int
}

// GetDailyStats returns completion stats for the last N days
func (r *Repository) GetDailyStats(days int) ([]DailyStats, error) {
	query := `
		WITH RECURSIVE dates(date) AS (
			SELECT date('now', 'localtime')
			UNION ALL
			SELECT date(date, '-1 day')
			FROM dates
			WHERE date > date('now', 'localtime', ? || ' days')
		),
		daily_habits AS (
			SELECT d.date, h.id as habit_id
			FROM dates d
			CROSS JOIN habits h
			WHERE h.archived_at IS NULL
			AND h.frequency_type = 'daily'
			AND date(h.created_at) <= d.date
		)
		SELECT
			dh.date,
			COUNT(DISTINCT c.habit_id) as completed,
			COUNT(DISTINCT dh.habit_id) as total
		FROM daily_habits dh
		LEFT JOIN completions c ON dh.habit_id = c.habit_id AND dh.date = c.completed_at
		GROUP BY dh.date
		ORDER BY dh.date DESC
	`

	rows, err := r.db.Query(query, -days+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		var dateStr string
		if err := rows.Scan(&dateStr, &s.Completed, &s.Total); err != nil {
			return nil, err
		}
		s.Date, _ = time.Parse("2006-01-02", dateStr)
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetWeeklyStats returns completion stats for the last N weeks
func (r *Repository) GetWeeklyStats(weeks int) ([]DailyStats, error) {
	query := `
		WITH RECURSIVE week_starts(week_start) AS (
			SELECT date('now', 'localtime', 'weekday 0', '-6 days')
			UNION ALL
			SELECT date(week_start, '-7 days')
			FROM week_starts
			WHERE week_start > date('now', 'localtime', ? || ' days')
		)
		SELECT
			ws.week_start as date,
			(SELECT COUNT(DISTINCT c.id) FROM completions c
			 JOIN habits h ON c.habit_id = h.id
			 WHERE h.archived_at IS NULL
			 AND c.completed_at >= ws.week_start
			 AND c.completed_at < date(ws.week_start, '+7 days')) as completed,
			(SELECT COUNT(*) * 7 FROM habits h
			 WHERE h.archived_at IS NULL
			 AND h.frequency_type = 'daily'
			 AND date(h.created_at) <= date(ws.week_start, '+6 days')) as total
		FROM week_starts ws
		ORDER BY ws.week_start DESC
	`

	rows, err := r.db.Query(query, -weeks*7)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		var dateStr string
		if err := rows.Scan(&dateStr, &s.Completed, &s.Total); err != nil {
			return nil, err
		}
		s.Date, _ = time.Parse("2006-01-02", dateStr)
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// HabitStats represents statistics for a single habit
type HabitStats struct {
	HabitID       int64
	HabitName     string
	CurrentStreak int
	BestStreak    int
	TotalDays     int
	CompletedDays int
	CompletionRate float64
}

// GetHabitStats returns detailed stats for all habits
func (r *Repository) GetHabitStats() ([]HabitStats, error) {
	query := `
		SELECT
			h.id,
			h.name,
			COALESCE(
				(SELECT COUNT(*) FROM completions c WHERE c.habit_id = h.id),
				0
			) as completed_days,
			CAST(julianday('now', 'localtime') - julianday(h.created_at) + 1 AS INTEGER) as total_days
		FROM habits h
		WHERE h.archived_at IS NULL
		ORDER BY h.name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []HabitStats
	for rows.Next() {
		var s HabitStats
		if err := rows.Scan(&s.HabitID, &s.HabitName, &s.CompletedDays, &s.TotalDays); err != nil {
			return nil, err
		}
		if s.TotalDays > 0 {
			s.CompletionRate = float64(s.CompletedDays) / float64(s.TotalDays) * 100
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetOverallStats returns overall completion statistics
func (r *Repository) GetOverallStats() (completed, total int, err error) {
	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM completions c
			 JOIN habits h ON c.habit_id = h.id
			 WHERE h.archived_at IS NULL), 0) as completed,
			COALESCE((SELECT SUM(
				CAST(julianday('now', 'localtime') - julianday(h.created_at) + 1 AS INTEGER)
			 ) FROM habits h
			 WHERE h.archived_at IS NULL
			 AND h.frequency_type = 'daily'), 0) as total
	`
	err = r.db.QueryRow(query).Scan(&completed, &total)
	return
}
