package main

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	listView uint = iota
	titleView
	bodyView
)

var (
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("99")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	faintStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)
	enumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)
)

type model struct {
	state     uint
	store     *Store
	notes     []Note
	currNote  Note
	listIndex int
	textarea  textarea.Model
	textinput textinput.Model
	width     int
	height    int
}

func newModel(store *Store) model {
	return model{
		state:     listView,
		store:     store,
		textarea:  textarea.New(),
		textinput: textinput.New(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch m.state {
		case listView:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "n":
				m.textinput.SetValue("")
				m.textinput.Focus()
				m.currNote = Note{}
				m.state = titleView
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
					m.textarea.SetValue(m.currNote.Body)
					m.textarea.Focus()
					m.state = bodyView
				}
			}
		case titleView:
			switch msg.String() {
			case "enter":
				if title := m.textinput.Value(); title != "" {
					m.currNote.Title = title
					m.textarea.SetValue("")
					m.textarea.Focus()
					m.state = bodyView
				}
			case "esc":
				m.state = listView
			}
		case bodyView:
			switch msg.String() {
			case "ctrl+s":
				m.currNote.Body = m.textarea.Value()
				m.store.SaveNote(m.currNote)
				m.notes, _ = m.store.GetNotes()
				m.state = listView
			case "esc":
				m.state = listView
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	header := headerStyle.Render("NOTES")
	if m.width > 0 {
		header = headerStyle.Width(m.width).Render("NOTES")
	}

	var content string
	switch m.state {
	case bodyView:
		content = "Note: \n\n" + m.textarea.View() + "\n\n" + faintStyle.Render("ctrl+s - save, esc - discard")
	case titleView:
		content = "Note title: \n\n" + m.textinput.View() + "\n\n" + faintStyle.Render("enter - save, esc - discard")
	case listView:
		var sb strings.Builder
		for i, n := range m.notes {
			prefix := " "
			if i == m.listIndex {
				prefix = ">"
			}
			shortBody := strings.ReplaceAll(n.Body, "\n", " ")
			if len(shortBody) > 30 {
				shortBody = shortBody[:30]
			}
			sb.WriteString(enumeratorStyle.Render(prefix) + n.Title + " | " + faintStyle.Render(shortBody) + "\n")
		}
		sb.WriteString(faintStyle.Render("n - new note, q - quit"))
		content = sb.String()
	}

	var statusText string
	switch m.state {
	case listView:
		statusText = "MODE: LIST"
	case titleView:
		statusText = "MODE: TITLE"
	case bodyView:
		statusText = "MODE: BODY"
	}
	status := statusStyle.Render(statusText)

	return tea.NewView(strings.Join([]string{header, "\n" + content + "\n", status}, "\n"))
}
