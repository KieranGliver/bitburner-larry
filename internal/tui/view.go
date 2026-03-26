package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
)

var (
	green = lipgloss.Color("#00FC00")

	// Header
	headerBg    = lipgloss.NewStyle().Background(lipgloss.Color("232"))
	headerLogo  = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Bold(true).Padding(0, 2)
	headerTitle = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 1)
	headerSub   = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 2)

	// Tab bar
	tabActiveBg   = lipgloss.NewStyle().Background(green).Foreground(lipgloss.Color("232")).Bold(true).Padding(0, 2)
	tabInactiveBg = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(green).Faint(true).Padding(0, 2)
	tabBarFill    = lipgloss.NewStyle().Background(lipgloss.Color("235"))

	// Terminal popup
	popupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(green).
			Padding(1, 2)

	// Status bar
	statusBg          = lipgloss.NewStyle().Background(lipgloss.Color("232"))
	connectedStyle    = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Bold(true).Padding(0, 1)
	disconnectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(lipgloss.Color("244")).Padding(0, 1)
	keyCapStyle       = lipgloss.NewStyle().Background(green).Foreground(lipgloss.Color("232")).Bold(true).Padding(0, 1)
	keyDescStyle      = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 1)

	// Body
	faintStyle      = lipgloss.NewStyle().Foreground(green).Faint(true)
	enumeratorStyle = lipgloss.NewStyle().Foreground(green).Bold(true).MarginRight(1)

	// Log levels
	logTimeStyle  = lipgloss.NewStyle().Foreground(green).Faint(true)
	logInfoStyle  = lipgloss.NewStyle().Foreground(green).Faint(true)
	logWarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	logErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
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

func (m model) renderTabBar() string {
	// Use the bottom of the stack to determine which top-level tab is active
	effectiveState := m.state
	if len(m.stateStack) > 0 {
		effectiveState = m.stateStack[0]
	}

	tabStyle := func(active bool, label string) string {
		if active {
			return tabActiveBg.Render(label)
		}
		return tabInactiveBg.Render(label)
	}

	logsTab := tabStyle(effectiveState == logsView || effectiveState == logDetailView, "Logs")
	notesTab := tabStyle(effectiveState == listView || effectiveState == titleView || effectiveState == bodyView, "Notes")

	bar := logsTab + notesTab
	fillW := max(0, m.width-lipgloss.Width(bar))
	return bar + tabBarFill.Width(fillW).Render("")
}

type keyBinding struct{ key, desc string }

func (m model) currentBindings() []keyBinding {
	switch m.state {
	case logsView:
		return []keyBinding{
			{"tab", "notes"},
			{"↑↓", "navigate"},
			{"enter", "expand"},
			{"ctrl+t", "terminal"},
			{"q", "quit"},
		}
	case logDetailView:
		return []keyBinding{
			{"↑↓", "scroll"},
			{"esc", "back"},
		}
	case listView:
		return []keyBinding{
			{"tab", "logs"},
			{"n", "new"},
			{"↑↓", "navigate"},
			{"enter", "open"},
			{"ctrl+t", "terminal"},
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
	case terminalView:
		bindings := []keyBinding{
			{"enter", "run"},
			{"↑↓", "history"},
			{"ctrl+c", "close"},
		}
		if m.terminalCmd != "" {
			bindings = append(bindings, keyBinding{"ctrl+d", "details"})
		}
		return bindings
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

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func (m model) renderLogsView() string {
	if len(m.logs) == 0 {
		return "\n" + faintStyle.Render("  no logs yet")
	}

	visibleLines := m.logBodyHeight()
	start := m.logOffset
	end := min(start+visibleLines, len(m.logs))
	visible := m.logs[start:end]

	var sb strings.Builder
	for i, entry := range visible {
		absIdx := m.logOffset + i
		cursor := "  "
		if absIdx == m.logSelected {
			cursor = "> "
		}

		ts := logTimeStyle.Render(entry.Time.Format("15:04:05"))

		var level string
		switch entry.Level {
		case logger.WARN:
			level = logWarnStyle.Render(entry.Level.String())
		case logger.ERROR:
			level = logErrorStyle.Render(entry.Level.String())
		default:
			level = logInfoStyle.Render(entry.Level.String())
		}

		var msg string
		switch entry.Level {
		case logger.WARN:
			msg = logWarnStyle.Render(entry.Summary)
		case logger.ERROR:
			msg = logErrorStyle.Render(entry.Summary)
		default:
			msg = faintStyle.Render(entry.Summary)
		}

		sb.WriteString(cursor + ts + "  " + level + "  " + msg + "\n")
	}
	return sb.String()
}

func (m model) renderLogDetailView() string {
	if len(m.logs) == 0 {
		return ""
	}
	entry := m.logs[m.logSelected]

	content := entry.Detail
	if content == "" {
		content = entry.Summary
	}
	lines := splitLines(content)

	visibleLines := m.logBodyHeight() - 2 // reserve 2 for header
	start := m.logDetailOffset
	end := min(start+visibleLines, len(lines))

	var sb strings.Builder
	header := logTimeStyle.Render(entry.Time.Format("15:04:05")) + "  " + logInfoStyle.Render(entry.Level.String()) + "  " + faintStyle.Render(entry.Summary)
	sb.WriteString(header + "\n\n")
	for _, l := range lines[start:end] {
		sb.WriteString(faintStyle.Render(l) + "\n")
	}
	return sb.String()
}

func (m model) View() tea.View {
	header := m.renderHeader()
	tabBar := m.renderTabBar()
	statusBar := m.renderStatusBar()

	var body strings.Builder

	switch m.state {
	case logsView:
		body.WriteString(m.renderLogsView())

	case logDetailView:
		body.WriteString(m.renderLogDetailView())

	case bodyView:
		body.WriteString("Note: \n\n")
		body.WriteString(m.textarea.View())

	case titleView:
		body.WriteString("Note title: \n\n")
		body.WriteString(m.textinput.View())

	case listView:
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

	case terminalView:
		var popupContent strings.Builder
		popupContent.WriteString(m.termInput.View())
		if m.terminalCmd != "" {
			// border(2) + popup padding top/bottom(2) + input(1) + blank(1) + cmd(1) = 7
			maxOutputLines := max(1, m.logBodyHeight()-7)
			result := "\n\n> " + m.terminalCmd
			if m.terminalOutput != "" {
				lines := splitLines(m.terminalOutput)
				if len(lines) > maxOutputLines {
					result += "\n" + strings.Join(lines[:maxOutputLines], "\n")
					result += fmt.Sprintf("\n(+%d more lines — press d for details)", len(lines)-maxOutputLines)
				} else {
					result += "\n" + m.terminalOutput
				}
			}
			popupContent.WriteString(faintStyle.Render(result))
		}
		popup := popupStyle.Width(m.width / 2).Render(popupContent.String())
		bodyH := m.logBodyHeight()
		body.WriteString(lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, popup) + "\n")
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabBar,
		"\n"+body.String(),
		statusBar,
	))
}
