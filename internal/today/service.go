package today

import (
	"database/sql"
	"time"

	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/model"
)

// Service handles today's habits logic
type Service struct {
	db   *db.DB
	repo *Repository
}

// NewService creates a new today service
func NewService(database *db.DB) *Service {
	return &Service{
		db:   database,
		repo: NewRepository(database),
	}
}

// HabitWithStatus contains a habit with its completion status
type HabitWithStatus struct {
	model.Habit
	CompletedToday      bool // deprecated: use CompletionsToday >= TargetPerDay
	CompletionsToday    int
	CurrentStreak       int
	BestStreak          int
	CompletionsThisWeek int
	IsDue               bool
}

// GetHabitsForToday returns all habits with their status for today
func (s *Service) GetHabitsForToday() ([]HabitWithStatus, error) {
	// Get all active habits
	query := `
		SELECT h.id, h.name, h.description, h.emoji, h.category_id, h.frequency_type,
		       h.frequency_value, h.target_per_day, h.created_at, h.archived_at,
		       c.id, c.name, c.color, c.emoji
		FROM habits h
		LEFT JOIN categories c ON h.category_id = c.id
		WHERE h.archived_at IS NULL
		ORDER BY h.name
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	today := time.Now()
	var habits []HabitWithStatus

	for rows.Next() {
		var h model.Habit
		var categoryID, catID sql.NullInt64
		var catName, catColor, catEmoji sql.NullString
		var archivedAt sql.NullTime

		err := rows.Scan(
			&h.ID, &h.Name, &h.Description, &h.Emoji, &categoryID, &h.FrequencyType,
			&h.FrequencyValue, &h.TargetPerDay, &h.CreatedAt, &archivedAt,
			&catID, &catName, &catColor, &catEmoji,
		)
		if err != nil {
			return nil, err
		}

		if categoryID.Valid {
			h.CategoryID = &categoryID.Int64
		}
		if archivedAt.Valid {
			h.ArchivedAt = &archivedAt.Time
		}
		if catID.Valid {
			h.Category = &model.Category{
				ID:    catID.Int64,
				Name:  catName.String,
				Color: catColor.String,
				Emoji: catEmoji.String,
			}
		}

		// Get completion status
		completionsToday, _ := s.repo.CountCompletionsOn(h.ID, today)
		completionsThisWeek, _ := s.repo.CountCompletionsThisWeek(h.ID)
		currentStreak, _ := s.repo.CalculateCurrentStreak(h.ID)
		bestStreak, _ := s.repo.CalculateBestStreak(h.ID)

		status := HabitWithStatus{
			Habit:               h,
			CompletedToday:      completionsToday >= h.TargetPerDay,
			CompletionsToday:    completionsToday,
			CurrentStreak:       currentStreak,
			BestStreak:          bestStreak,
			CompletionsThisWeek: completionsThisWeek,
			IsDue:               h.IsDueToday(completionsThisWeek),
		}

		habits = append(habits, status)
	}

	return habits, rows.Err()
}

// ToggleCompletion toggles the completion status for today
// If there are any completions, it removes one. Otherwise, it adds one.
func (s *Service) ToggleCompletion(habitID int64) (bool, error) {
	today := time.Now()
	count, err := s.repo.CountCompletionsOn(habitID, today)
	if err != nil {
		return false, err
	}

	if count > 0 {
		err = s.repo.Uncomplete(habitID, today)
		return false, err
	}
	err = s.repo.Complete(habitID, today, "")
	return true, err
}

// CompleteWithNotes marks a habit as completed with notes
func (s *Service) CompleteWithNotes(habitID int64, notes string) error {
	return s.repo.Complete(habitID, time.Now(), notes)
}
