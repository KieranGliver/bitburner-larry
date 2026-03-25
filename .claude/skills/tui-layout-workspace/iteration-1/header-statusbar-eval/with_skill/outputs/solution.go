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

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("99")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	modeBadgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("99")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			MarginRight(1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	faintStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)
	enumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)
)

type model struct {
	state     uint
	width     int
	height    int
	notes     []Note
	currNote  Note
	listIndex int
	textarea  textarea.Model
	textinput textinput.Model
	store     *Store
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
		// Propagate to components
		m.textarea.SetWidth(m.width - 4)
		m.textarea.SetHeight(m.height - 6) // rough estimate; shell will constrain
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
					m.textarea.CursorEnd()
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
				if err := m.store.SaveNote(m.currNote); err != nil {
					return m, tea.Quit
				}
				m.notes, _ = m.store.GetNotes()
				m.currNote = Note{}
				m.state = listView
			case "esc":
				m.state = listView
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	return tea.NewView(m.renderShell())
}

func (m model) renderShell() string {
	header := headerStyle.Width(m.width).Render("NOTES")
	status := m.renderStatusBar()
	headerH := lipgloss.Height(header)
	statusH := lipgloss.Height(status)
	bodyH := m.height - headerH - statusH
	if bodyH < 1 {
		bodyH = 1
	}
	body := lipgloss.NewStyle().
		Width(m.width).
		Height(bodyH).
		Padding(1, 2).
		Render(m.renderContent())
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m model) renderStatusBar() string {
	var mode, hints string
	switch m.state {
	case listView:
		mode = "LIST"
		hints = "n new  •  ↑↓/jk navigate  •  enter open  •  q quit"
	case titleView:
		mode = "TITLE"
		hints = "enter save  •  esc cancel"
	case bodyView:
		mode = "BODY"
		hints = "ctrl+s save  •  esc discard"
	}
	badge := modeBadgeStyle.Render(mode)
	hint := hintStyle.Render(hints)
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(badge)-lipgloss.Width(hint)-2))
	return statusBarStyle.Width(m.width).Render(badge + gap + hint)
}

func (m model) renderContent() string {
	switch m.state {
	case bodyView:
		return "Note:\n\n" + m.textarea.View()
	case titleView:
		return "Note title:\n\n" + m.textinput.View()
	case listView:
		var s strings.Builder
		for i, n := range m.notes {
			prefix := "  "
			if i == m.listIndex {
				prefix = "> "
			}
			shortBody := strings.ReplaceAll(n.Body, "\n", " ")
			if len(shortBody) > 30 {
				shortBody = shortBody[:30] + "…"
			}
			s.WriteString(enumeratorStyle.Render(prefix) + n.Title + " " + faintStyle.Render(shortBody) + "\n")
		}
		if len(m.notes) == 0 {
			s.WriteString(faintStyle.Render("No notes yet. Press n to create one.") + "\n")
		}
		return s.String()
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
