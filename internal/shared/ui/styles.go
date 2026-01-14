package ui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	Primary     = lipgloss.Color("#7C3AED") // Purple
	Secondary   = lipgloss.Color("#06B6D4") // Cyan
	Success     = lipgloss.Color("#10B981") // Green
	Warning     = lipgloss.Color("#F59E0B") // Amber
	Danger      = lipgloss.Color("#EF4444") // Red
	Muted       = lipgloss.Color("#6B7280") // Gray
	Background  = lipgloss.Color("#1F2937") // Dark gray
	Foreground  = lipgloss.Color("#F9FAFB") // Light gray
	Border      = lipgloss.Color("#374151") // Medium gray
)

// Tab styles
var (
	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(Primary).
			Padding(0, 2)

	InactiveTab = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 2)

	TabBar = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(Border).
		MarginBottom(1)
)

// List styles
var (
	SelectedItem = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(Foreground)

	CompletedItem = lipgloss.NewStyle().
			Foreground(Success).
			Strikethrough(true)

	MutedText = lipgloss.NewStyle().
			Foreground(Muted)
)

// Form styles
var (
	FormLabel = lipgloss.NewStyle().
			Foreground(Muted).
			MarginRight(1)

	FormInput = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1)

	FormInputFocused = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)
)

// Status indicators
var (
	Checkbox = lipgloss.NewStyle().
			Foreground(Muted)

	CheckboxChecked = lipgloss.NewStyle().
				Foreground(Success)

	StreakBadge = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)
)

// Layout
var (
	Container = lipgloss.NewStyle().
			Padding(1, 2)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Foreground).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginBottom(1)

	HelpText = lipgloss.NewStyle().
			Foreground(Muted).
			MarginTop(1)
)

// CategoryTag returns a styled category tag with the given color
func CategoryTag(name, color string) string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(color)).
		Foreground(lipgloss.Color("#000000")).
		Padding(0, 1).
		Render(name)
}
