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
	serversTab := tabStyle(effectiveState == serversView || effectiveState == serverDetailView, "Servers")
	batchesTab := tabStyle(effectiveState == batchesView, "Batches")

	bar := logsTab + notesTab + serversTab + batchesTab
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
			{"tab", "servers"},
			{"n", "new"},
			{"↑↓", "navigate"},
			{"enter", "open"},
			{"ctrl+t", "terminal"},
			{"q", "quit"},
		}
	case serversView:
		return []keyBinding{
			{"tab", "batches"},
			{"↑↓", "navigate"},
			{"enter", "details"},
			{"ctrl+t", "terminal"},
			{"q", "quit"},
		}
	case batchesView:
		return []keyBinding{
			{"tab", "logs"},
			{"↑↓", "scroll"},
			{"ctrl+t", "terminal"},
			{"q", "quit"},
		}
	case serverDetailView:
		return []keyBinding{
			{"↑↓", "scroll"},
			{"esc", "back"},
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

func fmtMoney(f float64) string {
	switch {
	case f >= 1e12:
		return fmt.Sprintf("$%.1ft", f/1e12)
	case f >= 1e9:
		return fmt.Sprintf("$%.1fb", f/1e9)
	case f >= 1e6:
		return fmt.Sprintf("$%.1fm", f/1e6)
	case f >= 1e3:
		return fmt.Sprintf("$%.1fk", f/1e3)
	default:
		return fmt.Sprintf("$%.0f", f)
	}
}

func fmtRAM(gb float64) string {
	if gb >= 1024 {
		return fmt.Sprintf("%.0fTB", gb/1024)
	}
	return fmt.Sprintf("%.0fGB", gb)
}

func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

var (
	serverAdminStyle   = lipgloss.NewStyle().Foreground(green).Bold(true)
	serverNormalStyle  = lipgloss.NewStyle().Foreground(green).Faint(true)
	serverCardStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(green).Padding(0, 1)
	portOpenStyle      = lipgloss.NewStyle().Foreground(green).Bold(true)
	portClosedStyle    = lipgloss.NewStyle().Foreground(green).Faint(true)
)

func (m model) renderServersView() string {
	servers := m.worldServers()
	if len(servers) == 0 {
		return "\n" + faintStyle.Render("  no world data yet — run 'col' to scan")
	}

	visibleLines := m.logBodyHeight() - 1
	start := m.serverListOffset
	end := min(start+visibleLines, len(servers))

	var sb strings.Builder
	for i, s := range servers[start:end] {
		absIdx := start + i
		cursor := "  "
		if absIdx == m.serverIndex {
			cursor = "> "
		}

		admin := "----"
		if s.HasAdminRights {
			admin = serverAdminStyle.Render("ROOT")
		} else {
			admin = portClosedStyle.Render("----")
		}

		ram := fmt.Sprintf("RAM %s/%s", fmtRAM(s.RamUsed), fmtRAM(s.MaxRam))
		ports := fmt.Sprintf("ports:%d/%d", s.OpenPortCount, s.NumOpenPortsRequired)
		hack := fmt.Sprintf("hack:%-4d", s.RequiredHackingSkill)

		var money string
		if s.MoneyMax > 0 {
			money = fmt.Sprintf("%-12s", fmtMoney(s.MoneyAvailable)+"/"+fmtMoney(s.MoneyMax))
		} else {
			money = fmt.Sprintf("%-12s", "")
		}

		hostname := fmt.Sprintf("%-24s", s.Hostname)

		line := cursor + serverNormalStyle.Render(hostname) + "  " +
			faintStyle.Render(fmt.Sprintf("%-14s", ram)) + "  " +
			admin + "  " +
			faintStyle.Render(money) + "  " +
			faintStyle.Render(hack) + "  " +
			faintStyle.Render(ports)

		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func (m model) renderServerDetailView() string {
	s := m.selectedServer
	if s == nil {
		return faintStyle.Render("  no server selected")
	}

	portStr := func(name string, open bool) string {
		if open {
			return portOpenStyle.Render(name)
		}
		return portClosedStyle.Render(name)
	}

	cardLines := []string{
		faintStyle.Render(s.Hostname) + "  " + serverNormalStyle.Render(s.OrganizationName) + "  " + faintStyle.Render(s.Ip),
		fmt.Sprintf("Admin: %s  Backdoor: %s  Cores: %d  RAM: %s/%s",
			yesNo(s.HasAdminRights), yesNo(s.BackdoorInstalled), s.CpuCores,
			fmtRAM(s.RamUsed), fmtRAM(s.MaxRam)),
		fmt.Sprintf("Hack req: %d  Diff: %.0f/%.0f  Money: %s/%s  Growth: %.0f",
			s.RequiredHackingSkill, s.HackDifficulty, s.MinDifficulty,
			fmtMoney(s.MoneyAvailable), fmtMoney(s.MoneyMax), s.ServerGrowth),
		"Ports: " + portStr("SSH", s.SshPortOpen) + " " +
			portStr("FTP", s.FtpPortOpen) + " " +
			portStr("SMTP", s.SmtpPortOpen) + " " +
			portStr("HTTP", s.HttpPortOpen) + " " +
			portStr("SQL", s.SqlPortOpen),
	}

	card := serverCardStyle.Width(m.width - 2).Render(strings.Join(cardLines, "\n"))

	var sb strings.Builder
	sb.WriteString(card + "\n")

	if len(s.Processes) == 0 {
		sb.WriteString("\n" + faintStyle.Render("  no processes running"))
		return sb.String()
	}

	sb.WriteString("\n" + faintStyle.Render(fmt.Sprintf("  %-6s  %-32s  %-7s  %s", "PID", "SCRIPT", "THREADS", "ARGS")) + "\n")

	cardHeight := len(cardLines) + 3 // border(2) + padding(1) + header line + blank
	visibleProcs := max(1, m.logBodyHeight()-cardHeight)
	start := m.serverDetailOffset
	end := min(start+visibleProcs, len(s.Processes))

	for _, p := range s.Processes[start:end] {
		var args []string
		for _, a := range p.Args {
			args = append(args, fmt.Sprintf("%v", a))
		}
		argsStr := strings.Join(args, " ")
		sb.WriteString(faintStyle.Render(fmt.Sprintf("  %-6d  %-32s  %-7d  %s", p.Pid, p.Filename, p.Threads, argsStr)) + "\n")
	}

	return sb.String()
}

func (m model) renderBatchesView() string {
	if m.world == nil {
		return "\n" + faintStyle.Render("  no world data yet — run 'col' to scan")
	}
	targets := m.world.GetBatchTargets()
	if len(targets) == 0 {
		return "\n" + faintStyle.Render("  no active batches")
	}

	header := faintStyle.Render(fmt.Sprintf("  %-24s  %8s  %8s  %8s", "TARGET", "HACK", "WEAKEN", "GROW"))
	var sb strings.Builder
	sb.WriteString(header + "\n\n")

	visibleLines := m.logBodyHeight() - 2
	start := m.batchScrollOffset
	end := min(start+visibleLines, len(targets))

	for _, host := range targets[start:end] {
		hack := m.world.GetHackTarget(host)
		weaken := m.world.GetWeakenTarget(host)
		grow := m.world.GetGrowTarget(host)
		line := fmt.Sprintf("  %-24s  %8d  %8d  %8d", host, hack, weaken, grow)
		sb.WriteString(faintStyle.Render(line) + "\n")
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

	case serversView:
		body.WriteString(m.renderServersView())

	case serverDetailView:
		body.WriteString(m.renderServerDetailView())

	case batchesView:
		body.WriteString(m.renderBatchesView())

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
