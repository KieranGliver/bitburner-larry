package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
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

	// Status bar
	statusBg          = lipgloss.NewStyle().Background(lipgloss.Color("232"))
	connectedStyle    = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Bold(true).Padding(0, 1)
	disconnectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(lipgloss.Color("244")).Padding(0, 1)
	keyCapStyle       = lipgloss.NewStyle().Background(green).Foreground(lipgloss.Color("232")).Bold(true).Padding(0, 1)
	keyDescStyle      = lipgloss.NewStyle().Background(lipgloss.Color("232")).Foreground(green).Faint(true).Padding(0, 1)

	// Body
	faintStyle = lipgloss.NewStyle().Foreground(green).Faint(true)
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
	notesTab := tabStyle(effectiveState == noteListView || effectiveState == noteTitleView || effectiveState == noteBodyView, "Notes")
	serversTab := tabStyle(effectiveState == serversView || effectiveState == serverDetailView, "Servers")
	bar := logsTab + notesTab + serversTab
	fillW := max(0, m.width-lipgloss.Width(bar))
	return bar + tabBarFill.Width(fillW).Render("")
}

type keyBinding struct{ key, desc string }

func (m model) currentBindings() []keyBinding {
	switch m.state {
	case logsView, logDetailView:
		return m.logsBindings()
	case noteListView, noteTitleView, noteBodyView:
		return m.notesBindings()
	case serversView, serverDetailView:
		return m.serversBindings()
	case terminalView:
		return m.terminalBindings()
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

	case noteBodyView:
		body.WriteString(m.renderNoteBodyView())

	case noteTitleView:
		body.WriteString(m.renderNoteTitleView())

	case noteListView:
		body.WriteString(m.renderNoteListView())

	case serversView:
		body.WriteString(m.renderServersView())

	case serverDetailView:
		body.WriteString(m.renderServerDetailView())

	case terminalView:
		body.WriteString(m.renderTerminalView())
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabBar,
		"\n"+body.String(),
		statusBar,
	))
}
