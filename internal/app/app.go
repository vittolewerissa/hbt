package app

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/habits"
	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/ui"
	"github.com/vittolewerissa/hbt/internal/stats"
	"github.com/vittolewerissa/hbt/internal/today"
)

// Tab represents a tab in the application
type Tab int

const (
	TabToday Tab = iota
	TabHabits
)

const numTabs = 2

var tabNames = []string{"Today", "Habits"}

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
	todayModel  today.Model
	habitsModel habits.Model
	statsModel  stats.Model
}

// New creates a new application model
func New(database *db.DB) Model {
	return Model{
		db:          database,
		keys:        ui.DefaultKeyMap,
		help:        help.New(),
		activeTab:   TabToday,
		todayModel:  today.New(database),
		habitsModel: habits.New(database),
		statsModel:  stats.New(database),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.todayModel.Init(),
		m.habitsModel.Init(),
		m.statsModel.Init(),
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
			m.activeTab = (m.activeTab + 1) % numTabs
			cmds = append(cmds, m.reloadTabData(oldTab)...)
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.PrevTab):
			oldTab := m.activeTab
			m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
			cmds = append(cmds, m.reloadTabData(oldTab)...)
			return m, tea.Batch(cmds...)
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
	}

	// Route today messages regardless of active tab
	switch msg.(type) {
	case today.TodayLoadedMsg, today.CompletionToggledMsg:
		var cmd tea.Cmd
		m.todayModel, cmd = m.todayModel.Update(msg)
		cmds = append(cmds, cmd)
		// Also reload stats when completion changes
		cmds = append(cmds, m.statsModel.Init())
	}

	// Route habit messages regardless of active tab
	switch msg.(type) {
	case habits.HabitsLoadedMsg, habits.HabitSavedMsg, habits.HabitDeletedMsg:
		var cmd tea.Cmd
		m.habitsModel, cmd = m.habitsModel.Update(msg)
		cmds = append(cmds, cmd)
		// Also reload stats when habits change
		cmds = append(cmds, m.statsModel.Init())
	}

	// Route stats messages regardless of active tab
	switch msg.(type) {
	case stats.StatsLoadedMsg:
		var cmd tea.Cmd
		m.statsModel, cmd = m.statsModel.Update(msg)
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
	return cmds
}

// View renders the application
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Render main content with stats panel (includes tab bar in left column)
	content := m.renderMainContent()

	// Render help bar
	helpView := m.help.View(m.keys)

	// Combine everything with explicit top padding via newlines
	return "\n\n" + lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		PaddingBottom(1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				content,
				helpView,
			),
		)
}

func (m Model) renderTabBar(width int) string {
	var tabs []string
	for i, name := range tabNames {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, ui.ActiveTab.Render(name))
		} else {
			tabs = append(tabs, ui.InactiveTab.Render(name))
		}
	}
	tabContent := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return ui.TabBar.Width(width).Render(tabContent)
}

func (m Model) renderMainContent() string {
	// Calculate dimensions
	contentHeight := m.height - 10 // Account for tab bar, help, padding
	totalWidth := m.width - 4

	// 60/40 split
	leftWidth := int(float64(totalWidth) * 0.6)
	rightWidth := totalWidth - leftWidth - 1 // -1 for gap

	// Render tab bar for left column only
	tabBar := m.renderTabBar(leftWidth)

	// Render left panel (tab content)
	var leftContent string
	var panelTitle string
	switch m.activeTab {
	case TabToday:
		leftContent = m.todayModel.ViewContent()
		panelTitle = "Today"
	case TabHabits:
		leftContent = m.habitsModel.ViewContent()
		panelTitle = "Habits"
	}

	leftPanel := ui.TitledPanel(panelTitle, leftContent, leftWidth, contentHeight)

	// Combine tab bar and left panel
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, tabBar, leftPanel)

	// Render right panel (stats) - add extra height for tab bar
	rightPanel := m.statsModel.RenderPanel(rightWidth, contentHeight+3)

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, " ", rightPanel)
}
