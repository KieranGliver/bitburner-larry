package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/KieranGliver/bitburner-larry/internal/world"
)

const (
	logsView uint = iota
	logDetailView
	noteListView
	noteTitleView
	noteBodyView
	terminalView
	serversView
	serverDetailView
)

type model struct {
	// Global data
	state      uint
	stateStack []uint
	width      int
	height     int
	conn       *communication.BitburnerConn
	world      *world.World
	runCmd     func(string) string

	notesModel
	serversModel
	logsModel
	terminalModel
}

var navBlockedStates = map[uint]bool{
	noteTitleView:    true,
	noteBodyView:     true,
	terminalView:     true,
	logDetailView:    true,
	serverDetailView: true,
}

func (m model) canTab() bool {
	data, ok := navBlockedStates[m.state]
	return !ok && !data
}

func pushState(stack []uint, current, next uint) ([]uint, uint) {
	return append(stack, current), next
}

func popState(stack []uint) ([]uint, uint) {
	if len(stack) == 0 {
		return stack, logsView
	}
	return stack[:len(stack)-1], stack[len(stack)-1]
}

func NewModel(runCmd func(string) string) model {
	nm, err := NewNotesModel()
	if err != nil {
		fmt.Printf("Unable to load notes log: %v", err)
	}
	tm, err := NewTerminalModel()
	if err != nil {
		fmt.Printf("Unable to load terminal log: %v", err)
	}
	lm, err := NewLogsModel()
	if err != nil {
		fmt.Printf("Unable to open log file: %v", err)
	}

	return model{
		state:         logsView,
		runCmd:        runCmd,
		notesModel:    nm,
		terminalModel: tm,
		logsModel:     lm,
		serversModel:  NewServersModel(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) Close() {
	m.notesModel.Close()
	m.terminalModel.Close()
	m.logsModel.Close()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	// Update components
	m.noteTextinput, cmd = m.noteTextinput.Update(msg)
	cmds = append(cmds, cmd)

	m.noteTextarea, cmd = m.noteTextarea.Update(msg)
	cmds = append(cmds, cmd)

	m.termTextinput, cmd = m.termTextinput.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case communication.BitburnerConnected:
		m.conn = msg.Conn

	case communication.BitburnerDisconnected:
		m.conn = nil

	case *world.World:
		m.world = msg

	case terminalResultMsg:
		return m, m.handleTerminalResult(msg)

	case logger.LogMsg:
		visibleLines := m.logBodyHeight()
		atBottom := m.logSelected >= len(m.logs)-1

		m.logsModel.AppendLog(msg.Entry)

		if atBottom {
			m.logSelected = len(m.logs) - 1
			m.logOffset = max(0, len(m.logs)-visibleLines)
		}

	case tea.KeyMsg:
		key := msg.String()

		// Open/close terminal with ctrl+t
		if key == "ctrl+t" {
			if m.state == terminalView {
				m.stateStack, m.state = popState(m.stateStack)
				m.termTextinput.Blur()
			} else {
				m.termTextinput.SetValue("")
				m.terminalCmd = ""
				m.terminalOutput = ""
				m.termTextinput.Focus()
				m.stateStack, m.state = pushState(m.stateStack, m.state, terminalView)
			}
			return m, tea.Batch(cmds...)
		}

		canTab := m.canTab()
		// Tab switch
		if key == "tab" && canTab {
			switch m.state {
			case logsView:
				m.state = noteListView
			case noteListView:
				m.state = serversView
			default:
				m.state = logsView
			}
			return m, tea.Batch(cmds...)
		}

		// Quit — blocked while editing
		if key == "q" && canTab {
			return m, tea.Quit
		}

		switch m.state {
		case logsView:
			m.handleLogsKey(key)
		case logDetailView:
			m.handleLogsDetailKey(key)
		case noteListView:
			m.handleNoteListKey(key)

		case noteTitleView:
			m.handleNoteTitleKey(key)

		case noteBodyView:
			if cmd = m.handleNoteBodyKey(key); cmd != nil {
				return m, cmd
			}

		case serversView:
			m.handleServersKey(key)

		case serverDetailView:
			m.handleServerDetailKey(key)

		case terminalView:
			if cmd = m.handleTerminalKey(key); cmd != nil {
				return m, cmd
			}
		}
	}
	return m, tea.Batch(cmds...)
}
