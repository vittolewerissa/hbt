package category

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// Model is the categories tab model
type Model struct {
	service    *Service
	categories []model.Category
	cursor     int
	mode       viewMode
	form       *FormModel
	width      int
	height     int
	keys       ui.KeyMap
	err        error
}

// FormModel handles category creation/editing
type FormModel struct {
	category       *model.Category
	nameInput      textinput.Model
	emojiSearch    textinput.Model
	selectedEmoji  string
	emojiIndex     int
	scrollOffset   int  // For scrolling through emoji list
	focusIndex     int  // 0: name, 1: emoji button
	showEmojiModal bool
	width          int
	height         int
	cancelled      bool
	submitted      bool
}

// New creates a new categories model
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

// CategoriesLoadedMsg is sent when categories are loaded
type CategoriesLoadedMsg struct {
	Categories []model.Category
	Err        error
}

// CategorySavedMsg is sent when a category is saved
type CategorySavedMsg struct {
	Category *model.Category
	Err      error
}

// CategoryDeletedMsg is sent when a category is deleted
type CategoryDeletedMsg struct {
	Err error
}

func (m Model) loadData() tea.Msg {
	categories, err := m.service.List()
	if err != nil {
		return CategoriesLoadedMsg{Err: err}
	}
	return CategoriesLoadedMsg{Categories: categories}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CategoriesLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.categories = msg.Categories
		return m, nil

	case CategorySavedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.mode = modeList
		m.form = nil
		return m, m.loadData

	case CategoryDeletedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.mode = modeList
		if m.cursor >= len(m.categories)-1 && m.cursor > 0 {
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
		// Update form first if active
		if m.mode == modeForm && m.form != nil {
			var cmd tea.Cmd
			*m.form, cmd = m.form.Update(msg)
			if m.form.cancelled {
				m.mode = modeList
				m.form = nil
				return m, nil
			} else if m.form.submitted {
				return m, m.saveCategory(m.form.GetCategory())
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
			if m.cursor < len(m.categories)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Add):
			m.form = NewCategoryForm(nil, m.width, m.height)
			m.mode = modeForm
			return m, m.form.Init()
		case key.Matches(msg, m.keys.Edit):
			if len(m.categories) > 0 {
				cat := m.categories[m.cursor]
				m.form = NewCategoryForm(&cat, m.width, m.height)
				m.mode = modeForm
				return m, m.form.Init()
			}
		case key.Matches(msg, m.keys.Delete):
			if len(m.categories) > 0 {
				m.mode = modeConfirmDelete
			}
		}

	case modeConfirmDelete:
		switch {
		case key.Matches(msg, m.keys.Confirm):
			if len(m.categories) > 0 {
				return m, m.deleteCategory(m.categories[m.cursor].ID)
			}
		case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Back):
			m.mode = modeList
		}
	}

	return m, nil
}

func (m Model) saveCategory(c *model.Category) tea.Cmd {
	return func() tea.Msg {
		var err error
		if c.ID == 0 {
			err = m.service.Create(c)
		} else {
			err = m.service.Update(c)
		}
		return CategorySavedMsg{Category: c, Err: err}
	}
}

func (m Model) deleteCategory(id int64) tea.Cmd {
	return func() tea.Msg {
		err := m.service.Delete(id)
		return CategoryDeletedMsg{Err: err}
	}
}

// ViewContent renders just the content without title
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

func (m Model) renderListContent() string {
	var s string

	if len(m.categories) == 0 {
		s += ui.MutedText.Render("No categories yet. Press 'a' to add one.")
		return s
	}

	for i, cat := range m.categories {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		name := cat.Name
		if i == m.cursor {
			name = ui.SelectedItem.Render(name)
		} else {
			name = ui.NormalItem.Render(name)
		}

		// Use custom emoji from category (optional)
		if cat.Emoji != "" {
			s += fmt.Sprintf("%s%s %s\n", cursor, cat.Emoji, name)
		} else {
			s += fmt.Sprintf("%s%s\n", cursor, name)
		}
	}

	s += "\n" + ui.MutedText.Render("a: add  e: edit  d: delete")

	return s
}

