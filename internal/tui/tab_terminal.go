package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	larcmd "github.com/KieranGliver/bitburner-larry/cmd"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
)

type terminalResultMsg struct {
	output string
	cmd    string
}

type terminalModel struct {
	termTextinput  textinput.Model
	cmdHistory     []string
	cmdHistoryIdx  int
	terminalCmd    string
	terminalOutput string
	terminalLogIdx int
	terminalCmdLog BinLog
}

func (tm *terminalModel) Open() error {
	var err error
	if err = tm.terminalCmdLog.Open(); err != nil {
		return err
	}

	ent := &termEntry{}
	eof := false
	eof, err = tm.terminalCmdLog.Read(ent)
	for !eof {
		if err != nil {
			return err
		}
		tm.cmdHistory = append(tm.cmdHistory, ent.cmd)
		eof, err = tm.terminalCmdLog.Read(ent)

	}
	return nil
}

func (tm *terminalModel) Close() error {
	return tm.terminalCmdLog.Close()
}

func (m model) terminalBindings() []keyBinding {
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

var popupStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(green).
	Padding(1, 2)

func (m model) renderTerminalView() string {
	var popupContent strings.Builder
	popupContent.WriteString(m.termTextinput.View())
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
	return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, popup) + "\n"
}

func (m *model) handleTerminalKey(key string) tea.Cmd {
	switch key {
	case "ctrl+d":
		if m.terminalCmd != "" && len(m.logs) > 0 {
			m.logSelected = len(m.logs) - 1
			m.logDetailOffset = 0
			m.stateStack, m.state = pushState(m.stateStack, m.state, logDetailView)
		}
	case "ctrl+c":
		m.stateStack, m.state = popState(m.stateStack)
		m.termTextinput.Blur()
		m.termTextinput.SetValue("")
		m.cmdHistoryIdx = -1
		m.terminalCmd = ""
		m.terminalOutput = ""
	case "up":
		if len(m.cmdHistory) > 0 {
			if m.cmdHistoryIdx < len(m.cmdHistory)-1 {
				m.cmdHistoryIdx++
			}
			m.termTextinput.SetValue(m.cmdHistory[len(m.cmdHistory)-1-m.cmdHistoryIdx])
			m.termTextinput.CursorEnd()
		}
	case "down":
		if m.cmdHistoryIdx > 0 {
			m.cmdHistoryIdx--
			m.termTextinput.SetValue(m.cmdHistory[len(m.cmdHistory)-1-m.cmdHistoryIdx])
			m.termTextinput.CursorEnd()
		} else {
			m.cmdHistoryIdx = -1
			m.termTextinput.SetValue("")
		}
	case "enter":
		cmdVal := m.termTextinput.Value()
		m.termTextinput.SetValue("")
		m.cmdHistoryIdx = -1
		if cmdVal != "" {
			m.cmdHistory = append(m.cmdHistory, cmdVal)
			if len(m.cmdHistory) > 500 && len(m.cmdHistory)%50 == 0 {
				m.cmdHistory = m.cmdHistory[len(m.cmdHistory)-500:]
				entries := make([]entry, len(m.cmdHistory))
				for i := range m.cmdHistory {
					entries[i] = &termEntry{cmd: m.cmdHistory[i]}
				}
				m.terminalCmdLog.Rewrite(entries)
			} else {
				m.terminalCmdLog.Write(&termEntry{cmd: cmdVal})
			}
			m.terminalCmd = cmdVal
			m.terminalOutput = ""
			conn := m.conn
			return func() tea.Msg {
				output := larcmd.ExecuteCommand(cmdVal, conn)
				if output == "" {
					output = "(no output)"
				}
				return terminalResultMsg{output: output, cmd: cmdVal}
			}
		}
	}
	return nil
}

func (m *model) handleTerminalResult(msg terminalResultMsg) tea.Cmd {
	m.terminalOutput = msg.output
	cmdVal := m.terminalCmd
	output := msg.output
	return func() tea.Msg {
		return logger.InfoDetail(fmt.Sprintf("[terminal] %s", cmdVal), output)
	}
}

func NewTerminalModel() (tm terminalModel, err error) {
	tm = terminalModel{
		termTextinput:  textinput.New(),
		cmdHistoryIdx:  -1,
		terminalLogIdx: -1,
		terminalCmdLog: BinLog{
			FileName: "./bin/.cmdlog",
		},
	}
	err = tm.Open()
	if err != nil {
		return tm, err
	}
	return
}
