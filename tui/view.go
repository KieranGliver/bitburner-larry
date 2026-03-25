package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/communication"
)

var (
	appNameStyle    = lipgloss.NewStyle().Background(lipgloss.Color("99")).Padding(0, 1)
	faintStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)
	enumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)
)

func (m model) View() tea.View {

	connection := "○ - Disconnected"
	if m.conn != nil && m.conn.Status == communication.Connected {
		connection = "● - Connected"
	}

	width := m.width

	left := appNameStyle.Render("Larry")
	right := faintStyle.Render(connection)
	spacer := strings.Repeat(" ", max(width-lipgloss.Width(left)-lipgloss.Width(right), 0))

	header := left + spacer + right

	s := header + "\n\n"

	if m.state == bodyView {
		s += "Note: \n\n"
		s += m.textarea.View() + "\n\n"
		s += faintStyle.Render("ctrl+s - save, esc - discard")
	}

	if m.state == titleView {
		s += "Note title: \n\n"
		s += m.textinput.View() + "\n\n"
		s += faintStyle.Render("enter - save, esc - discard")
	}

	if m.state == listView {
		for i, n := range m.notes {
			prefix := " "
			if i == m.listIndex {
				prefix = ">"
			}
			shortBody := strings.ReplaceAll(n.Body, "/n", " ")
			if len(shortBody) > 30 {
				shortBody = shortBody[:30]
			}
			s += enumeratorStyle.Render(prefix) + n.Title + " | " + faintStyle.Render(shortBody) + "\n"
		}
		s += faintStyle.Render("n - new note, q - quit")
	}

	return tea.NewView(s)
}