func (m Model) renderConfirmDeleteContent() string {
	cat := m.categories[m.cursor]
	return lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("Are you sure you want to delete '%s'?", cat.Name),
		"",
		ui.MutedText.Render("y: confirm  n: cancel"),
	)
}

// Focused returns whether this view should receive key events
func (m Model) Focused() bool {
	return m.mode == modeForm
}

// HasModal returns true if showing a modal dialog
func (m Model) HasModal() bool {
	return m.mode == modeForm && m.form != nil && m.form.showEmojiModal
}

// RenderModal renders the modal overlay (if any) centered on full screen
func (m Model) RenderModal(width, height int) string {
	if !m.HasModal() {
		return ""
	}
	return m.form.renderFullScreenModal(width, height)
}

// RenderModalContent renders just the modal box content for overlay
func (m Model) RenderModalContent() string {
	if !m.HasModal() {
		return ""
	}
	return m.form.renderModalBox()
}

// NewCategoryForm creates a new category form
func NewCategoryForm(c *model.Category, width, height int) *FormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Category name"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 30

	emojiSearch := textinput.New()
	emojiSearch.Placeholder = "Search emojis..."
	emojiSearch.CharLimit = 50
	emojiSearch.Width = 30

	selectedEmoji := "" // Empty by default (optional)
	if c != nil {
		nameInput.SetValue(c.Name)
		selectedEmoji = c.Emoji // Keep existing emoji if editing
	}

	return &FormModel{
		category:      c,
		nameInput:     nameInput,
		emojiSearch:   emojiSearch,
		selectedEmoji: selectedEmoji,
		width:         width,
		height:        height,
	}
}

