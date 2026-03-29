package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/internal/world"
)

var (
	serverAdminStyle  = lipgloss.NewStyle().Foreground(green).Bold(true)
	serverNormalStyle = lipgloss.NewStyle().Foreground(green).Faint(true)
	serverCardStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(green).Padding(0, 1)
	portOpenStyle     = lipgloss.NewStyle().Foreground(green).Bold(true)
	portClosedStyle   = lipgloss.NewStyle().Foreground(green).Faint(true)
)

func (m model) serversBindings() []keyBinding {
	switch m.state {
	case serversView:
		return []keyBinding{
			{"tab", "logs"},
			{"↑↓", "navigate"},
			{"enter", "details"},
			{"ctrl+t", "terminal"},
			{"q", "quit"},
		}
	case serverDetailView:
		return []keyBinding{
			{"↑↓", "scroll"},
			{"esc", "back"},
		}
	}
	return nil
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
	servers := m.worldServers()
	if m.selectedServer < 0 || m.selectedServer >= len(servers) {
		return faintStyle.Render("  no server selected")
	}
	s := servers[m.selectedServer]

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

	cardHeight := len(cardLines) + 3
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

type serversModel struct {
	serverIndex        int
	serverListOffset   int
	selectedServer     int
	serverDetailOffset int
}

func NewServersModel() serversModel {
	return serversModel{selectedServer: -1}
}

func (m *model) handleServersKey(key string) {
	servers := m.worldServers()
	switch key {
	case "up", "k":
		if m.serverIndex > 0 {
			m.serverIndex--
		}
	case "down", "j":
		if m.serverIndex < len(servers)-1 {
			m.serverIndex++
		}
	case "enter":
		if len(servers) > 0 {
			m.selectedServer = m.serverIndex
			m.serverDetailOffset = 0
			m.stateStack, m.state = pushState(m.stateStack, m.state, serverDetailView)
		}
	}
	// keep serverIndex in the visible window
	visibleLines := m.logBodyHeight()
	if m.serverIndex < m.serverListOffset {
		m.serverListOffset = m.serverIndex
	}
	if m.serverIndex >= m.serverListOffset+visibleLines {
		m.serverListOffset = m.serverIndex - visibleLines + 1
	}
}

func (m *model) handleServerDetailKey(key string) {
	servers := m.worldServers()
	if m.selectedServer >= 0 && m.selectedServer < len(servers) {
		maxOffset := max(0, len(servers[m.selectedServer].Processes)-(m.logBodyHeight()-7))
		switch key {
		case "up", "k":
			if m.serverDetailOffset > 0 {
				m.serverDetailOffset--
			}
		case "down", "j":
			if m.serverDetailOffset < maxOffset {
				m.serverDetailOffset++
			}
		case "esc":
			m.selectedServer = -1
			m.stateStack, m.state = popState(m.stateStack)
		}
	} else {
		if key == "esc" {
			m.stateStack, m.state = popState(m.stateStack)
		}
	}
}

// worldServers returns the server list from world, or nil if world isn't loaded.
func (m model) worldServers() []world.BitServer {
	if m.world == nil {
		return nil
	}
	return m.world.Servers
}
