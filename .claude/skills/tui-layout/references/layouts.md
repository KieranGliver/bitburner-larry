# TUI Layout Skeletons

Ready-to-adapt patterns for common terminal UI structures.

---

## 1. Full-Screen App Shell (Header + Body + Status Bar)

```go
var (
    headerStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("99")).
        Foreground(lipgloss.Color("230")).
        Padding(0, 1).Width(0) // width set at render time

    statusBarStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("235")).
        Foreground(lipgloss.Color("245")).
        Padding(0, 1)

    bodyStyle = lipgloss.NewStyle().Padding(1, 2)
)

func (m model) View() tea.View {
    header := headerStyle.Width(m.width).Render("APP NAME")
    status := statusBarStyle.Width(m.width).Render(m.statusText())
    headerH := lipgloss.Height(header)
    statusH := lipgloss.Height(status)

    body := bodyStyle.
        Width(m.width).
        Height(m.height - headerH - statusH).
        Render(m.renderContent())

    return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body, status))
}
```

---

## 2. Sidebar + Main Content (Two-Column)

```go
const sidebarW = 28

var (
    sidebarStyle = lipgloss.NewStyle().
        Border(lipgloss.NormalBorder(), false, true, false, false).
        BorderForeground(lipgloss.Color("240")).
        Padding(1, 1)

    mainStyle = lipgloss.NewStyle().Padding(1, 2)
)

func (m model) renderSplit() string {
    sidebar := sidebarStyle.
        Width(sidebarW).
        Height(m.height).
        Render(m.renderSidebar())

    main := mainStyle.
        Width(m.width - sidebarW - 1). // -1 for border
        Height(m.height).
        Render(m.renderMain())

    return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}
```

---

## 3. Three-Pane Layout (Sidebar + Content + Inspector)

```go
const (
    leftW  = 20
    rightW = 30
)

func (m model) renderThreePane() string {
    midW := m.width - leftW - rightW

    left := leftPaneStyle.Width(leftW).Height(m.height).Render(m.renderLeft())
    mid  := midPaneStyle.Width(midW).Height(m.height).Render(m.renderMid())
    right := rightPaneStyle.Width(rightW).Height(m.height).Render(m.renderRight())

    return lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
}
```

---

## 4. Centered Modal Over Background

```go
var (
    overlayStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("0")).
        Foreground(lipgloss.Color("240"))

    modalStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("235")).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("99")).
        Padding(1, 3)
)

func (m model) renderWithModal(background, content string) string {
    // Dim the background (optional: just render it normally)
    bg := lipgloss.NewStyle().
        Width(m.width).Height(m.height).
        Render(background)

    modal := modalStyle.Width(min(60, m.width-4)).Render(content)

    // Place the modal centered over the background
    return lipgloss.Place(m.width, m.height,
        lipgloss.Center, lipgloss.Center, modal,
        lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
    )
}
```

---

## 5. Tabbed Layout

```go
type tab uint
const (
    overviewTab tab = iota
    detailTab
    settingsTab
)

var tabNames = []string{"Overview", "Detail", "Settings"}

var (
    activeTabStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder(), true, true, false, true).
        BorderForeground(lipgloss.Color("99")).
        Padding(0, 1)

    inactiveTabStyle = lipgloss.NewStyle().
        Border(lipgloss.NormalBorder(), false, false, true, false).
        BorderForeground(lipgloss.Color("240")).
        Foreground(lipgloss.Color("240")).
        Padding(0, 1)

    tabGapStyle = lipgloss.NewStyle().
        Border(lipgloss.NormalBorder(), false, false, true, false).
        BorderForeground(lipgloss.Color("240"))
)

func (m model) renderTabBar() string {
    tabs := make([]string, len(tabNames))
    for i, name := range tabNames {
        if tab(i) == m.activeTab {
            tabs[i] = activeTabStyle.Render(name)
        } else {
            tabs[i] = inactiveTabStyle.Render(name)
        }
    }
    tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
    // Fill remaining width with the bottom border
    gap := tabGapStyle.Width(m.width - lipgloss.Width(tabBar)).Render("")
    return lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, gap)
}

func (m model) View() tea.View {
    tabBar := m.renderTabBar()
    var content string
    switch m.activeTab {
    case overviewTab:
        content = m.renderOverview()
    case detailTab:
        content = m.renderDetail()
    case settingsTab:
        content = m.renderSettings()
    }
    return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, tabBar, content))
}
```

---

## 6. Split View with Focus Indicator

```go
type focusedPane int
const (
    leftFocus focusedPane = iota
    rightFocus
)

var (
    focusedPaneStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("99"))

    blurredPaneStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("240"))
)

func (m model) paneStyle(p focusedPane) lipgloss.Style {
    if m.focus == p {
        return focusedPaneStyle
    }
    return blurredPaneStyle
}

func (m model) renderSplitFocused() string {
    halfW := m.width / 2
    left := m.paneStyle(leftFocus).
        Width(halfW - 2).Height(m.height - 2). // -2 for borders
        Render(m.list.View())
    right := m.paneStyle(rightFocus).
        Width(m.width - halfW - 2).Height(m.height - 2).
        Render(m.viewport.View())
    return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// In Update(): only route keys to the focused component
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            if m.focus == leftFocus {
                m.focus = rightFocus
            } else {
                m.focus = leftFocus
            }
        default:
            if m.focus == leftFocus {
                m.list, cmd = m.list.Update(msg)
            } else {
                m.viewport, cmd = m.viewport.Update(msg)
            }
            cmds = append(cmds, cmd)
        }
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        half := m.width / 2
        m.list.SetSize(half-4, m.height-4)
        m.viewport.Width = m.width - half - 4
        m.viewport.Height = m.height - 4
    }
    return m, tea.Batch(cmds...)
}
```

---

## Helpful Utilities

```go
func min(a, b int) int {
    if a < b { return a }
    return b
}

func max(a, b int) int {
    if a > b { return a }
    return b
}

func clamp(v, lo, hi int) int {
    if v < lo { return lo }
    if v > hi { return hi }
    return v
}
```