func (f *FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (f *FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	var cmd tea.Cmd

	// Handle emoji modal
	if f.showEmojiModal {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				f.showEmojiModal = false
				f.emojiSearch.SetValue("")
				f.emojiIndex = 0
				f.scrollOffset = 0
				f.focusIndex = 1 // Return to emoji field
				f.nameInput.Blur()
				return *f, nil
			case "enter":
				// Select current emoji
				filtered := f.getFilteredEmojis()
				if len(filtered) > 0 && f.emojiIndex < len(filtered) {
					f.selectedEmoji = filtered[f.emojiIndex]
				}
				f.showEmojiModal = false
				f.emojiSearch.SetValue("")
				f.emojiIndex = 0
				f.scrollOffset = 0
				f.focusIndex = 1 // Return to emoji field
				f.nameInput.Blur()
				return *f, nil
			case "left":
				if f.emojiIndex > 0 {
					f.emojiIndex--
					f.ensureVisibleEmoji()
				}
			case "right":
				filtered := f.getFilteredEmojis()
				if f.emojiIndex < len(filtered)-1 {
					f.emojiIndex++
					f.ensureVisibleEmoji()
				}
			case "up":
				if f.emojiIndex >= 8 {
					f.emojiIndex -= 8
					f.ensureVisibleEmoji()
				}
			case "down":
				filtered := f.getFilteredEmojis()
				if f.emojiIndex+8 < len(filtered) {
					f.emojiIndex += 8
					f.ensureVisibleEmoji()
				}
			case "pgup":
				// Scroll up by visible rows
				const maxVisibleRows = 8
				f.emojiIndex -= 8 * maxVisibleRows
				if f.emojiIndex < 0 {
					f.emojiIndex = 0
				}
				f.ensureVisibleEmoji()
			case "pgdown":
				// Scroll down by visible rows
				const maxVisibleRows = 8
				filtered := f.getFilteredEmojis()
				f.emojiIndex += 8 * maxVisibleRows
				if f.emojiIndex >= len(filtered) {
					f.emojiIndex = len(filtered) - 1
				}
				f.ensureVisibleEmoji()
			default:
				// Update search input
				f.emojiSearch, cmd = f.emojiSearch.Update(msg)
				f.emojiIndex = 0 // Reset selection when searching
				f.scrollOffset = 0
				return *f, cmd
			}
		}
		return *f, nil
	}

	// Handle main form
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			f.cancelled = true
			return *f, nil
		case "enter":
			if f.focusIndex == 1 {
				// On emoji field - open picker
				f.showEmojiModal = true
				f.emojiSearch.Focus()
				return *f, textinput.Blink
			} else if f.focusIndex == 0 && f.nameInput.Value() != "" {
				// On name field with text - move to emoji field
				f.focusIndex = 1
				f.nameInput.Blur()
				return *f, nil
			}
		case "ctrl+s":
			// Ctrl+S to save from anywhere
			if f.nameInput.Value() != "" {
				f.submitted = true
				return *f, nil
			}
		case "backspace", "delete":
			// Clear emoji if on emoji field
			if f.focusIndex == 1 && f.selectedEmoji != "" {
				f.selectedEmoji = ""
				return *f, nil
			}
		case "tab", "down":
			// Toggle between fields
			f.focusIndex = (f.focusIndex + 1) % 2
			if f.focusIndex == 0 {
				f.nameInput.Focus()
				cmd = textinput.Blink
			} else {
				f.nameInput.Blur()
			}
			return *f, cmd
		case "shift+tab", "up":
			// Toggle backwards
			f.focusIndex = (f.focusIndex - 1 + 2) % 2
			if f.focusIndex == 0 {
				f.nameInput.Focus()
				cmd = textinput.Blink
			} else {
				f.nameInput.Blur()
			}
			return *f, cmd
		case " ":
			// Space on emoji field opens picker
			if f.focusIndex == 1 {
				f.showEmojiModal = true
				f.emojiSearch.Focus()
				return *f, textinput.Blink
			}
		}
	}

	// Update name input only if focused
	if f.focusIndex == 0 {
		f.nameInput, cmd = f.nameInput.Update(msg)
	}
	return *f, cmd
}

func (f *FormModel) getFilteredEmojis() []string {
	search := strings.ToLower(f.emojiSearch.Value())
	if search == "" {
		return model.CommonEmojis
	}

	var filtered []string
	for _, emoji := range model.CommonEmojis {
		keywords := model.EmojiKeywords[emoji]
		if strings.Contains(strings.ToLower(keywords), search) {
			filtered = append(filtered, emoji)
		}
	}
	return filtered
}

// ensureVisibleEmoji ensures the selected emoji is within the visible scroll area
func (f *FormModel) ensureVisibleEmoji() {
	const emojisPerRow = 8
	const maxVisibleRows = 8 // Maximum rows to show at once

	filtered := f.getFilteredEmojis()
	if len(filtered) == 0 {
		return
	}

	// Calculate current row of selected emoji
	selectedRow := f.emojiIndex / emojisPerRow

	// Adjust scroll offset to keep selected emoji visible
	if selectedRow < f.scrollOffset {
		// Selected emoji is above visible area
		f.scrollOffset = selectedRow
	} else if selectedRow >= f.scrollOffset+maxVisibleRows {
		// Selected emoji is below visible area
		f.scrollOffset = selectedRow - maxVisibleRows + 1
	}

	// Ensure scroll offset is valid
	totalRows := (len(filtered) + emojisPerRow - 1) / emojisPerRow
	maxScroll := totalRows - maxVisibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if f.scrollOffset > maxScroll {
		f.scrollOffset = maxScroll
	}
	if f.scrollOffset < 0 {
		f.scrollOffset = 0
	}
}

