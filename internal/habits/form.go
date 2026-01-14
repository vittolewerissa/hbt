package habits

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/habit-cli/internal/shared/model"
	"github.com/vittolewerissa/habit-cli/internal/shared/ui"
)

// FormField represents a form field
type FormField int

const (
	fieldName FormField = iota
	fieldDescription
	fieldFrequency
	fieldFrequencyValue
	fieldCategory
)

// FormModel handles habit creation/editing
type FormModel struct {
	habit           *model.Habit
	categories      []model.Category
	nameInput       textinput.Model
	descInput       textinput.Model
	freqValueInput  textinput.Model
	focusedField    FormField
	frequencyType   model.FrequencyType
	categoryIndex   int // -1 for no category
	width           int
	height          int
	submitted       bool
	cancelled       bool
	isEdit          bool
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

	m := &FormModel{
		categories:     categories,
		nameInput:      nameInput,
		descInput:      descInput,
		freqValueInput: freqValueInput,
		frequencyType:  model.FreqDaily,
		categoryIndex:  -1,
		width:          width,
		height:         height,
	}

	if habit != nil {
		m.habit = habit
		m.isEdit = true
		m.nameInput.SetValue(habit.Name)
		m.descInput.SetValue(habit.Description)
		m.frequencyType = habit.FrequencyType
		m.freqValueInput.SetValue(fmt.Sprintf("%d", habit.FrequencyValue))

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
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return *m, nil

		case "enter":
			if m.focusedField == fieldCategory {
				// Submit form
				if m.nameInput.Value() != "" {
					m.submitted = true
				}
			} else {
				m.nextField()
			}
			return *m, nil

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
			} else if m.focusedField == fieldCategory {
				m.prevCategory()
				return *m, nil
			}
			// Fall through to let text inputs handle left arrow

		case "right":
			if m.focusedField == fieldFrequency {
				m.nextFrequency()
				return *m, nil
			} else if m.focusedField == fieldCategory {
				m.nextCategory()
				return *m, nil
			}
			// Fall through to let text inputs handle right arrow
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
	}

	return *m, cmd
}

func (m *FormModel) nextField() {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.freqValueInput.Blur()

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
	}
}

func (m *FormModel) prevField() {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.freqValueInput.Blur()

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

func (m *FormModel) nextCategory() {
	m.categoryIndex++
	if m.categoryIndex >= len(m.categories) {
		m.categoryIndex = -1
	}
}

func (m *FormModel) prevCategory() {
	m.categoryIndex--
	if m.categoryIndex < -1 {
		m.categoryIndex = len(m.categories) - 1
	}
}

// GetHabit returns the habit with form values
func (m *FormModel) GetHabit() *model.Habit {
	m.habit.Name = m.nameInput.Value()
	m.habit.Description = m.descInput.Value()
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

	// Frequency field
	freqDisplay := m.renderFrequencySelector()
	s += m.renderField("Frequency", freqDisplay, m.focusedField == fieldFrequency)

	// Frequency value (only for times per week)
	if m.frequencyType == model.FreqTimesPerWeek {
		s += m.renderField("Times/Week", m.freqValueInput.View(), m.focusedField == fieldFrequencyValue)
	}

	// Category field
	catDisplay := m.renderCategorySelector()
	s += m.renderField("Category", catDisplay, m.focusedField == fieldCategory)

	s += "\n" + ui.MutedText.Render("tab/arrows: navigate  enter: submit  esc: cancel")

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

func (m *FormModel) renderCategorySelector() string {
	if len(m.categories) == 0 {
		return ui.MutedText.Render("No categories")
	}

	if m.categoryIndex < 0 {
		return ui.MutedText.Render("< None >")
	}

	cat := m.categories[m.categoryIndex]
	return "< " + ui.CategoryTag(cat.Name, cat.Color) + " >"
}
