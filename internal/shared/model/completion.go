package model

import "time"

// Completion records a habit completion for a specific date
type Completion struct {
	ID          int64
	HabitID     int64
	CompletedAt time.Time // Date only (no time component)
	Notes       string
}
