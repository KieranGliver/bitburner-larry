package main

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type focusedPane int

const (
	leftPane focusedPane = iota
	rightPane
)

type serverItem struct {
	name, status, region, uptime string
}

func (s serverItem) Title() string       { return s.name }
func (s serverItem) Description() string { return s.status }
func (s serverItem) FilterValue() string { return s.name }

var (
	focusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("135"))

	blurredPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("135")).MarginBottom(1)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(10)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginTop(1)
)

const leftPanelWidth = 25

type model struct {
	width, height int
	focus         focusedPane
	list          list.Model
}

func newModel() model {
	servers := []list.Item{
		serverItem{"web-01", "running", "us-east-1", "14d 3h"},
		serverItem{"web-02", "running", "us-east-1", "14d 3h"},
		serverItem{"db-primary", "running", "us-west-2", "30d 12h"},
		serverItem{"db-replica", "stopped", "us-west-2", "0d 0h"},
		serverItem{"cache-01", "running", "eu-west-1", "7d 22h"},
		serverItem{"worker-01", "running", "us-east-1", "3d 8h"},
		serverItem{"worker-02", "starting", "us-east-1", "0d 0h"},
	}
	l := list.New(servers, list.NewDefaultDelegate(), leftPanelWidth-2, 20)
	l.Title = "Servers"
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	return model{focus: leftPane, list: l}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(leftPanelWidth-2, m.height-2)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus == leftPane {
				m.focus = rightPane
			} else {
				m.focus = leftPane
			}
		default:
			if m.focus == leftPane {
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			}
			// right pane is read-only detail view — no key routing needed
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading…")
	}
	return tea.NewView(m.renderSplit())
}

func (m model) paneStyle(p focusedPane) lipgloss.Style {
	if m.focus == p {
		return focusedPaneStyle
	}
	return blurredPaneStyle
}

func (m model) renderSplit() string {
	leftInnerW := leftPanelWidth - 2  // subtract border
	leftInnerH := m.height - 2
	rightInnerW := m.width - leftPanelWidth - 2
	rightInnerH := m.height - 2

	m.list.SetSize(leftInnerW, leftInnerH)
	left := m.paneStyle(leftPane).Width(leftInnerW).Height(leftInnerH).Render(m.list.View())
	right := m.paneStyle(rightPane).Width(rightInnerW).Height(rightInnerH).Render(m.renderDetail())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m model) renderDetail() string {
	sel, ok := m.list.SelectedItem().(serverItem)
	if !ok {
		return "No server selected."
	}

	row := func(label, value string) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), valueStyle.Render(value))
	}

	statusColor := lipgloss.Color("82")
	switch sel.status {
	case "stopped":
		statusColor = lipgloss.Color("196")
	case "starting":
		statusColor = lipgloss.Color("214")
	}

	return fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n%s",
		titleStyle.Render(sel.name),
		row("Status", lipgloss.NewStyle().Foreground(statusColor).Render(sel.status)),
		row("Region", sel.region),
		row("Uptime", sel.uptime),
		helpStyle.Render("tab: switch pane  •  ↑↓: navigate list  •  q: quit"),
	)
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
