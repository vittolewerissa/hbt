package habits

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/category"
	"github.com/vittolewerissa/hbt/internal/shared/db"
	"github.com/vittolewerissa/hbt/internal/shared/model"
	"github.com/vittolewerissa/hbt/internal/shared/ui"
)

// View modes
type viewMode int

const (
	modeList viewMode = iota
	modeForm
	modeConfirmDelete
)

// Model is the habits tab model
type Model struct {
	service     *Service
	catService  *category.Service
	habits      []model.Habit
	categories  []model.Category
	cursor      int
	mode        viewMode
	form        *FormModel
	width       int
	height      int
	keys        ui.KeyMap
	err         error
}

// New creates a new habits model
func New(database *db.DB) Model {
	return Model{
		service:    NewService(database),
		catService: category.NewService(database),
		keys:       ui.DefaultKeyMap,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadData
}

// HabitsLoadedMsg is sent when habits are loaded
type HabitsLoadedMsg struct {
	Habits     []model.Habit
	Categories []model.Category
	Err        error
}

// HabitSavedMsg is sent when a habit is saved
type HabitSavedMsg struct {
	Habit *model.Habit
	Err   error
}

// HabitDeletedMsg is sent when a habit is deleted
type HabitDeletedMsg struct {
	Err error
}

func (m Model) loadData() tea.Msg {
	habits, err := m.service.List()
	if err != nil {
		return HabitsLoadedMsg{Err: err}
	}
	categories, err := m.catService.List()
	if err != nil {
		return HabitsLoadedMsg{Err: err}
	}
	return HabitsLoadedMsg{Habits: habits, Categories: categories}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HabitsLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.habits = msg.Habits
		m.categories = msg.Categories
		return m, nil

	case HabitSavedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.mode = modeList
		m.form = nil
		return m, m.loadData

	case HabitDeletedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.mode = modeList
		if m.cursor >= len(m.habits)-1 && m.cursor > 0 {
			m.cursor--
		}
		return m, m.loadData

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.form != nil {
			m.form.width = msg.Width
			m.form.height = msg.Height
		}

	case tea.KeyMsg:
		// Update form first if active, so it receives key events
		if m.mode == modeForm && m.form != nil {
			var cmd tea.Cmd
			*m.form, cmd = m.form.Update(msg)
			// Check if form state changed
			if m.form.cancelled {
				m.mode = modeList
				m.form = nil
				return m, nil
			} else if m.form.submitted {
				return m, m.saveHabit(m.form.GetHabit())
			}
			return m, cmd
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.mode {
	case modeList:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.habits)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Add):
			m.form = NewForm(nil, m.categories, m.width, m.height)
			m.mode = modeForm
			return m, m.form.Init()
		case key.Matches(msg, m.keys.Edit):
			if len(m.habits) > 0 {
				habit := m.habits[m.cursor]
				m.form = NewForm(&habit, m.categories, m.width, m.height)
				m.mode = modeForm
				return m, m.form.Init()
			}
		case key.Matches(msg, m.keys.Delete):
			if len(m.habits) > 0 {
				m.mode = modeConfirmDelete
			}
		}

	case modeForm:
		if m.form != nil {
			if m.form.cancelled {
				m.mode = modeList
				m.form = nil
			} else if m.form.submitted {
				return m, m.saveHabit(m.form.GetHabit())
			}
		}

	case modeConfirmDelete:
		switch {
		case key.Matches(msg, m.keys.Confirm):
			if len(m.habits) > 0 {
				return m, m.deleteHabit(m.habits[m.cursor].ID)
			}
		case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Back):
			m.mode = modeList
		}
	}

	return m, nil
}

func (m Model) saveHabit(h *model.Habit) tea.Cmd {
	return func() tea.Msg {
		var err error
		if h.ID == 0 {
			err = m.service.Create(h)
		} else {
			err = m.service.Update(h)
		}
		return HabitSavedMsg{Habit: h, Err: err}
	}
}

func (m Model) deleteHabit(id int64) tea.Cmd {
	return func() tea.Msg {
		err := m.service.Archive(id)
		return HabitDeletedMsg{Err: err}
	}
}

// View renders the habits tab (with title)
func (m Model) View() string {
	if m.err != nil {
		return ui.MutedText.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.mode {
	case modeForm:
		if m.form != nil {
			return m.form.View()
		}
	case modeConfirmDelete:
		return m.renderConfirmDelete()
	}

	return m.renderList()
}

// ViewContent renders just the content without title (for titled panels)
func (m Model) ViewContent() string {
	if m.err != nil {
		return ui.MutedText.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.mode {
	case modeForm:
		if m.form != nil {
			return m.form.ViewContent()
		}
	case modeConfirmDelete:
		return m.renderConfirmDeleteContent()
	}

	return m.renderListContent()
}

func (m Model) renderList() string {
	var s string
	s += ui.Title.Render("Manage Habits") + "\n\n"
	s += m.renderListContent()
	return s
}

func (m Model) renderListContent() string {
	var s string

	if len(m.habits) == 0 {
		s += ui.MutedText.Render("No habits yet. Press 'a' to add one.")
		return s
	}

	for i, habit := range m.habits {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		name := habit.Name
		if i == m.cursor {
			name = ui.SelectedItem.Render(name)
		} else {
			name = ui.NormalItem.Render(name)
		}

		freq := m.formatFrequency(habit)

		line := fmt.Sprintf("%s%s %s", cursor, name, ui.MutedText.Render(freq))

		if habit.Category != nil {
			line += " " + ui.CategoryTag(habit.Category.Name, habit.Category.Color)
		}

		s += line + "\n"
	}

	s += "\n" + ui.MutedText.Render("a: add  e: edit  d: delete")

	return s
}

func (m Model) formatFrequency(h model.Habit) string {
	switch h.FrequencyType {
	case model.FreqDaily:
		return "(daily)"
	case model.FreqWeekly:
		return "(weekly)"
	case model.FreqTimesPerWeek:
		return fmt.Sprintf("(%dx/week)", h.FrequencyValue)
	default:
		return ""
	}
}

func (m Model) renderConfirmDelete() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		ui.Title.Render("Delete Habit"),
		"",
		m.renderConfirmDeleteContent(),
	)
}

func (m Model) renderConfirmDeleteContent() string {
	habit := m.habits[m.cursor]
	return lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("Are you sure you want to delete '%s'?", habit.Name),
		"",
		ui.MutedText.Render("y: confirm  n: cancel"),
	)
}

// Focused returns whether this view should receive key events
func (m Model) Focused() bool {
	return m.mode == modeForm
}
