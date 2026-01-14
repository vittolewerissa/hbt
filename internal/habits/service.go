package habits

import (
	"github.com/vittolewerissa/habit-cli/internal/shared/db"
	"github.com/vittolewerissa/habit-cli/internal/shared/model"
)

// Service handles habit business logic
type Service struct {
	repo *Repository
}

// NewService creates a new habit service
func NewService(database *db.DB) *Service {
	return &Service{
		repo: NewRepository(database),
	}
}

// List returns all active habits
func (s *Service) List() ([]model.Habit, error) {
	return s.repo.List()
}

// ListAll returns all habits including archived
func (s *Service) ListAll() ([]model.Habit, error) {
	return s.repo.ListAll()
}

// Get returns a habit by ID
func (s *Service) Get(id int64) (*model.Habit, error) {
	return s.repo.GetByID(id)
}

// Create creates a new habit
func (s *Service) Create(h *model.Habit) error {
	// Set defaults
	if h.FrequencyType == "" {
		h.FrequencyType = model.FreqDaily
	}
	if h.FrequencyValue == 0 {
		h.FrequencyValue = 1
	}
	return s.repo.Create(h)
}

// Update updates an existing habit
func (s *Service) Update(h *model.Habit) error {
	return s.repo.Update(h)
}

// Archive archives a habit (soft delete)
func (s *Service) Archive(id int64) error {
	return s.repo.Archive(id)
}

// Unarchive restores an archived habit
func (s *Service) Unarchive(id int64) error {
	return s.repo.Unarchive(id)
}

// Delete permanently removes a habit
func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}
