package category

import (
	"github.com/vl/habit-cli/internal/shared/db"
	"github.com/vl/habit-cli/internal/shared/model"
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
	// Set default color if not provided
	if c.Color == "" {
		c.Color = model.DefaultColors[0]
	}
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
