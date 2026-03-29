package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
)

var (
	logTimeStyle  = lipgloss.NewStyle().Foreground(green).Faint(true)
	logInfoStyle  = lipgloss.NewStyle().Foreground(green).Faint(true)
	logWarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	logErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
)

const maxLogs = 500

type logsModel struct {
	logs            []logger.LogEntry
	logOffset       int
	logSelected     int
	logDetailOffset int
	logFile         *os.File
}

func (lm *logsModel) Open() error {
	var err error
	lm.logFile, err = os.OpenFile("./bin/.larrylog", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	return err
}

func (lm *logsModel) Close() error {
	if lm.logFile != nil {
		return lm.logFile.Close()
	}
	return nil
}

func (lm *logsModel) AppendLog(entry logger.LogEntry) {
	lm.logs = append(lm.logs, entry)
	if len(lm.logs) > maxLogs {
		lm.logs = lm.logs[len(lm.logs)-maxLogs:]
	}
	if lm.logFile != nil {
		fmt.Fprintf(lm.logFile, "%s  %s  %s\n", entry.Time.Format("15:04:05"), entry.Level, entry.Summary)
		if entry.Detail != "" {
			fmt.Fprintf(lm.logFile, "%s\n", entry.Detail)
		}
	}
}

func (m model) logsBindings() []keyBinding {
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
	}
	return nil
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

	visibleLines := m.logBodyHeight() - 2
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

func NewLogsModel() (lm logsModel, err error) {
	err = lm.Open()
	return
}

func (m *model) handleLogsKey(key string) {
	switch key {
	case "up", "k":
		if m.logSelected > 0 {
			m.logSelected--
		}
	case "down", "j":
		if m.logSelected < len(m.logs)-1 {
			m.logSelected++
		}
	case "enter":
		if len(m.logs) > 0 {
			m.logDetailOffset = 0
			m.stateStack, m.state = pushState(m.stateStack, m.state, logDetailView)
		}
	}
	// keep logSelected in the visible window
	visibleLines := m.logBodyHeight()
	if m.logSelected < m.logOffset {
		m.logOffset = m.logSelected
	}
	if m.logSelected >= m.logOffset+visibleLines {
		m.logOffset = m.logSelected - visibleLines + 1
	}
}

func (m *model) handleLogsDetailKey(key string) {
	detail := m.logs[m.logSelected].Detail
	if detail == "" {
		detail = m.logs[m.logSelected].Summary
	}
	lines := splitLines(detail)
	maxOffset := max(0, len(lines)-(m.logBodyHeight()-2))
	switch key {
	case "up", "k":
		if m.logDetailOffset > 0 {
			m.logDetailOffset--
		}
	case "down", "j":
		if m.logDetailOffset < maxOffset {
			m.logDetailOffset++
		}
	case "esc":
		m.stateStack, m.state = popState(m.stateStack)
	}
}

// logBodyHeight returns how many log lines fit in the body area.
func (m model) logBodyHeight() int {
	// header(1) + tabBar(1) + blank(3) + statusBar(1) = 6
	return max(1, m.height-6)
}
