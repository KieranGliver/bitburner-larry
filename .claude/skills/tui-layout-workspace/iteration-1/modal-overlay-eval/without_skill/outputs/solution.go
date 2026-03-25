package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	listView uint = iota
	titleView
	bodyView
	deleteConfirmView
)

var (
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
)

type model struct {
	state     uint
	width     int
	height    int
	notes     []string
	listIndex int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			case "up", "k":
				if m.listIndex > 0 {
					m.listIndex--
				}
			case "down", "j":
				if m.listIndex < len(m.notes)-1 {
					m.listIndex++
				}
			case "d":
				if len(m.notes) > 0 {
					m.state = deleteConfirmView
				}
			}
		case deleteConfirmView:
			switch msg.String() {
			case "y":
				m.notes = append(m.notes[:m.listIndex], m.notes[m.listIndex+1:]...)
				if m.listIndex >= len(m.notes) && m.listIndex > 0 {
					m.listIndex--
				}
				m.state = listView
			case "n", "esc":
				m.state = listView
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var sb strings.Builder

	if m.state == deleteConfirmView {
		// Show the list behind the dialog
		sb.WriteString("Notes\n\n")
		for i, note := range m.notes {
			if i == m.listIndex {
				sb.WriteString(selectedStyle.Render("> "+note) + "\n")
			} else {
				sb.WriteString("  " + note + "\n")
			}
		}
		sb.WriteString("\n")

		// Render dialog inline (not an overlay — just appended below the list)
		dialogContent := "Delete this note? (y/n)\n\n" + dimStyle.Render("y: confirm  n/esc: cancel")
		dialog := dialogStyle.Render(dialogContent)
		sb.WriteString(dialog)

		return tea.NewView(sb.String())
	}

	// Normal list view
	sb.WriteString("Notes\n\n")
	for i, note := range m.notes {
		prefix := "  "
		if i == m.listIndex {
			prefix = "> "
		}
		sb.WriteString(prefix + note + "\n")
	}
	sb.WriteString("\n" + dimStyle.Render("d: delete  ↑↓: navigate  q: quit"))

	return tea.NewView(sb.String())
}

func main() {
	m := model{
		notes: []string{"Shopping list", "Meeting notes", "Project ideas", "Random thoughts"},
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
