package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/category"
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
	TabCategories
)

const numTabs = 3

var tabNames = []string{"Today", "Habits", "Categories"}

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
	todayModel      today.Model
	habitsModel     habits.Model
	categoriesModel category.Model
	statsModel      stats.Model
}

// New creates a new application model
func New(database *db.DB) Model {
	return Model{
		db:              database,
		keys:            ui.DefaultKeyMap,
		help:            help.New(),
		activeTab:       TabToday,
		todayModel:      today.New(database),
		habitsModel:     habits.New(database),
		categoriesModel: category.New(database),
		statsModel:      stats.New(database),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.todayModel.Init(),
		m.habitsModel.Init(),
		m.categoriesModel.Init(),
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

		// If categories form is focused, let it handle all keys
		if m.activeTab == TabCategories && m.categoriesModel.Focused() {
			var cmd tea.Cmd
			m.categoriesModel, cmd = m.categoriesModel.Update(msg)
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
	case TabCategories:
		var cmd tea.Cmd
		m.categoriesModel, cmd = m.categoriesModel.Update(msg)
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

	// Route category messages regardless of active tab
	switch msg.(type) {
	case category.CategoriesLoadedMsg, category.CategorySavedMsg, category.CategoryDeletedMsg:
		var cmd tea.Cmd
		m.categoriesModel, cmd = m.categoriesModel.Update(msg)
		cmds = append(cmds, cmd)
		// Also reload habits when categories change (in case they reference categories)
		cmds = append(cmds, m.habitsModel.Init())
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

	// Check if there's a modal to overlay
	if m.activeTab == TabCategories && m.categoriesModel.HasModal() {
		// Render the modal with transparent overlay showing background
		return m.renderWithModal()
	}
	if m.activeTab == TabHabits && m.habitsModel.HasModal() {
		// Render the modal with transparent overlay showing background
		return m.renderWithModal()
	}

	// Render main content with stats panel (includes tab bar in left column)
	content := m.renderMainContent()

	// Render help bar
	helpView := m.help.View(m.keys)

	// Combine everything with explicit top padding via newlines
	baseView := "\n\n" + lipgloss.NewStyle().
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

	return baseView
}

func (m Model) renderWithModal() string {
	// Render the base content (dimmed)
	content := m.renderMainContent()
	helpView := m.help.View(m.keys)

	// Dim the base content by applying muted colors
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")) // Darker gray

	dimmedContent := dimStyle.Render(content)
	dimmedHelp := dimStyle.Render(helpView)

	baseView := "\n\n" + lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		PaddingBottom(1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				dimmedContent,
				dimmedHelp,
			),
		)

	// Get the modal content based on active tab (just the box, no placement)
	var modalContent string
	if m.activeTab == TabCategories {
		modalContent = m.categoriesModel.RenderModalContent()
	} else if m.activeTab == TabHabits {
		modalContent = m.habitsModel.RenderModalContent()
	}

	// Overlay the modal box onto the base view
	return m.overlayModalOnBase(baseView, modalContent)
}

func (m Model) overlayModalOnBase(base, modal string) string {
	baseLines := strings.Split(base, "\n")
	modalLines := strings.Split(modal, "\n")

	// Calculate dimensions
	baseHeight := len(baseLines)
	modalHeight := len(modalLines)

	// Find the widest base line (for horizontal centering calculation)
	var baseWidth int
	for _, line := range baseLines {
		w := lipgloss.Width(line)
		if w > baseWidth {
			baseWidth = w
		}
	}

	// Find modal width
	var modalWidth int
	for _, line := range modalLines {
		w := lipgloss.Width(line)
		if w > modalWidth {
			modalWidth = w
		}
	}

	// Calculate centered position
	startRow := (baseHeight - modalHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (baseWidth - modalWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Start with a copy of base
	result := make([]string, baseHeight)
	copy(result, baseLines)

	// Overlay modal lines onto result
	for i, modalLine := range modalLines {
		targetRow := startRow + i
		if targetRow >= 0 && targetRow < baseHeight {
			baseLine := result[targetRow]

			// We need to insert modalLine at startCol visible position in baseLine
			// This requires ANSI-aware string manipulation

			// Get visible width of base line
			baseLineWidth := lipgloss.Width(baseLine)

			// Calculate how much of the base line to keep before and after modal
			// We'll split the base line into: [left part] [modal] [right part]

			// Build the new line by concatenating visible character ranges
			var newLine string

			// Add left part (0 to startCol visible chars)
			if startCol > 0 {
				newLine += m.getVisibleSubstring(baseLine, 0, startCol)
			}

			// Add modal content
			newLine += modalLine

			// Add right part (startCol+modalWidth to end)
			rightStart := startCol + lipgloss.Width(modalLine)
			if rightStart < baseLineWidth {
				newLine += m.getVisibleSubstring(baseLine, rightStart, baseLineWidth)
			}

			result[targetRow] = newLine
		}
	}

	return strings.Join(result, "\n")
}

// getVisibleSubstring extracts a substring based on visible character positions (ANSI-aware)
// from position start (inclusive) to end (exclusive) in visible character count
func (m Model) getVisibleSubstring(s string, start, end int) string {
	if start >= end {
		return ""
	}

	// Use lipgloss.Width to measure visible width as we iterate through runes
	// This is a simplified approach - it strips ANSI codes and works with visible chars

	// For a more robust solution, we'd need ansi-aware truncation
	// For now, use a simple approach: iterate through the string and track visible position

	var result strings.Builder
	visiblePos := 0
	inEscape := false

	for _, r := range s {
		// Track ANSI escape sequences
		if r == '\x1b' {
			inEscape = true
		}

		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		// Regular character
		if visiblePos >= start && visiblePos < end {
			result.WriteRune(r)
		}
		visiblePos++

		// If we've reached the end position, we can keep copying ANSI codes
		// but stop adding visible characters
		if visiblePos >= end {
			// Continue to preserve any remaining ANSI sequences that might reset styles
			if r == '\x1b' {
				inEscape = true
				result.WriteRune(r)
			}
		}
	}

	return result.String()
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
	case TabCategories:
		leftContent = m.categoriesModel.ViewContent()
		panelTitle = "Categories"
	}

	leftPanel := ui.TitledPanel(panelTitle, leftContent, leftWidth, contentHeight)

	// Combine tab bar and left panel
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, tabBar, leftPanel)

	// Render right panel (stats) - add extra height for tab bar
	rightPanel := m.statsModel.RenderPanel(rightWidth, contentHeight+3)

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, " ", rightPanel)
}
