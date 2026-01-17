package category

import (
	"database/sql"

	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/model"
)

// Repository handles category database operations
type Repository struct {
	db *db.DB
}

// NewRepository creates a new category repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// List returns all categories
func (r *Repository) List() ([]model.Category, error) {
	query := `SELECT id, name, color, emoji, created_at FROM categories ORDER BY name`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Emoji, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// GetByID returns a category by ID
func (r *Repository) GetByID(id int64) (*model.Category, error) {
	query := `SELECT id, name, color, emoji, created_at FROM categories WHERE id = ?`
	var c model.Category
	err := r.db.QueryRow(query, id).Scan(&c.ID, &c.Name, &c.Color, &c.Emoji, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Create creates a new category
func (r *Repository) Create(c *model.Category) error {
	query := `INSERT INTO categories (name, color, emoji) VALUES (?, ?, ?)`
	result, err := r.db.Exec(query, c.Name, c.Color, c.Emoji)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

// Update updates an existing category
func (r *Repository) Update(c *model.Category) error {
	query := `UPDATE categories SET name = ?, color = ?, emoji = ? WHERE id = ?`
	_, err := r.db.Exec(query, c.Name, c.Color, c.Emoji, c.ID)
	return err
}

// Delete removes a category
func (r *Repository) Delete(id int64) error {
	query := `DELETE FROM categories WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}
