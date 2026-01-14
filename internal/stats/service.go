package stats

import (
	"github.com/vl/habit-cli/internal/shared/db"
	"github.com/vl/habit-cli/internal/today"
)

// Service handles statistics business logic
type Service struct {
	repo      *Repository
	todayRepo *today.Repository
}

// NewService creates a new stats service
func NewService(database *db.DB) *Service {
	return &Service{
		repo:      NewRepository(database),
		todayRepo: today.NewRepository(database),
	}
}

// Overview contains overall statistics
type Overview struct {
	TotalHabits       int
	TotalCompletions  int
	TotalPossible     int
	OverallRate       float64
	CurrentBestStreak int
	AllTimeBestStreak int
}

// GetOverview returns overall statistics
func (s *Service) GetOverview() (*Overview, error) {
	completed, total, err := s.repo.GetOverallStats()
	if err != nil {
		return nil, err
	}

	habitStats, err := s.repo.GetHabitStats()
	if err != nil {
		return nil, err
	}

	overview := &Overview{
		TotalHabits:      len(habitStats),
		TotalCompletions: completed,
		TotalPossible:    total,
	}

	if total > 0 {
		overview.OverallRate = float64(completed) / float64(total) * 100
	}

	// Find best streaks
	for _, h := range habitStats {
		streak, _ := s.todayRepo.CalculateCurrentStreak(h.HabitID)
		if streak > overview.CurrentBestStreak {
			overview.CurrentBestStreak = streak
		}

		best, _ := s.todayRepo.CalculateBestStreak(h.HabitID)
		if best > overview.AllTimeBestStreak {
			overview.AllTimeBestStreak = best
		}
	}

	return overview, nil
}

// GetDailyStats returns daily completion stats
func (s *Service) GetDailyStats(days int) ([]DailyStats, error) {
	return s.repo.GetDailyStats(days)
}

// GetWeeklyStats returns weekly completion stats
func (s *Service) GetWeeklyStats(weeks int) ([]DailyStats, error) {
	return s.repo.GetWeeklyStats(weeks)
}

// GetHabitStats returns per-habit statistics
func (s *Service) GetHabitStats() ([]HabitStats, error) {
	stats, err := s.repo.GetHabitStats()
	if err != nil {
		return nil, err
	}

	// Add streak info
	for i := range stats {
		stats[i].CurrentStreak, _ = s.todayRepo.CalculateCurrentStreak(stats[i].HabitID)
		stats[i].BestStreak, _ = s.todayRepo.CalculateBestStreak(stats[i].HabitID)
	}

	return stats, nil
}
