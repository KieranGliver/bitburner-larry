package main

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type pane int

const (
	listPaneFocus pane = iota
	detailPaneFocus
)

type server struct {
	name   string
	status string
	region string
}

func (s server) Title() string       { return s.name }
func (s server) Description() string { return s.status }
func (s server) FilterValue() string { return s.name }

type model struct {
	width      int
	height     int
	activePane pane
	serverList list.Model
}

var (
	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99"))

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
)

func newModel() model {
	items := []list.Item{
		server{"web-01", "running", "us-east-1"},
		server{"web-02", "running", "us-east-1"},
		server{"db-primary", "running", "us-west-2"},
		server{"db-replica", "stopped", "us-west-2"},
		server{"cache-01", "running", "eu-west-1"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 23, 10)
	l.Title = "Servers"
	l.SetShowHelp(false)
	return model{activePane: listPaneFocus, serverList: l}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.serverList.SetSize(23, m.height-4)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.activePane == listPaneFocus {
				m.activePane = detailPaneFocus
			} else {
				m.activePane = listPaneFocus
			}
		default:
			// route all keys to active pane — but only list pane is interactive
			if m.activePane == listPaneFocus {
				m.serverList, cmd = m.serverList.Update(msg)
			}
		}
	}
	return m, cmd
}

func (m model) View() tea.View {
	const leftW = 25

	listStyle := inactiveBorderStyle
	detailStyle := inactiveBorderStyle
	if m.activePane == listPaneFocus {
		listStyle = activeBorderStyle
	} else {
		detailStyle = activeBorderStyle
	}

	rightW := m.width - leftW
	if rightW < 10 {
		rightW = 10
	}

	leftPanel := listStyle.Width(leftW - 2).Height(m.height - 2).Render(m.serverList.View())

	var detail string
	if sel, ok := m.serverList.SelectedItem().(server); ok {
		detail = fmt.Sprintf("Server: %s\nStatus: %s\nRegion: %s\n\ntab: switch focus\nq: quit",
			sel.name, sel.status, sel.region)
	} else {
		detail = "Select a server"
	}
	rightPanel := detailStyle.Width(rightW - 2).Height(m.height - 2).Render(detail)

	return tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel))
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
	}
}
