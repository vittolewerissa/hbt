package stats

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/ui"
)

// View modes
type viewMode int

const (
	modeOverview viewMode = iota
	modeHabits
)

// Model is the stats tab model
type Model struct {
	service     *Service
	overview    *Overview
	habitStats  []HabitStats
	dailyStats  []DailyStats
	mode        viewMode
	cursor      int
	width       int
	height      int
	keys        ui.KeyMap
	err         error
}

// New creates a new stats model
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

// StatsLoadedMsg is sent when stats are loaded
type StatsLoadedMsg struct {
	Overview   *Overview
	HabitStats []HabitStats
	DailyStats []DailyStats
	Err        error
}

func (m Model) loadData() tea.Msg {
	overview, err := m.service.GetOverview()
	if err != nil {
		return StatsLoadedMsg{Err: err}
	}

	habitStats, err := m.service.GetHabitStats()
	if err != nil {
		return StatsLoadedMsg{Err: err}
	}

	dailyStats, err := m.service.GetDailyStats(14) // Last 2 weeks
	if err != nil {
		return StatsLoadedMsg{Err: err}
	}

	return StatsLoadedMsg{
		Overview:   overview,
		HabitStats: habitStats,
		DailyStats: dailyStats,
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatsLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.overview = msg.Overview
		m.habitStats = msg.HabitStats
		m.dailyStats = msg.DailyStats
		return m, nil

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
	case key.Matches(msg, m.keys.Left):
		if m.mode > modeOverview {
			m.mode--
		}
	case key.Matches(msg, m.keys.Right):
		if m.mode < modeHabits {
			m.mode++
		}
	case key.Matches(msg, m.keys.Up):
		if m.mode == modeHabits && m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.mode == modeHabits && m.cursor < len(m.habitStats)-1 {
			m.cursor++
		}
	}

	return m, nil
}

// View renders the stats tab
func (m Model) View() string {
	if m.err != nil {
		return ui.MutedText.Render(fmt.Sprintf("Error: %v", m.err))
	}

	var s string
	s += ui.Title.Render("Statistics") + "\n\n"

	// Tab selector
	tabs := []string{"Overview", "Per Habit"}
	var tabLine string
	for i, tab := range tabs {
		if viewMode(i) == m.mode {
			tabLine += ui.SelectedItem.Render("["+tab+"]") + "  "
		} else {
			tabLine += ui.MutedText.Render(" "+tab+" ") + "  "
		}
	}
	s += tabLine + "\n\n"

	switch m.mode {
	case modeOverview:
		s += m.renderOverview()
	case modeHabits:
		s += m.renderHabitStats()
	}

	s += "\n" + ui.MutedText.Render("←/→: switch view  ↑/↓: navigate")

	return s
}

func (m Model) renderOverview() string {
	if m.overview == nil {
		return ui.MutedText.Render("Loading...")
	}

	var s string

	// Summary stats
	s += lipgloss.NewStyle().Bold(true).Render("Summary") + "\n"
	s += fmt.Sprintf("  Total habits: %d\n", m.overview.TotalHabits)
	s += fmt.Sprintf("  Total completions: %d\n", m.overview.TotalCompletions)
	s += fmt.Sprintf("  Overall rate: %.1f%%\n", m.overview.OverallRate)
	s += fmt.Sprintf("  Current best streak: %d days\n", m.overview.CurrentBestStreak)
	s += fmt.Sprintf("  All-time best streak: %d days\n", m.overview.AllTimeBestStreak)
	s += "\n"

	// Daily sparkline
	if len(m.dailyStats) > 0 {
		s += lipgloss.NewStyle().Bold(true).Render("Last 14 Days") + "\n"

		// Convert to percentages for sparkline
		var values []float64
		for i := len(m.dailyStats) - 1; i >= 0; i-- {
			stat := m.dailyStats[i]
			if stat.Total > 0 {
				values = append(values, float64(stat.Completed)/float64(stat.Total)*100)
			} else {
				values = append(values, 0)
			}
		}

		spark := NewSparkline(40)
		s += "  " + spark.Render(values) + "\n"

		// Show recent days as bar chart
		s += "\n"
		chart := NewBarChart(m.width - 10)

		limit := 7
		if len(m.dailyStats) < limit {
			limit = len(m.dailyStats)
		}

		for i := 0; i < limit; i++ {
			stat := m.dailyStats[i]
			var pct float64
			if stat.Total > 0 {
				pct = float64(stat.Completed) / float64(stat.Total) * 100
			}
			label := stat.Date.Format("Mon 01/02")
			s += "  " + chart.Render(pct, label) + "\n"
		}
	}

	return s
}

