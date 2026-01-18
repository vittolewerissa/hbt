package model

import "time"

// FrequencyType defines how often a habit should be completed
type FrequencyType string

const (
	FreqDaily        FrequencyType = "daily"
	FreqWeekly       FrequencyType = "weekly"
	FreqTimesPerWeek FrequencyType = "times_per_week"
)

// Habit represents a trackable habit
type Habit struct {
	ID             int64
	Name           string
	Description    string
	Emoji          string
	CategoryID     *int64
	FrequencyType  FrequencyType
	FrequencyValue int // times per week if FrequencyType is times_per_week
	TargetPerDay   int // how many times to complete per day
	CreatedAt      time.Time
	ArchivedAt     *time.Time

	// Joined fields (not stored directly)
	Category           *Category
	CurrentStreak      int
	BestStreak         int
	CompletionsToday   int  // how many times completed today
	CompletedToday     bool // deprecated: use CompletionsToday >= TargetPerDay
}

// IsArchived returns true if the habit has been archived
func (h *Habit) IsArchived() bool {
	return h.ArchivedAt != nil
}

// IsDueToday returns true if the habit should be completed today
func (h *Habit) IsDueToday(completionsThisWeek int) bool {
	switch h.FrequencyType {
	case FreqDaily:
		return true
	case FreqWeekly:
		// Due once per week - not yet completed this week
		return completionsThisWeek == 0
	case FreqTimesPerWeek:
		// Due if we haven't hit the target this week
		return completionsThisWeek < h.FrequencyValue
	default:
		return true
	}
}
