package habits

import (
	"database/sql"
	"time"

	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/model"
)

// Repository handles habit database operations
type Repository struct {
	db *db.DB
}

// NewRepository creates a new habit repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// List returns all non-archived habits
func (r *Repository) List() ([]model.Habit, error) {
	query := `
		SELECT h.id, h.name, h.description, h.emoji, h.category_id, h.frequency_type,
		       h.frequency_value, h.target_per_day, h.created_at, h.archived_at,
		       c.id, c.name, c.color, c.emoji
		FROM habits h
		LEFT JOIN categories c ON h.category_id = c.id
		WHERE h.archived_at IS NULL
		ORDER BY c.name NULLS LAST, h.name
	`
	return r.queryHabits(query)
}

// ListAll returns all habits including archived
func (r *Repository) ListAll() ([]model.Habit, error) {
	query := `
		SELECT h.id, h.name, h.description, h.emoji, h.category_id, h.frequency_type,
		       h.frequency_value, h.target_per_day, h.created_at, h.archived_at,
		       c.id, c.name, c.color, c.emoji
		FROM habits h
		LEFT JOIN categories c ON h.category_id = c.id
		ORDER BY h.archived_at IS NULL DESC, h.name
	`
	return r.queryHabits(query)
}

// GetByID returns a habit by ID
func (r *Repository) GetByID(id int64) (*model.Habit, error) {
	query := `
		SELECT h.id, h.name, h.description, h.emoji, h.category_id, h.frequency_type,
		       h.frequency_value, h.target_per_day, h.created_at, h.archived_at,
		       c.id, c.name, c.color, c.emoji
		FROM habits h
		LEFT JOIN categories c ON h.category_id = c.id
		WHERE h.id = ?
	`
	habits, err := r.queryHabits(query, id)
	if err != nil {
		return nil, err
	}
	if len(habits) == 0 {
		return nil, sql.ErrNoRows
	}
	return &habits[0], nil
}

// Create creates a new habit
func (r *Repository) Create(h *model.Habit) error {
	query := `
		INSERT INTO habits (name, description, emoji, category_id, frequency_type, frequency_value, target_per_day)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, h.Name, h.Description, h.Emoji, h.CategoryID, h.FrequencyType, h.FrequencyValue, h.TargetPerDay)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	h.ID = id
	h.CreatedAt = time.Now()
	return nil
}

// Update updates an existing habit
func (r *Repository) Update(h *model.Habit) error {
	query := `
		UPDATE habits
		SET name = ?, description = ?, emoji = ?, category_id = ?, frequency_type = ?, frequency_value = ?, target_per_day = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, h.Name, h.Description, h.Emoji, h.CategoryID, h.FrequencyType, h.FrequencyValue, h.TargetPerDay, h.ID)
	return err
}

// Archive archives a habit
func (r *Repository) Archive(id int64) error {
	query := `UPDATE habits SET archived_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Unarchive unarchives a habit
func (r *Repository) Unarchive(id int64) error {
	query := `UPDATE habits SET archived_at = NULL WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// Delete permanently deletes a habit
func (r *Repository) Delete(id int64) error {
	query := `DELETE FROM habits WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *Repository) queryHabits(query string, args ...interface{}) ([]model.Habit, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []model.Habit
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

		habits = append(habits, h)
	}

	return habits, rows.Err()
}
