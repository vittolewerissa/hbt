package habits

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/shared/model"
	"github.com/vittolewerissa/hbt/internal/shared/ui"
)

// FormField represents a form field
type FormField int

const (
	fieldName FormField = iota
	fieldDescription
	fieldEmoji
	fieldFrequency
	fieldFrequencyValue
	fieldTargetPerDay
	fieldCategory
)

// FormModel handles habit creation/editing
type FormModel struct {
	habit              *model.Habit
	categories         []model.Category
	nameInput          textinput.Model
	descInput          textinput.Model
	emojiSearch        textinput.Model
	freqValueInput     textinput.Model
	targetPerDayInput  textinput.Model
	focusedField       FormField
	frequencyType      model.FrequencyType
	selectedEmoji      string
	emojiIndex         int
	scrollOffset       int
	showEmojiModal     bool
	categoryIndex      int // -1 for no category
	showCategoryModal  bool
	categoryModalIndex int // Index within modal
	width              int
	height             int
	submitted          bool
	cancelled          bool
	isEdit             bool
}

// NewForm creates a new form model
func NewForm(habit *model.Habit, categories []model.Category, width, height int) *FormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Habit name"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 40

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"
	descInput.CharLimit = 200
	descInput.Width = 40

	freqValueInput := textinput.New()
	freqValueInput.Placeholder = "Times per week"
	freqValueInput.CharLimit = 2
	freqValueInput.Width = 10

	targetPerDayInput := textinput.New()
	targetPerDayInput.Placeholder = "Target per day"
	targetPerDayInput.CharLimit = 2
	targetPerDayInput.Width = 10

	emojiSearch := textinput.New()
	emojiSearch.Placeholder = "Search emojis..."
	emojiSearch.CharLimit = 50
	emojiSearch.Width = 30

	m := &FormModel{
		categories:        categories,
		nameInput:         nameInput,
		descInput:         descInput,
		emojiSearch:       emojiSearch,
		freqValueInput:    freqValueInput,
		targetPerDayInput: targetPerDayInput,
		frequencyType:     model.FreqDaily,
		categoryIndex:     -1,
		width:             width,
		height:            height,
	}

	if habit != nil {
		m.habit = habit
		m.isEdit = true
		m.nameInput.SetValue(habit.Name)
		m.descInput.SetValue(habit.Description)
		m.selectedEmoji = habit.Emoji
		m.frequencyType = habit.FrequencyType
		m.freqValueInput.SetValue(fmt.Sprintf("%d", habit.FrequencyValue))
		if habit.TargetPerDay > 0 {
			m.targetPerDayInput.SetValue(fmt.Sprintf("%d", habit.TargetPerDay))
		} else {
			m.targetPerDayInput.SetValue("1")
		}

		if habit.CategoryID != nil {
			for i, c := range categories {
				if c.ID == *habit.CategoryID {
					m.categoryIndex = i
					break
				}
			}
		}
	} else {
		m.habit = &model.Habit{}
		m.freqValueInput.SetValue("3")
		m.targetPerDayInput.SetValue("1")
	}

	return m
}

