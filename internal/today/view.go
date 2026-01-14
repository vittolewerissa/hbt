package today

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/habit-cli/internal/shared/db"
	"github.com/vittolewerissa/habit-cli/internal/shared/ui"
)

// Model is the today tab model
type Model struct {
	service *Service
	habits  []HabitWithStatus
	cursor  int
	width   int
	height  int
	keys    ui.KeyMap
	err     error
}

// New creates a new today model
func New(database *db.DB) Model {
	return Model{
		service: NewService(database),
		keys:    ui.DefaultKeyMap,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadData
}

// TodayLoadedMsg is sent when today's data is loaded
type TodayLoadedMsg struct {
	Habits []HabitWithStatus
	Err    error
}

// CompletionToggledMsg is sent when a completion is toggled
type CompletionToggledMsg struct {
	HabitID   int64
	Completed bool
	Err       error
}

func (m Model) loadData() tea.Msg {
	habits, err := m.service.GetHabitsForToday()
	return TodayLoadedMsg{Habits: habits, Err: err}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TodayLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.habits = msg.Habits
		return m, nil

	case CompletionToggledMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		// Reload data to refresh streaks
		return m, m.loadData

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.habits)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Toggle), key.Matches(msg, m.keys.Select):
		if len(m.habits) > 0 {
			habit := m.habits[m.cursor]
			return m, m.toggleCompletion(habit.ID)
		}
	}

	return m, nil
}

func (m Model) toggleCompletion(habitID int64) tea.Cmd {
	return func() tea.Msg {
		completed, err := m.service.ToggleCompletion(habitID)
		return CompletionToggledMsg{HabitID: habitID, Completed: completed, Err: err}
	}
}

// View renders the today tab (with title)
func (m Model) View() string {
	var s string
	date := time.Now().Format("Monday, January 2")
	s += ui.Title.Render("Today - " + date) + "\n\n"
	s += m.ViewContent()
	return s
}

// ViewContent renders just the content without title (for titled panels)
func (m Model) ViewContent() string {
	if m.err != nil {
		return ui.MutedText.Render(fmt.Sprintf("Error: %v", m.err))
	}

	var s string

	// Date subtitle
	date := time.Now().Format("Monday, January 2")
	s += ui.MutedText.Render(date) + "\n\n"

	if len(m.habits) == 0 {
		s += ui.MutedText.Render("No habits yet. Switch to the Habits tab to add some.")
		return s
	}

	// Count completed
	var completedCount, dueCount int
	for _, h := range m.habits {
		if h.IsDue {
			dueCount++
			if h.CompletedToday {
				completedCount++
			}
		}
	}

	// Progress
	if dueCount > 0 {
		progress := fmt.Sprintf("%d/%d completed", completedCount, dueCount)
		s += ui.Subtitle.Render(progress) + "\n"
	}

	// Habit list
	for i, habit := range m.habits {
		s += m.renderHabit(i, habit) + "\n"
	}

	return s
}

func (m Model) renderHabit(index int, habit HabitWithStatus) string {
	cursor := "  "
	if index == m.cursor {
		cursor = "> "
	}

	// Checkbox
	checkbox := "[ ]"
	if habit.CompletedToday {
		checkbox = "[x]"
	}

	// Style based on state
	var checkStyle, nameStyle lipgloss.Style
	if habit.CompletedToday {
		checkStyle = ui.CheckboxChecked
		nameStyle = ui.CompletedItem
	} else {
		checkStyle = ui.Checkbox
		if index == m.cursor {
			nameStyle = ui.SelectedItem
		} else {
			nameStyle = ui.NormalItem
		}
	}

	// Build the line
	line := cursor + checkStyle.Render(checkbox) + " " + nameStyle.Render(habit.Name)

	// Add streak badge if > 0
	if habit.CurrentStreak > 0 {
		streak := fmt.Sprintf(" %d", habit.CurrentStreak)
		line += ui.StreakBadge.Render(streak)
	}

	// Add category tag
	if habit.Category != nil {
		line += " " + ui.CategoryTag(habit.Category.Name, habit.Category.Color)
	}

	// Add frequency info for non-daily habits
	if habit.FrequencyType != "daily" {
		var freqInfo string
		switch habit.FrequencyType {
		case "weekly":
			if habit.CompletionsThisWeek > 0 {
				freqInfo = "(done this week)"
			} else {
				freqInfo = "(weekly)"
			}
		case "times_per_week":
			freqInfo = fmt.Sprintf("(%d/%d this week)", habit.CompletionsThisWeek, habit.FrequencyValue)
		}
		line += " " + ui.MutedText.Render(freqInfo)
	}

	return line
}

// Focused returns whether this view should receive key events
func (m Model) Focused() bool {
	return false
}
