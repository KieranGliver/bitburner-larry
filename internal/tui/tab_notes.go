package tui

import (
	"math/rand"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type notesModel struct {
	notes         []noteEntry
	currNote      noteEntry
	listIndex     int
	noteTextarea  textarea.Model
	noteTextinput textinput.Model
	noteLog       BinLog
}

func (nm *notesModel) Open() error {
	var err error
	if err = nm.noteLog.Open(); err != nil {
		return err
	}

	ent := &noteEntry{}
	eof := false
	eof, err = nm.noteLog.Read(ent)
	for !eof {
		if err != nil {
			return err
		}
		nm.notes = append(nm.notes, *ent)
		eof, err = nm.noteLog.Read(ent)

	}
	return nil
}

func (nm *notesModel) Close() error {
	return nm.noteLog.Close()
}

func (m *model) handleNoteListKey(key string) {
	switch key {
	case "n":
		m.noteTextinput.SetValue("")
		m.noteTextinput.Focus()
		m.currNote = noteEntry{}
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

func (m *notesModel) nextNoteID() uint64 {
	return uint64(time.Now().Unix())<<32 | uint64(rand.Uint32())
}

func (m *model) handleNoteBodyKey(key string) tea.Cmd {
	switch key {
	case "ctrl+s":
		m.currNote.Body = m.noteTextarea.Value()

		if m.currNote.id != 0 {
			found := false
			for i, note := range m.notes {
				if note.id == m.currNote.id {
					m.notes[i] = m.currNote
					found = true
					break
				}
			}
			if !found {
				m.notes = append(m.notes, m.currNote)
			}
			entries := make([]entry, len(m.notes))
			for i := range m.notes {
				entries[i] = &m.notes[i]
			}
			if err := m.noteLog.Rewrite(entries); err != nil {
				return tea.Quit
			}
		} else {
			m.currNote.id = m.nextNoteID()
			m.notes = append(m.notes, m.currNote)
			if err := m.noteLog.Write(&m.currNote); err != nil {
				return tea.Quit
			}
		}
		m.currNote = noteEntry{}
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

func NewNotesModel() (notesModel, error) {

	nm := notesModel{
		noteTextarea:  textarea.New(),
		noteTextinput: textinput.New(),
		noteLog: BinLog{
			FileName: "./bin/.notelog",
		},
	}

	if err := nm.Open(); err != nil {
		return nm, err
	}

	return nm, nil
}
