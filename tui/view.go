package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/communication"
)

var (
	green = lipgloss.Color("#00FC00")

	// Header
	headerBg    = lipgloss.NewStyle().Background(lipgloss.Color("232"))
	headerLogo  = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Bold(true).Padding(0, 2)
	headerTitle = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 1)
	headerSub   = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 2)

	// Status bar
	statusBg          = lipgloss.NewStyle().Background(lipgloss.Color("232"))
	connectedStyle    = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Bold(true).Padding(0, 1)
	disconnectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(lipgloss.Color("244")).Padding(0, 1)
	keyCapStyle       = lipgloss.NewStyle().Background(green).Foreground(lipgloss.Color("232")).Bold(true).Padding(0, 1)
	keyDescStyle      = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 1)

	// Body
	faintStyle      = lipgloss.NewStyle().Foreground(green).Faint(true)
	enumeratorStyle = lipgloss.NewStyle().Foreground(green).Bold(true).MarginRight(1)
)

func (m model) renderHeader() string {
	logo := headerLogo.Render("LARRY ☺")
	title := headerTitle.Render("bitburner filesync")
	sub := headerSub.Render("ws://localhost:12525")

	left := logo + title
	spacerW := max(0, m.width-lipgloss.Width(left)-lipgloss.Width(sub))
	spacer := headerBg.Width(spacerW).Render("")

	return left + spacer + sub
}

type keyBinding struct{ key, desc string }

func (m model) currentBindings() []keyBinding {
	switch m.state {
	case listView:
		return []keyBinding{
			{"n", "new"},
			{"↑↓", "navigate"},
			{"enter", "open"},
			{"q", "quit"},
		}
	case titleView:
		return []keyBinding{
			{"enter", "confirm"},
			{"esc", "cancel"},
		}
	case bodyView:
		return []keyBinding{
			{"ctrl+s", "save"},
			{"esc", "cancel"},
		}
	}
	return nil
}

func (m model) renderStatusBar() string {
	var conn string
	if m.conn != nil && m.conn.Status == communication.Connected {
		conn = connectedStyle.Render("● Connected")
	} else {
		conn = disconnectedStyle.Render("○ Disconnected")
	}

	var parts []string
	for _, b := range m.currentBindings() {
		parts = append(parts, keyCapStyle.Render(b.key)+keyDescStyle.Render(b.desc))
	}
	keys := strings.Join(parts, "")

	spacerW := max(0, m.width-lipgloss.Width(conn)-lipgloss.Width(keys))
	spacer := statusBg.Width(spacerW).Render("")

	return conn + spacer + keys
}

func (m model) View() tea.View {
	header := m.renderHeader()
	statusBar := m.renderStatusBar()

	var body strings.Builder

	if m.state == bodyView {
		body.WriteString("Note: \n\n")
		body.WriteString(m.textarea.View())
	}

	if m.state == titleView {
		body.WriteString("Note title: \n\n")
		body.WriteString(m.textinput.View())
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
			body.WriteString(enumeratorStyle.Render(prefix) + n.Title + " | " + faintStyle.Render(shortBody) + "\n")
		}
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n"+body.String(),
		statusBar,
	))
}
