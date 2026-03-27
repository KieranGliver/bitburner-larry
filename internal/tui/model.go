package tui

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	larcmd "github.com/KieranGliver/bitburner-larry/cmd"
	"github.com/KieranGliver/bitburner-larry/internal/brain"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/db"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
)

const (
	logsView uint = iota
	logDetailView
	listView
	titleView
	bodyView
	terminalView
	serversView
	serverDetailView
)

const maxLogs = 500

type model struct {
	state           uint
	stateStack      []uint
	width           int
	height          int
	store           *db.Store
	notes           []db.Note
	currNote        db.Note
	listIndex            int
	serverIndex          int
	serverListOffset     int
	selectedServer       *brain.BitServer
	serverDetailOffset   int
	textarea             textarea.Model
	textinput       textinput.Model
	termInput       textinput.Model
	conn            *communication.BitburnerConn
	logs            []logger.LogEntry
	logOffset       int
	logSelected     int
	logDetailOffset int
	logFile         *os.File
	cmdHistory      []string
	cmdHistoryIdx   int
	terminalCmd     string
	terminalOutput  string
	world           *brain.World
}

type terminalResultMsg string

func pushState(stack []uint, current, next uint) ([]uint, uint) {
	return append(stack, current), next
}

func popState(stack []uint) ([]uint, uint) {
	if len(stack) == 0 {
		return stack, logsView
	}
	return stack[:len(stack)-1], stack[len(stack)-1]
}