func (m Model) renderHabitStats() string {
	if len(m.habitStats) == 0 {
		return ui.MutedText.Render("No habits yet.")
	}

	var s string
	chart := NewBarChart(m.width - 15)

	for i, stat := range m.habitStats {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		// Name line
		nameStyle := ui.NormalItem
		if i == m.cursor {
			nameStyle = ui.SelectedItem
		}
		s += cursor + nameStyle.Render(stat.HabitName) + "\n"

		// Stats line
		streakInfo := fmt.Sprintf("    Streak: %d (best: %d)", stat.CurrentStreak, stat.BestStreak)
		s += ui.MutedText.Render(streakInfo) + "\n"

		// Completion bar
		s += "    " + chart.Render(stat.CompletionRate, "") + "\n"
		s += "\n"
	}

	return s
}

// Focused returns whether this view should receive key events
func (m Model) Focused() bool {
	return false
}

// RenderPanel renders a compact stats panel for the sidebar
func (m Model) RenderPanel(width, height int) string {
	var s string

	if m.err != nil {
		s += ui.MutedText.Render("Error loading stats")
		return ui.TitledPanel("Stats", s, width, height)
	}

	if m.overview == nil {
		s += ui.MutedText.Render("Loading...")
		return ui.TitledPanel("Stats", s, width, height)
	}

	// Today's progress
	s += lipgloss.NewStyle().Bold(true).Render("Today") + "\n"
	todayCompleted := 0
	todayTotal := 0
	if len(m.dailyStats) > 0 {
		todayCompleted = m.dailyStats[0].Completed
		todayTotal = m.dailyStats[0].Total
	}
	if todayTotal > 0 {
		pct := float64(todayCompleted) / float64(todayTotal) * 100
		s += fmt.Sprintf("  %d/%d completed (%.0f%%)\n", todayCompleted, todayTotal, pct)
	} else {
		s += "  No habits due\n"
	}
	s += "\n"

	// Streaks
	s += lipgloss.NewStyle().Bold(true).Render("Streaks") + "\n"
	s += fmt.Sprintf("  Current best: %d days\n", m.overview.CurrentBestStreak)
	s += fmt.Sprintf("  All-time: %d days\n", m.overview.AllTimeBestStreak)
	s += "\n"

	// Weekly sparkline
	if len(m.dailyStats) > 0 {
		s += lipgloss.NewStyle().Bold(true).Render("Last 7 Days") + "\n"

		var values []float64
		limit := 7
		if len(m.dailyStats) < limit {
			limit = len(m.dailyStats)
		}
		for i := limit - 1; i >= 0; i-- {
			stat := m.dailyStats[i]
			if stat.Total > 0 {
				values = append(values, float64(stat.Completed)/float64(stat.Total)*100)
			} else {
				values = append(values, 0)
			}
		}

		spark := NewSparkline(width - 8)
		s += "  " + spark.Render(values) + "\n"
	}
	s += "\n"

	// Overall stats
	s += lipgloss.NewStyle().Bold(true).Render("Overall") + "\n"
	s += fmt.Sprintf("  %d habits\n", m.overview.TotalHabits)
	s += fmt.Sprintf("  %.0f%% completion rate\n", m.overview.OverallRate)

	return ui.TitledPanel("Stats", s, width, height)
}