// Init initializes the form
func (m *FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles form messages
func (m *FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle emoji modal if open
		if m.showEmojiModal {
			var cmd tea.Cmd
			switch msg.String() {
			case "esc":
				m.showEmojiModal = false
				m.emojiSearch.SetValue("")
				m.emojiIndex = 0
				m.scrollOffset = 0
				return *m, nil
			case "enter":
				// Select current emoji or clear it
				if m.emojiIndex == -1 {
					m.selectedEmoji = ""
				} else {
					filtered := m.getFilteredEmojis()
					if len(filtered) > 0 && m.emojiIndex < len(filtered) {
						m.selectedEmoji = filtered[m.emojiIndex]
					}
				}
				m.showEmojiModal = false
				m.emojiSearch.SetValue("")
				m.emojiIndex = 0
				m.scrollOffset = 0
				return *m, nil
			case "left":
				m.emojiIndex--
				if m.emojiIndex < -1 {
					filtered := m.getFilteredEmojis()
					m.emojiIndex = len(filtered) - 1
				}
				m.ensureVisibleEmoji()
			case "right":
				filtered := m.getFilteredEmojis()
				m.emojiIndex++
				if m.emojiIndex >= len(filtered) {
					m.emojiIndex = -1
				}
				m.ensureVisibleEmoji()
			case "up":
				if m.emojiIndex == -1 {
					return *m, nil
				} else if m.emojiIndex < 8 {
					m.emojiIndex = -1
					m.ensureVisibleEmoji()
				} else {
					m.emojiIndex -= 8
					m.ensureVisibleEmoji()
				}
			case "down":
				filtered := m.getFilteredEmojis()
				if m.emojiIndex == -1 {
					m.emojiIndex = 0
					m.ensureVisibleEmoji()
				} else if m.emojiIndex+8 < len(filtered) {
					m.emojiIndex += 8
					m.ensureVisibleEmoji()
				}
			case "pgup":
				const maxVisibleRows = 8
				m.emojiIndex -= 8 * maxVisibleRows
				if m.emojiIndex < -1 {
					m.emojiIndex = -1
				}
				m.ensureVisibleEmoji()
			case "pgdown":
				const maxVisibleRows = 8
				filtered := m.getFilteredEmojis()
				m.emojiIndex += 8 * maxVisibleRows
				if m.emojiIndex >= len(filtered) {
					m.emojiIndex = len(filtered) - 1
				}
				m.ensureVisibleEmoji()
			default:
				m.emojiSearch, cmd = m.emojiSearch.Update(msg)
				m.emojiIndex = -1
				m.scrollOffset = 0
				return *m, cmd
			}
			return *m, nil
		}

		// Handle category modal if open
		if m.showCategoryModal {
			switch msg.String() {
			case "esc":
				m.showCategoryModal = false
				return *m, nil
			case "enter":
				// Select current category
				m.categoryIndex = m.categoryModalIndex
				m.showCategoryModal = false
				return *m, nil
			case "up":
				m.categoryModalIndex--
				if m.categoryModalIndex < -1 {
					m.categoryModalIndex = len(m.categories) - 1
				}
			case "down":
				m.categoryModalIndex++
				if m.categoryModalIndex >= len(m.categories) {
					m.categoryModalIndex = -1
				}
			}
			return *m, nil
		}

		switch msg.String() {
		case "esc":
			m.cancelled = true
			return *m, nil

		case "enter":
			if m.focusedField == fieldEmoji {
				// Open emoji modal
				m.showEmojiModal = true
				m.emojiSearch.Focus()
				if m.selectedEmoji == "" {
					m.emojiIndex = -1
				} else {
					m.emojiIndex = 0
				}
				return *m, textinput.Blink
			} else if m.focusedField == fieldCategory {
				// Open category modal
				if len(m.categories) > 0 {
					m.showCategoryModal = true
					m.categoryModalIndex = m.categoryIndex
					return *m, nil
				}
				// If no categories, submit form
				if m.nameInput.Value() != "" {
					m.submitted = true
				}
			} else {
				m.nextField()
			}
			return *m, nil

		case " ":
			// Space on emoji field opens modal
			if m.focusedField == fieldEmoji {
				m.showEmojiModal = true
				m.emojiSearch.Focus()
				if m.selectedEmoji == "" {
					m.emojiIndex = -1
				} else {
					m.emojiIndex = 0
				}
				return *m, textinput.Blink
			}
			// Space on category field opens modal
			if m.focusedField == fieldCategory && len(m.categories) > 0 {
				m.showCategoryModal = true
				m.categoryModalIndex = m.categoryIndex
				return *m, nil
			}
			// Don't consume space for text inputs
			// Fall through

		case "tab", "down":
			m.nextField()
			return *m, nil

		case "shift+tab", "up":
			m.prevField()
			return *m, nil

		case "left":
			if m.focusedField == fieldFrequency {
				m.prevFrequency()
				return *m, nil
			}
			// Fall through to let text inputs handle left arrow

		case "right":
			if m.focusedField == fieldFrequency {
				m.nextFrequency()
				return *m, nil
			}
			// Fall through to let text inputs handle right arrow

		case "backspace", "delete":
			// Clear emoji if on emoji field
			if m.focusedField == fieldEmoji && m.selectedEmoji != "" {
				m.selectedEmoji = ""
				return *m, nil
			}
			// Clear category if on category field
			if m.focusedField == fieldCategory && m.categoryIndex >= 0 {
				m.categoryIndex = -1
				return *m, nil
			}

		case "ctrl+s":
			// Ctrl+S to save from anywhere
			if m.nameInput.Value() != "" {
				m.submitted = true
				return *m, nil
			}
		}
	}

	// Update focused input
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case fieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case fieldFrequencyValue:
		m.freqValueInput, cmd = m.freqValueInput.Update(msg)
	case fieldTargetPerDay:
		m.targetPerDayInput, cmd = m.targetPerDayInput.Update(msg)
	}

	return *m, cmd
}

