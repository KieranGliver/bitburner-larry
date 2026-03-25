package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	listView uint = iota
	titleView
	bodyView
	confirmModal // new state: confirm delete dialog
)

var (
	faintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	enumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Background(lipgloss.Color("235")).
			Padding(1, 3).
			Width(40)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			MarginBottom(1)

	modalHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginTop(1)
)

type model struct {
	state     uint
	width     int
	height    int
	notes     []string // simplified
	listIndex int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch m.state {
		case listView:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "up", "k":
				if m.listIndex > 0 {
					m.listIndex--
				}
			case "down", "j":
				if m.listIndex < len(m.notes)-1 {
					m.listIndex++
				}
			case "d":
				// Only open modal if there's something to delete
				if len(m.notes) > 0 {
					m.state = confirmModal
				}
			}

		case confirmModal:
			switch msg.String() {
			case "y", "Y":
				// Delete the selected note
				if m.listIndex < len(m.notes) {
					m.notes = append(m.notes[:m.listIndex], m.notes[m.listIndex+1:]...)
					if m.listIndex > 0 && m.listIndex >= len(m.notes) {
						m.listIndex = len(m.notes) - 1
					}
				}
				m.state = listView
			case "n", "N", "esc":
				// Cancel — return to list without deleting
				m.state = listView
			}
			// All other keys are intentionally ignored while modal is open
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	// Always render the list as the base layer
	bg := m.renderList()

	if m.state == confirmModal {
		// Overlay the modal centered on top of the list
		return tea.NewView(m.renderModal(bg))
	}

	return tea.NewView(bg)
}

func (m model) renderList() string {
	var sb strings.Builder
	sb.WriteString("Notes\n\n")
	for i, note := range m.notes {
		prefix := "  "
		if i == m.listIndex {
			prefix = "> "
		}
		sb.WriteString(enumeratorStyle.Render(prefix) + note + "\n")
	}
	if len(m.notes) == 0 {
		sb.WriteString(faintStyle.Render("No notes yet.") + "\n")
	}
	sb.WriteString("\n" + faintStyle.Render("d delete  •  ↑↓/jk navigate  •  q quit"))
	return sb.String()
}

func (m model) renderModal(background string) string {
	var noteName string
	if m.listIndex < len(m.notes) {
		noteName = m.notes[m.listIndex]
	}

	content := modalTitleStyle.Render("Delete this note?") +
		"\n" + faintStyle.Render(noteName) +
		"\n" + modalHintStyle.Render("y confirm  •  n / esc cancel")

	box := modalStyle.Render(content)

	// Place the modal centered over the background.
	// lipgloss.Place fills the full width×height with the background
	// content and positions the box at the center.
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceChars(" "),
	)
}

func main() {
	m := model{
		notes: []string{"Shopping list", "Meeting notes", "Project ideas", "Random thoughts"},
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
