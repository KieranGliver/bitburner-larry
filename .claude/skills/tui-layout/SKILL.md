---
name: tui-layout
description: >
  Build and modify TUI (terminal user interface) layouts using Charmbracelet's BubbleTea, Lipgloss,
  and Bubbles libraries (v2, charm.land/* import paths). Use this skill whenever the user wants to
  add or change how their TUI looks: add a sidebar, split the screen, add a status bar or header,
  create a modal, compose multiple panes, wire up a new Bubbles component, or make the layout
  respond to the terminal size. Also trigger for questions about the MVU architecture, state machine
  view management, or how to structure a model that has multiple views. If the user is working on
  any terminal app using bubbletea, lipgloss, or bubbles — even if they don't say "layout" — use
  this skill.
---

# TUI Layout with BubbleTea, Lipgloss, and Bubbles (v2)

This project uses the v2 Charmbracelet stack:

```go
import (
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
    "charm.land/bubbles/v2/<component>"
)
```

## Architecture: Model-View-Update (MVU)

Every BubbleTea app has one root model implementing three methods:

```go
func (m model) Init() tea.Cmd            // startup side effects
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd)  // handle events → new state
func (m model) View() tea.View           // pure render of current state
```

`tea.View` (v2) wraps a string — return `tea.NewView(s)` where `s` is your rendered layout string.

**The golden rule:** `View()` is a pure function. Never store render state; everything comes from `m`.

---

## Managing Multiple Views: State Machine Pattern

The cleanest way to handle multiple "screens" or "modes" is a state enum plus a switch in `View()` and `Update()`.

```go
const (
    listView uint = iota
    detailView
    editView
)

type model struct {
    state uint
    // ...
}

func (m model) View() tea.View {
    switch m.state {
    case listView:
        return tea.NewView(m.renderList())
    case detailView:
        return tea.NewView(m.renderDetail())
    case editView:
        return tea.NewView(m.renderEdit())
    }
    return tea.NewView("")
}
```

For complex apps, split each view into its own `render<Name>() string` method to keep `View()` readable.

---

## Responsive Layout: Tracking Window Size

Handle `tea.WindowSizeMsg` to make your layout adapt to the terminal:

```go
type model struct {
    width  int
    height int
    // ...
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        // Propagate to any Bubbles components that need it:
        m.viewport.Width = msg.Width - sidebarWidth
        m.viewport.Height = msg.Height - headerHeight
    }
    // ...
}
```

Always store `width` and `height` on the model — they're needed for computing pane sizes in `View()`.

---

## Lipgloss Layout Composition

Lipgloss builds layouts by composing styled strings. The two key joining functions:

```go
// Stack vertically (top to bottom)
lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

// Place side by side
lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent)
```

Alignment options: `lipgloss.Left`, `lipgloss.Center`, `lipgloss.Right`, `lipgloss.Top`, `lipgloss.Bottom`

### Common Layout Patterns

**Header + content + footer:**
```go
func (m model) renderBase(content string) string {
    header := headerStyle.Width(m.width).Render("MY APP")
    footer := footerStyle.Width(m.width).Render("q quit • ? help")
    // content area height = total - header - footer
    body := bodyStyle.
        Width(m.width).
        Height(m.height - lipgloss.Height(header) - lipgloss.Height(footer)).
        Render(content)
    return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
```

**Sidebar + main content:**
```go
const sidebarWidth = 24

func (m model) renderSplit(sidebar, main string) string {
    s := sidebarStyle.
        Width(sidebarWidth).
        Height(m.height).
        Render(sidebar)
    c := mainStyle.
        Width(m.width - sidebarWidth).
        Height(m.height).
        Render(main)
    return lipgloss.JoinHorizontal(lipgloss.Top, s, c)
}
```

**Centered modal overlay:**
```go
func (m model) renderModal(content string) string {
    box := modalStyle.
        Width(60).
        Border(lipgloss.RoundedBorder()).
        Padding(1, 2).
        Render(content)
    // Place centers the box within the full terminal area
    return lipgloss.Place(m.width, m.height,
        lipgloss.Center, lipgloss.Center, box)
}
```

**Status bar at the bottom:**
```go
func (m model) renderStatusBar() string {
    mode := modeStyle.Render(m.modeName())
    hint := hintStyle.Render(m.keyHints())
    // Push hint to the right
    gap := strings.Repeat(" ", max(0, m.width - lipgloss.Width(mode) - lipgloss.Width(hint)))
    return statusBarStyle.Width(m.width).Render(mode + gap + hint)
}
```

### Style Building

Define styles as package-level vars. Keep styling out of `View()` logic:

```go
var (
    headerStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("62")).
        Foreground(lipgloss.Color("230")).
        Padding(0, 1).
        Bold(true)

    sidebarStyle = lipgloss.NewStyle().
        Border(lipgloss.NormalBorder(), false, true, false, false). // right border only
        BorderForeground(lipgloss.Color("240"))

    focusedBorderStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62"))

    blurredBorderStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("240"))
)
```

Use `lipgloss.Width(s)` and `lipgloss.Height(s)` to measure rendered strings — these account for ANSI escape codes, which `len()` does not.

---

## Embedding Bubbles Components

Bubbles components follow the same MVU contract as your app. Embed them in your model and always update them before your own switch:

```go
type model struct {
    list      list.Model       // charm.land/bubbles/v2/list
    viewport  viewport.Model   // charm.land/bubbles/v2/viewport
    textinput textinput.Model  // charm.land/bubbles/v2/textinput
    textarea  textarea.Model   // charm.land/bubbles/v2/textarea
    spinner   spinner.Model    // charm.land/bubbles/v2/spinner
    // ...
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd

    // Always update components — even when they're not the active view,
    // so they stay in sync (e.g. spinner keeps ticking)
    m.list, cmd = m.list.Update(msg)
    cmds = append(cmds, cmd)
    m.viewport, cmd = m.viewport.Update(msg)
    cmds = append(cmds, cmd)

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // your own key handling
    }
    return m, tea.Batch(cmds...)
}
```

**Render a component** by calling `.View()` on it and using the string in your layout:
```go
content := m.viewport.View()
return tea.NewView(m.renderBase(content))
```

### Focusing Components

Most Bubbles components support `Focus()`/`Blur()` to show a visual focus indicator and enable typing:

```go
case "tab":
    if m.focused == inputField {
        m.textinput.Blur()
        m.textarea.Focus()
        m.focused = bodyField
    } else {
        m.textarea.Blur()
        m.textinput.Focus()
        m.focused = inputField
    }
```

Use focused/blurred border styles to give clear visual feedback about which pane is active.

### Sizing Components to Fit the Layout

Set sizes on components after computing your layout dimensions — not at init time:

```go
case tea.WindowSizeMsg:
    m.width, m.height = msg.Width, msg.Height
    m.viewport.Width = m.width - sidebarWidth - 2  // -2 for borders
    m.viewport.Height = m.height - headerHeight - footerHeight - 2
    m.list.SetSize(sidebarWidth-2, m.height-headerHeight)
```

---

## Common Bubbles Components Quick Reference

See `references/bubbles.md` for initialization examples and full API notes.

| Component | Import suffix | Use for |
|-----------|--------------|---------|
| `list` | `list` | Scrollable item list with filtering |
| `table` | `table` | Tabular data with row selection |
| `viewport` | `viewport` | Scrollable read-only text/content |
| `textarea` | `textarea` | Multi-line editable text |
| `textinput` | `textinput` | Single-line input with placeholder |
| `spinner` | `spinner` | Animated loading indicator |
| `progress` | `progress` | Progress bar |
| `filepicker` | `filepicker` | File system navigation |

---

## Multiple Focused Panes (Split-View with Active Pane)

Track which pane is active and style it differently:

```go
type pane uint
const (
    leftPane pane = iota
    rightPane
)

type model struct {
    activePane pane
    // ...
}

func (m model) renderLeft() string {
    style := blurredPaneStyle
    if m.activePane == leftPane {
        style = focusedPaneStyle
    }
    return style.Width(m.leftWidth()).Height(m.height).Render(m.list.View())
}
```

Only forward key messages to the active component to avoid double-handling:

```go
case tea.KeyMsg:
    switch msg.String() {
    case "tab":
        m.togglePane()
    default:
        if m.activePane == leftPane {
            m.list, cmd = m.list.Update(msg)
        } else {
            m.viewport, cmd = m.viewport.Update(msg)
        }
        cmds = append(cmds, cmd)
    }
```

---

## Useful Utilities

```go
// Clamp a value between min and max
func clamp(v, lo, hi int) int {
    if v < lo { return lo }
    if v > hi { return hi }
    return v
}

// Truncate a string to fit a width (accounts for multi-byte runes)
func truncate(s string, w int) string {
    if lipgloss.Width(s) <= w { return s }
    return s[:w-1] + "…"
}

// Fill remaining width with a repeated character
func fill(width int, ch string) string {
    return strings.Repeat(ch, max(0, width))
}
```

---

## Reference Files

- `references/bubbles.md` — Initialization and API examples for each Bubbles component
- `references/layouts.md` — Ready-to-use layout skeletons for common TUI structures
