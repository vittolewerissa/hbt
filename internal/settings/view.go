package settings

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/ui"
)

// Model is the settings tab model
type Model struct {
	service  *Service
	dbPath   string
	settings map[string]string
	width    int
	height   int
	keys     ui.KeyMap
	err      error
}

// New creates a new settings model
func New(database *db.DB, dbPath string) Model {
	return Model{
		service:  NewService(database),
		dbPath:   dbPath,
		settings: make(map[string]string),
		keys:     ui.DefaultKeyMap,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadSettings
}

// SettingsLoadedMsg is sent when settings are loaded
type SettingsLoadedMsg struct {
	Settings map[string]string
	Err      error
}

func (m Model) loadSettings() tea.Msg {
	settings, err := m.service.GetAll()
	return SettingsLoadedMsg{Settings: settings, Err: err}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SettingsLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.settings = msg.Settings
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the settings tab
func (m Model) View() string {
	if m.err != nil {
		return ui.MutedText.Render(fmt.Sprintf("Error: %v", m.err))
	}

	var s string
	s += ui.Title.Render("Settings") + "\n\n"

	// Info section
	infoStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ui.Border).
		Padding(1, 2).
		Width(m.width - 10)

	var info string
	info += lipgloss.NewStyle().Bold(true).Render("Application Info") + "\n\n"
	info += fmt.Sprintf("Database: %s\n", m.dbPath)
	info += fmt.Sprintf("Version: 0.1.0\n")

	s += infoStyle.Render(info) + "\n\n"

	// Keyboard shortcuts
	s += lipgloss.NewStyle().Bold(true).Render("Keyboard Shortcuts") + "\n\n"
	s += ui.MutedText.Render("  Tab/Shift+Tab  ") + "Switch tabs\n"
	s += ui.MutedText.Render("  j/k or ↑/↓     ") + "Navigate list\n"
	s += ui.MutedText.Render("  Space/Enter    ") + "Toggle/Select\n"
	s += ui.MutedText.Render("  a              ") + "Add new habit\n"
	s += ui.MutedText.Render("  e              ") + "Edit selected\n"
	s += ui.MutedText.Render("  d              ") + "Delete selected\n"
	s += ui.MutedText.Render("  ?              ") + "Toggle help\n"
	s += ui.MutedText.Render("  q              ") + "Quit\n"

	return s
}

// Focused returns whether this view should receive key events
func (m Model) Focused() bool {
	return false
}
