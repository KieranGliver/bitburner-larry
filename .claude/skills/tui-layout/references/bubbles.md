# Bubbles Component Reference (v2)

All imports are under `charm.land/bubbles/v2/<component>`.

---

## list

A scrollable, filterable list of items.

```go
import "charm.land/bubbles/v2/list"

// Define your item type
type item struct {
    title, desc string
}
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Initialize
items := []list.Item{
    item{"First", "subtitle"},
    item{"Second", "subtitle"},
}
l := list.New(items, list.NewDefaultDelegate(), width, height)
l.Title = "My List"
l.SetShowHelp(false) // hide the built-in help bar if you have your own footer

// In Update()
m.list, cmd = m.list.Update(msg)

// Get selected item
if sel, ok := m.list.SelectedItem().(item); ok {
    fmt.Println(sel.title)
}

// In View()
return m.list.View()
```

Key settings:
- `l.SetSize(w, h)` — resize
- `l.SetFilteringEnabled(false)` — disable '/' filter
- `l.SetShowStatusBar(false)` — hide count/filter status
- `l.Styles.Title` — style the title bar
- Custom delegates via `list.NewDefaultDelegate()` + `d.Styles.*`

---

## table

Tabular data with keyboard row selection.

```go
import "charm.land/bubbles/v2/table"

cols := []table.Column{
    {Title: "Name", Width: 20},
    {Title: "Status", Width: 10},
    {Title: "Score", Width: 8},
}
rows := []table.Row{
    {"Alice", "active", "92"},
    {"Bob", "idle", "77"},
}
t := table.New(
    table.WithColumns(cols),
    table.WithRows(rows),
    table.WithFocused(true),
    table.WithHeight(10),
)
t.SetStyles(table.DefaultStyles()) // or customize

// In Update()
m.table, cmd = m.table.Update(msg)

// Selected row
row := m.table.SelectedRow() // []string

// In View()
return m.table.View()
```

---

## viewport

A scrollable read-only pane. Great for displaying long content, logs, previews.

```go
import "charm.land/bubbles/v2/viewport"

vp := viewport.New(width, height)
vp.SetContent(someString)          // set the full content
vp.GotoTop()                        // or GotoBottom()

// In Update()
m.viewport, cmd = m.viewport.Update(msg)
// viewport handles PageUp/PageDown/Up/Down automatically

// In View()
return m.viewport.View()

// Check scroll position
m.viewport.AtBottom()     // bool
m.viewport.ScrollPercent() // float64 0..1
```

---

## textarea

Multi-line, editable text input.

```go
import "charm.land/bubbles/v2/textarea"

ta := textarea.New()
ta.Placeholder = "Write your note…"
ta.SetWidth(60)
ta.SetHeight(10)
ta.Focus()

// In Update()
m.textarea, cmd = m.textarea.Update(msg)

// Get/set value
ta.SetValue("initial content")
body := m.textarea.Value()

// Focus management
ta.Focus()
ta.Blur()
```

---

## textinput

Single-line input with placeholder, masking, and completion.

```go
import "charm.land/bubbles/v2/textinput"

ti := textinput.New()
ti.Placeholder = "Enter title…"
ti.CharLimit = 80
ti.Width = 40
ti.Focus()

// For passwords
ti.EchoMode = textinput.EchoPassword

// In Update()
m.textinput, cmd = m.textinput.Update(msg)

// Value
val := m.textinput.Value()
m.textinput.SetValue("")     // clear
m.textinput.CursorEnd()     // move cursor to end
```

---

## spinner

Animated loading indicator. Requires a `tea.Cmd` to keep ticking.

```go
import "charm.land/bubbles/v2/spinner"

s := spinner.New()
s.Spinner = spinner.Dot     // Dot, Line, MiniDot, Jump, Pulse, Points, Globe, Moon, Monkey
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

// Start the spinner — return this Cmd from Init() or Update()
cmd = m.spinner.Tick

// In Update()
case spinner.TickMsg:
    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)

// In View()
return m.spinner.View() + " Loading…"
```

---

## progress

Animated progress bar.

```go
import "charm.land/bubbles/v2/progress"

p := progress.New(
    progress.WithDefaultGradient(),       // or WithSolidFill("62")
    progress.WithWidth(40),
    progress.WithoutPercentage(),         // hide "50%" label
)

// Set percentage (0.0 – 1.0)
cmd = m.progress.SetPercent(0.5)         // animated
// or instantly:
m.progress.SetPercent(0.5)

// In Update()
case progress.FrameMsg:
    progressModel, cmd := m.progress.Update(msg)
    m.progress = progressModel.(progress.Model)
    cmds = append(cmds, cmd)

// In View()
return m.progress.View()
```

---

## filepicker

File system navigation for selecting files or directories.

```go
import "charm.land/bubbles/v2/filepicker"

fp := filepicker.New()
fp.CurrentDirectory, _ = os.UserHomeDir()
fp.AllowedTypes = []string{".go", ".md"}
fp.Height = 20

// Init — required to load directory
initCmd = m.filepicker.Init()

// In Update()
m.filepicker, cmd = m.filepicker.Update(msg)
cmds = append(cmds, cmd)

// Check for selection
if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
    m.selectedFile = path
}
```