func (m *FormModel) nextField() {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.freqValueInput.Blur()
	m.targetPerDayInput.Blur()

	m.focusedField++
	if m.focusedField == fieldFrequencyValue && m.frequencyType != model.FreqTimesPerWeek {
		m.focusedField++ // Skip frequency value for daily/weekly
	}
	if m.focusedField > fieldCategory {
		m.focusedField = fieldCategory
	}

	switch m.focusedField {
	case fieldName:
		m.nameInput.Focus()
	case fieldDescription:
		m.descInput.Focus()
	case fieldFrequencyValue:
		m.freqValueInput.Focus()
	case fieldTargetPerDay:
		m.targetPerDayInput.Focus()
	}
}

func (m *FormModel) prevField() {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.freqValueInput.Blur()
	m.targetPerDayInput.Blur()

	m.focusedField--
	if m.focusedField == fieldFrequencyValue && m.frequencyType != model.FreqTimesPerWeek {
		m.focusedField-- // Skip frequency value for daily/weekly
	}
	if m.focusedField < fieldName {
		m.focusedField = fieldName
	}

	switch m.focusedField {
	case fieldName:
		m.nameInput.Focus()
	case fieldDescription:
		m.descInput.Focus()
	case fieldFrequencyValue:
		m.freqValueInput.Focus()
	case fieldTargetPerDay:
		m.targetPerDayInput.Focus()
	}
}

func (m *FormModel) nextFrequency() {
	switch m.frequencyType {
	case model.FreqDaily:
		m.frequencyType = model.FreqWeekly
	case model.FreqWeekly:
		m.frequencyType = model.FreqTimesPerWeek
	case model.FreqTimesPerWeek:
		m.frequencyType = model.FreqDaily
	}
}

func (m *FormModel) prevFrequency() {
	switch m.frequencyType {
	case model.FreqDaily:
		m.frequencyType = model.FreqTimesPerWeek
	case model.FreqWeekly:
		m.frequencyType = model.FreqDaily
	case model.FreqTimesPerWeek:
		m.frequencyType = model.FreqWeekly
	}
}


// GetHabit returns the habit with form values
func (m *FormModel) GetHabit() *model.Habit {
	m.habit.Name = m.nameInput.Value()
	m.habit.Description = m.descInput.Value()
	m.habit.Emoji = m.selectedEmoji
	m.habit.FrequencyType = m.frequencyType

	if m.frequencyType == model.FreqTimesPerWeek {
		var val int
		fmt.Sscanf(m.freqValueInput.Value(), "%d", &val)
		if val < 1 {
			val = 1
		}
		if val > 7 {
			val = 7
		}
		m.habit.FrequencyValue = val
	} else {
		m.habit.FrequencyValue = 1
	}

	// Parse target per day
	var targetPerDay int
	fmt.Sscanf(m.targetPerDayInput.Value(), "%d", &targetPerDay)
	if targetPerDay < 1 {
		targetPerDay = 1
	}
	if targetPerDay > 99 {
		targetPerDay = 99
	}
	m.habit.TargetPerDay = targetPerDay

	if m.categoryIndex >= 0 && m.categoryIndex < len(m.categories) {
		id := m.categories[m.categoryIndex].ID
		m.habit.CategoryID = &id
	} else {
		m.habit.CategoryID = nil
	}

	return m.habit
}

// View renders the form (with title)
func (m *FormModel) View() string {
	title := "Add Habit"
	if m.isEdit {
		title = "Edit Habit"
	}

	var s string
	s += ui.Title.Render(title) + "\n\n"
	s += m.ViewContent()
	return s
}