func (f *FormModel) ViewContent() string {
	if f.showEmojiModal {
		return f.renderEmojiModal()
	}

	var s string

	// Name input
	nameLabel := "Name:"
	if f.focusIndex == 0 {
		nameLabel = ui.SelectedItem.Render("Name:")
	}
	s += nameLabel + "\n"
	s += f.nameInput.View() + "\n\n"

	// Emoji field
	emojiLabel := "Emoji:"
	if f.focusIndex == 1 {
		emojiLabel = ui.SelectedItem.Render("Emoji:")
	}

	var emojiDisplay string
	if f.selectedEmoji == "" {
		emojiDisplay = ui.MutedText.Render("(none)")
	} else {
		emojiDisplay = f.selectedEmoji
	}

	if f.focusIndex == 1 {
		if f.selectedEmoji == "" {
			emojiDisplay = ui.SelectedItem.Render("[(none)]")
		} else {
			emojiDisplay = ui.SelectedItem.Render("[" + f.selectedEmoji + "]")
		}
	}
	s += emojiLabel + " " + emojiDisplay + "\n\n"

	s += ui.MutedText.Render("tab: switch fields  enter/space: pick emoji  backspace: clear")
	s += "\n"
	s += ui.MutedText.Render("ctrl+s: save  esc: cancel")

	return s
}

func (f *FormModel) renderEmojiModal() string {
	// This is deprecated - use renderFullScreenModal instead
	return ""
}

func (f *FormModel) renderModalBox() string {
	var s string

	// Modal title
	s += ui.Title.Render("Pick an Emoji") + "\n\n"

	// Search input
	s += "Search: " + f.emojiSearch.View() + "\n\n"

	// Show filtered emoji grid - 8 per row, max 8 rows visible
	const emojisPerRow = 8
	const maxVisibleRows = 8
	filtered := f.getFilteredEmojis()

	if len(filtered) == 0 {
		s += ui.MutedText.Render("No emojis found") + "\n"
	} else {
		totalRows := (len(filtered) + emojisPerRow - 1) / emojisPerRow
		startRow := f.scrollOffset
		endRow := startRow + maxVisibleRows
		if endRow > totalRows {
			endRow = totalRows
		}

		// Show scroll indicator at top if not at beginning
		if startRow > 0 {
			s += ui.MutedText.Render("        ▲ more above ▲") + "\n"
		}

		// Render visible rows only
		for row := startRow; row < endRow; row++ {
			var rowEmojis []string
			for col := 0; col < emojisPerRow; col++ {
				idx := row*emojisPerRow + col
				if idx >= len(filtered) {
					break
				}
				emoji := filtered[idx]
				display := emoji
				if idx == f.emojiIndex {
					display = "[" + emoji + "]"
				} else {
					display = " " + emoji + " "
				}
				rowEmojis = append(rowEmojis, display)
			}
			s += lipgloss.JoinHorizontal(lipgloss.Left, rowEmojis...) + "\n"
		}

		// Show scroll indicator at bottom if there's more content
		if endRow < totalRows {
			s += ui.MutedText.Render("        ▼ more below ▼") + "\n"
		}
	}
	s += "\n"

	s += ui.MutedText.Render("↑↓←→: navigate  pgup/pgdn: scroll  enter: select  esc: cancel")

	// Add modal box styling with dark background
	modalWidth := 54
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Background(lipgloss.Color("#1F2937")).
		Padding(1, 2).
		Width(modalWidth)

	return modalStyle.Render(s)
}

func (f *FormModel) renderFullScreenModal(width, height int) string {
	modalContent := f.renderModalBox()

	// Center the modal on the full screen with a dark background
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		modalContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#0a0e1a")),
	)
}

func (f *FormModel) GetCategory() *model.Category {
	if f.category == nil {
		return &model.Category{
			Name:  f.nameInput.Value(),
			Color: "#CCCCCC", // Default gray color (not shown in UI)
			Emoji: f.selectedEmoji,
		}
	}

	f.category.Name = f.nameInput.Value()
	f.category.Emoji = f.selectedEmoji
	// Keep existing color if editing
	return f.category
}
