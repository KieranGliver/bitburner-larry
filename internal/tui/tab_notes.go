package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/internal/db"
)

type notesModel struct {
	notes         []db.Note
	currNote      db.Note
	listIndex     int
	noteTextarea  textarea.Model
	noteTextinput textinput.Model
}

func (m *model) handleNoteListKey(key string) {
	switch key {
	case "n":
		m.noteTextinput.SetValue("")
		m.noteTextinput.Focus()
		m.currNote = db.Note{}
		m.state = noteTitleView
	case "up", "k":
		if m.listIndex > 0 {
			m.listIndex--
		}
	case "down", "j":
		if m.listIndex < len(m.notes)-1 {
			m.listIndex++
		}
	case "enter":
		if len(m.notes) > 0 {
			m.currNote = m.notes[m.listIndex]
			m.noteTextarea.SetValue(m.currNote.Body)
			m.noteTextarea.Focus()
			m.noteTextarea.CursorEnd()
			m.state = noteBodyView
		}
	}
}

func (m *model) handleNoteTitleKey(key string) {
	switch key {
	case "enter":
		title := m.noteTextinput.Value()
		if title != "" {
			m.currNote.Title = title
			m.noteTextarea.SetValue("")
			m.noteTextarea.Focus()
			m.noteTextarea.CursorEnd()
			m.state = noteBodyView
		}
	case "esc":
		m.state = noteListView
	}
}

func (m *model) handleNoteBodyKey(key string) tea.Cmd {
	switch key {
	case "ctrl+s":
		body := m.noteTextarea.Value()
		m.currNote.Body = body

		var err error
		if err = m.store.SaveNote(m.currNote); err != nil {
			return tea.Quit
		}
		m.notes, err = m.store.GetNotes()
		if err != nil {
			return tea.Quit
		}
		m.currNote = db.Note{}
		m.state = noteListView
	case "esc":
		m.state = noteListView
	}
	return nil
}

func (m model) notesBindings() []keyBinding {
	switch m.state {
	case noteListView:
		return []keyBinding{
			{"tab", "servers"},
			{"n", "new"},
			{"↑↓", "navigate"},
			{"enter", "open"},
			{"ctrl+t", "terminal"},
			{"q", "quit"},
		}
	case noteTitleView:
		return []keyBinding{
			{"enter", "confirm"},
			{"esc", "cancel"},
		}
	case noteBodyView:
		return []keyBinding{
			{"ctrl+s", "save"},
			{"esc", "cancel"},
		}
	}
	return nil
}

var enumeratorStyle = lipgloss.NewStyle().Foreground(green).Bold(true).MarginRight(1)

func (m model) renderNoteListView() string {
	var sb strings.Builder
	for i, n := range m.notes {
		prefix := " "
		if i == m.listIndex {
			prefix = ">"
		}
		shortBody := strings.ReplaceAll(n.Body, "/n", " ")
		if len(shortBody) > 30 {
			shortBody = shortBody[:30]
		}
		sb.WriteString(enumeratorStyle.Render(prefix) + n.Title + " | " + faintStyle.Render(shortBody) + "\n")
	}
	return sb.String()
}

func (m model) renderNoteTitleView() string {
	return "Note title: \n\n" + m.noteTextinput.View()
}

func (m model) renderNoteBodyView() string {
	return "Note: \n\n" + m.noteTextarea.View()
}

func NewNotesModel(store *db.Store) (notesModel, error) {
	notes, err := store.GetNotes()
	if err != nil {
		return notesModel{}, err
	}
	return notesModel{
		notes:         notes,
		noteTextarea:  textarea.New(),
		noteTextinput: textinput.New(),
	}, nil
}