func NewModel(store *db.Store) model {
	notes, err := store.GetNotes()
	if err != nil {
		fmt.Printf("Unable to get notes: %v", err)
	}
	cmdHistory, _ := store.GetCommandHistory(100)
	logFile, _ := os.OpenFile("larry.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	return model{
		state:         logsView,
		store:         store,
		notes:         notes,
		textarea:      textarea.New(),
		textinput:     textinput.New(),
		termInput:     textinput.New(),
		logFile:       logFile,
		cmdHistory:    cmdHistory,
		cmdHistoryIdx: -1,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)
	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	if m.state == terminalView {
		m.termInput, cmd = m.termInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case communication.BitburnerConnected:
		m.conn = msg.Conn

	case communication.BitburnerDisconnected:

	case *brain.World:
		m.world = msg

	case terminalResultMsg:
		m.terminalOutput = string(msg)
		if w := larcmd.CurrentWorld; w != nil {
			m.world = w
		}
		cmdVal := m.terminalCmd
		output := string(msg)
		return m, func() tea.Msg {
			return logger.InfoDetail(fmt.Sprintf("[terminal] %s", cmdVal), output)
		}

	case logger.LogMsg:
		visibleLines := m.logBodyHeight()
		atBottom := m.logSelected >= len(m.logs)-1

		m.logs = append(m.logs, msg.Entry)
		if len(m.logs) > maxLogs {
			m.logs = m.logs[len(m.logs)-maxLogs:]
		}

		if atBottom {
			m.logSelected = len(m.logs) - 1
			m.logOffset = max(0, len(m.logs)-visibleLines)
		}

		if m.logFile != nil {
			fmt.Fprintf(m.logFile, "%s  %s  %s\n", msg.Entry.Time.Format("15:04:05"), msg.Entry.Level, msg.Entry.Summary)
			if msg.Entry.Detail != "" {
				fmt.Fprintf(m.logFile, "%s\n", msg.Entry.Detail)
			}
		}

	case tea.KeyMsg:
		key := msg.String()

		// Open/close terminal with ctrl+t
		if key == "ctrl+t" {
			if m.state == terminalView {
				m.stateStack, m.state = popState(m.stateStack)
				m.termInput.Blur()
				m.termInput.SetValue("")
				m.terminalCmd = ""
				m.terminalOutput = ""
			} else {
				m.stateStack, m.state = pushState(m.stateStack, m.state, terminalView)
				m.termInput.Focus()
			}
			return m, tea.Batch(cmds...)
		}

		// Tab switch — blocked while editing a note or in detail views
		if key == "tab" && m.state != titleView && m.state != bodyView && m.state != terminalView &&
			m.state != logDetailView && m.state != serverDetailView {
			switch m.state {
			case logsView:
				m.state = listView
			case listView:
				m.state = serversView
			default:
				m.state = logsView
			}
			return m, tea.Batch(cmds...)
		}

		// Quit — blocked while editing
		if key == "q" && m.state != titleView && m.state != bodyView && m.state != terminalView {
			return m, tea.Quit
		}

		switch m.state {
		case logsView:
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

		case logDetailView:
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

		case listView:
			switch key {
			case "n":
				m.textinput.SetValue("")
				m.textinput.Focus()
				m.currNote = db.Note{}
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
			switch key {
			case "enter":
				title := m.textinput.Value()
				if title != "" {
					m.currNote.Title = title
					m.textarea.SetValue("")
					m.textarea.Focus()
					m.textarea.CursorEnd()
					m.state = bodyView
				}
			case "esc":
				m.state = listView
			}

		case bodyView:
			switch key {
			case "ctrl+s":
				body := m.textarea.Value()
				m.currNote.Body = body

				var err error
				if err = m.store.SaveNote(m.currNote); err != nil {
					return m, tea.Quit
				}

				m.notes, err = m.store.GetNotes()
				if err != nil {
					return m, tea.Quit
				}

				m.currNote = db.Note{}
				m.state = listView
			case "esc":
				m.state = listView
			}

		case serversView:
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
					s := servers[m.serverIndex]
					m.selectedServer = &s
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

		case serverDetailView:
			if m.selectedServer != nil {
				maxOffset := max(0, len(m.selectedServer.Processes)-(m.logBodyHeight()-7))
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
					m.selectedServer = nil
					m.stateStack, m.state = popState(m.stateStack)
				}
			} else {
				if key == "esc" {
					m.stateStack, m.state = popState(m.stateStack)
				}
			}

		case terminalView:
			switch key {
			case "ctrl+d":
				if m.terminalCmd != "" && len(m.logs) > 0 {
					m.logSelected = len(m.logs) - 1
					m.logDetailOffset = 0
					m.stateStack, m.state = pushState(m.stateStack, m.state, logDetailView)
				}
			case "ctrl+c":
				m.stateStack, m.state = popState(m.stateStack)
				m.termInput.Blur()
				m.termInput.SetValue("")
				m.cmdHistoryIdx = -1
				m.terminalCmd = ""
				m.terminalOutput = ""
			case "up":
				if len(m.cmdHistory) > 0 {
					if m.cmdHistoryIdx < len(m.cmdHistory)-1 {
						m.cmdHistoryIdx++
					}
					m.termInput.SetValue(m.cmdHistory[len(m.cmdHistory)-1-m.cmdHistoryIdx])
					m.termInput.CursorEnd()
				}
			case "down":
				if m.cmdHistoryIdx > 0 {
					m.cmdHistoryIdx--
					m.termInput.SetValue(m.cmdHistory[len(m.cmdHistory)-1-m.cmdHistoryIdx])
					m.termInput.CursorEnd()
				} else {
					m.cmdHistoryIdx = -1
					m.termInput.SetValue("")
				}
			case "enter":
				cmdVal := m.termInput.Value()
				m.termInput.SetValue("")
				m.cmdHistoryIdx = -1
				if cmdVal != "" {
					m.cmdHistory = append(m.cmdHistory, cmdVal)
					m.store.SaveCommand(cmdVal)
					m.terminalCmd = cmdVal
					m.terminalOutput = ""
					conn := m.conn
					return m, func() tea.Msg {
						output := larcmd.ExecuteCommand(cmdVal, conn)
						if output == "" {
							output = "(no output)"
						}
						return terminalResultMsg(output)
					}
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
}

// logBodyHeight returns how many log lines fit in the body area.
func (m model) logBodyHeight() int {
	// header(1) + tabBar(1) + blank(3) + statusBar(1) = 4
	return max(1, m.height-6)
}

// worldServers returns the server list from world, or nil if world isn't loaded.
func (m model) worldServers() []brain.BitServer {
	if m.world == nil {
		return nil
	}
	return m.world.Servers
}
