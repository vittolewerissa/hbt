package app

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vl/habit-cli/internal/habits"
	"github.com/vl/habit-cli/internal/settings"
	"github.com/vl/habit-cli/internal/shared/db"
	"github.com/vl/habit-cli/internal/shared/ui"
	"github.com/vl/habit-cli/internal/stats"
	"github.com/vl/habit-cli/internal/today"
)

// Tab represents a tab in the application
type Tab int

const (
	TabToday Tab = iota
	TabHabits
	TabStats
	TabSettings
)

var tabNames = []string{"Today", "Habits", "Stats", "Settings"}

// Model is the main application model
type Model struct {
	db        *db.DB
	keys      ui.KeyMap
	help      help.Model
	activeTab Tab
	width     int
	height    int
	ready     bool

	// Tab models
	todayModel    today.Model
	habitsModel   habits.Model
	statsModel    stats.Model
	settingsModel settings.Model
}

// New creates a new application model
func New(database *db.DB, dbPath string) Model {
	return Model{
		db:            database,
		keys:          ui.DefaultKeyMap,
		help:          help.New(),
		activeTab:     TabToday,
		todayModel:    today.New(database),
		habitsModel:   habits.New(database),
		statsModel:    stats.New(database),
		settingsModel: settings.New(database, dbPath),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.todayModel.Init(),
		m.habitsModel.Init(),
		m.statsModel.Init(),
		m.settingsModel.Init(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If habits form is focused, let it handle all keys
		if m.activeTab == TabHabits && m.habitsModel.Focused() {
			var cmd tea.Cmd
			m.habitsModel, cmd = m.habitsModel.Update(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.NextTab):
			oldTab := m.activeTab
			m.activeTab = (m.activeTab + 1) % 4
			cmds = append(cmds, m.reloadTabData(oldTab)...)
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.PrevTab):
			oldTab := m.activeTab
			m.activeTab = (m.activeTab - 1 + 4) % 4
			cmds = append(cmds, m.reloadTabData(oldTab)...)
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.ready = true
	}

	// Route messages to active tab
	switch m.activeTab {
	case TabToday:
		var cmd tea.Cmd
		m.todayModel, cmd = m.todayModel.Update(msg)
		cmds = append(cmds, cmd)
	case TabHabits:
		var cmd tea.Cmd
		m.habitsModel, cmd = m.habitsModel.Update(msg)
		cmds = append(cmds, cmd)
	case TabStats:
		var cmd tea.Cmd
		m.statsModel, cmd = m.statsModel.Update(msg)
		cmds = append(cmds, cmd)
	case TabSettings:
		var cmd tea.Cmd
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Route today messages regardless of active tab
	switch msg.(type) {
	case today.TodayLoadedMsg, today.CompletionToggledMsg:
		var cmd tea.Cmd
		m.todayModel, cmd = m.todayModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Route habit messages regardless of active tab
	switch msg.(type) {
	case habits.HabitsLoadedMsg, habits.HabitSavedMsg, habits.HabitDeletedMsg:
		var cmd tea.Cmd
		m.habitsModel, cmd = m.habitsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Route stats messages regardless of active tab
	switch msg.(type) {
	case stats.StatsLoadedMsg:
		var cmd tea.Cmd
		m.statsModel, cmd = m.statsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Route settings messages regardless of active tab
	switch msg.(type) {
	case settings.SettingsLoadedMsg:
		var cmd tea.Cmd
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// reloadTabData returns commands to reload data when switching tabs
func (m Model) reloadTabData(oldTab Tab) []tea.Cmd {
	var cmds []tea.Cmd
	if m.activeTab == TabToday && oldTab != TabToday {
		cmds = append(cmds, m.todayModel.Init())
	}
	if m.activeTab == TabStats && oldTab != TabStats {
		cmds = append(cmds, m.statsModel.Init())
	}
	return cmds
}

// View renders the application
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Render tab bar
	tabBar := m.renderTabBar()

	// Render current tab content
	content := m.renderTabContent()

	// Render help
	helpView := m.help.View(m.keys)

	// Combine everything
	return ui.Container.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			tabBar,
			content,
			helpView,
		),
	)
}

func (m Model) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, ui.ActiveTab.Render(name))
		} else {
			tabs = append(tabs, ui.InactiveTab.Render(name))
		}
	}
	return ui.TabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}

func (m Model) renderTabContent() string {
	// Calculate available height for content
	contentHeight := m.height - 8 // Account for tab bar, help, padding

	contentStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		Height(contentHeight)

	var content string
	switch m.activeTab {
	case TabToday:
		content = m.todayModel.View()
	case TabHabits:
		content = m.habitsModel.View()
	case TabStats:
		content = m.statsModel.View()
	case TabSettings:
		content = m.settingsModel.View()
	}

	return contentStyle.Render(content)
}