// ViewContent renders just the form fields without title
func (m *FormModel) ViewContent() string {
	var s string

	// Name field
	s += m.renderField("Name", m.nameInput.View(), m.focusedField == fieldName)

	// Description field
	s += m.renderField("Description", m.descInput.View(), m.focusedField == fieldDescription)

	// Emoji field
	emojiDisplay := m.renderEmojiSelector(m.focusedField == fieldEmoji)
	s += m.renderField("Emoji", emojiDisplay, m.focusedField == fieldEmoji)

	// Frequency field
	freqDisplay := m.renderFrequencySelector()
	s += m.renderField("Frequency", freqDisplay, m.focusedField == fieldFrequency)

	// Frequency value (only for times per week)
	if m.frequencyType == model.FreqTimesPerWeek {
		s += m.renderField("Times/Week", m.freqValueInput.View(), m.focusedField == fieldFrequencyValue)
	}

	// Target per day field
	s += m.renderField("Target/Day", m.targetPerDayInput.View(), m.focusedField == fieldTargetPerDay)

	// Category field
	catDisplay := m.renderCategorySelector(m.focusedField == fieldCategory)
	s += m.renderField("Category", catDisplay, m.focusedField == fieldCategory)

	s += "\n" + ui.MutedText.Render("tab: navigate  enter/space: pick category  backspace: clear")
	s += "\n"
	s += ui.MutedText.Render("ctrl+s: save  esc: cancel")

	return s
}

func (m *FormModel) renderField(label, value string, focused bool) string {
	labelStyle := ui.FormLabel
	if focused {
		labelStyle = labelStyle.Copy().Foreground(ui.Primary)
	}
	return labelStyle.Render(fmt.Sprintf("%-12s", label)) + value + "\n"
}

func (m *FormModel) renderFrequencySelector() string {
	options := []struct {
		freq model.FrequencyType
		name string
	}{
		{model.FreqDaily, "Daily"},
		{model.FreqWeekly, "Weekly"},
		{model.FreqTimesPerWeek, "X/Week"},
	}

	var parts []string
	for _, opt := range options {
		style := ui.MutedText
		if opt.freq == m.frequencyType {
			style = lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
		}
		parts = append(parts, style.Render(opt.name))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts[0], " | ", parts[1], " | ", parts[2])
}

func (m *FormModel) renderCategorySelector(focused bool) string {
	if len(m.categories) == 0 {
		return ui.MutedText.Render("No categories")
	}

	var display string
	if m.categoryIndex < 0 {
		display = "(none)"
		if !focused {
			return ui.MutedText.Render(display)
		}
	} else {
		cat := m.categories[m.categoryIndex]
		if cat.Emoji != "" {
			display = cat.Emoji + " " + cat.Name
		} else {
			display = cat.Name
		}
		if !focused {
			return display
		}
	}

	// When focused, show with brackets
	return ui.SelectedItem.Render("[" + display + "]")
}

// renderCategoryModalBox renders the category picker modal content
func (m *FormModel) renderCategoryModalBox() string {
	var s string

	// Modal title
	s += ui.Title.Render("Pick a Category") + "\n\n"

	// Show categories list
	if len(m.categories) == 0 {
		s += ui.MutedText.Render("No categories available") + "\n"
	} else {
		// None option
		noneText := "(none)"
		if m.categoryModalIndex == -1 {
			s += "[" + noneText + "]" + "\n"
		} else {
			s += " " + noneText + " " + "\n"
		}
		s += "\n"

		// Category options
		for i, cat := range m.categories {
			catText := cat.Name
			if cat.Emoji != "" {
				catText = cat.Emoji + " " + catText
			}

			if i == m.categoryModalIndex {
				s += "[" + catText + "]" + "\n"
			} else {
				s += " " + catText + " " + "\n"
			}
		}
	}
	s += "\n"

	s += ui.MutedText.Render("↑↓: navigate  enter: select  esc: cancel")

	// Add modal box styling with dark background
	modalWidth := 40
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Background(lipgloss.Color("#1F2937")).
		Padding(1, 2).
		Width(modalWidth)

	return modalStyle.Render(s)
}
