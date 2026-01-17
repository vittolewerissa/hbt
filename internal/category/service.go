package category

import (
	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/model"
)

// Service handles category business logic
type Service struct {
	repo *Repository
}

// NewService creates a new category service
func NewService(database *db.DB) *Service {
	return &Service{
		repo: NewRepository(database),
	}
}

// List returns all categories
func (s *Service) List() ([]model.Category, error) {
	return s.repo.List()
}

// Get returns a category by ID
func (s *Service) Get(id int64) (*model.Category, error) {
	return s.repo.GetByID(id)
}

// Create creates a new category
func (s *Service) Create(c *model.Category) error {
	// Set default color if not provided (kept for DB compatibility)
	if c.Color == "" {
		c.Color = "#CCCCCC"
	}
	// Emoji is optional - allow empty string
	return s.repo.Create(c)
}

// Update updates an existing category
func (s *Service) Update(c *model.Category) error {
	return s.repo.Update(c)
}

// Delete removes a category
func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}
