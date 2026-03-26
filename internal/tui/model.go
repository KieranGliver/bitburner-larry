package tui

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/db"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
)

const (
	logsView uint = iota
	listView
	titleView
	bodyView
	terminalView
)

const maxLogs = 500

type model struct {
	state     uint
	prevState uint
	width     int
	height    int
	store     *db.Store
	notes     []db.Note
	currNote  db.Note
	listIndex int
	textarea  textarea.Model
	textinput textinput.Model
	termInput textinput.Model
	conn      *communication.BitburnerConn
	logs      []logger.LogEntry
	logOffset int
	logFile   *os.File
}

func NewModel(store *db.Store) model {
	notes, err := store.GetNotes()
	if err != nil {
		fmt.Printf("Unable to get notes: %v", err)
	}
	logFile, _ := os.OpenFile("larry.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	return model{
		state:     logsView,
		store:     store,
		notes:     notes,
		textarea:  textarea.New(),
		textinput: textinput.New(),
		termInput: textinput.New(),
		logFile:   logFile,
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

	case logger.LogMsg:
		// auto-scroll to bottom if already at bottom
		visibleLines := m.logBodyHeight()
		atBottom := m.logOffset >= len(m.logs)-visibleLines

		m.logs = append(m.logs, msg.Entry)
		if len(m.logs) > maxLogs {
			m.logs = m.logs[len(m.logs)-maxLogs:]
		}

		if atBottom {
			m.logOffset = max(0, len(m.logs)-visibleLines)
		}

		if m.logFile != nil {
			fmt.Fprintf(m.logFile, "%s  %s  %s\n", msg.Entry.Time.Format("15:04:05"), msg.Entry.Level, msg.Entry.Message)
		}

	case tea.KeyMsg:
		key := msg.String()

		// Open/close terminal with ctrl+t
		if key == "ctrl+t" {
			if m.state == terminalView {
				m.state = m.prevState
				m.termInput.Blur()
				m.termInput.SetValue("")
			} else {
				m.prevState = m.state
				m.state = terminalView
				m.termInput.Focus()
			}
			return m, tea.Batch(cmds...)
		}

		// Tab switch — blocked while editing a note
		if key == "tab" && m.state != titleView && m.state != bodyView && m.state != terminalView {
			if m.state == logsView {
				m.state = listView
			} else {
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
			visibleLines := m.logBodyHeight()
			maxOffset := max(0, len(m.logs)-visibleLines)
			switch key {
			case "up", "k":
				if m.logOffset > 0 {
					m.logOffset--
				}
			case "down", "j":
				if m.logOffset < maxOffset {
					m.logOffset++
				}
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

		case terminalView:
			switch key {
			case "ctrl+c":
				m.state = m.prevState
				m.termInput.Blur()
				m.termInput.SetValue("")
			case "enter":
				cmdVal := m.termInput.Value()
				m.termInput.Blur()
				m.termInput.SetValue("")
				m.state = m.prevState
				if cmdVal != "" {
					return m, func() tea.Msg {
						// TODO: Write command logic here
						return logger.Info(fmt.Sprintf(`[command] Ran command "%s"`, cmdVal))
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
