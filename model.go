package main

import (
	"fmt"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	listView uint = iota
	titleView
	bodyView
)

type model struct {
	state     uint
	store     *Store
	notes     []Note
	currNote  Note
	listIndex int
	textarea  textarea.Model
	textinput textinput.Model
}

func newModel(store *Store) model {
	notes, err := store.GetNotes()
	if err != nil {
		fmt.Printf("Unable to get notes: %v", err)
	}
	return model{
		state:     listView,
		store:     store,
		notes:     notes,
		textarea:  textarea.New(),
		textinput: textinput.New(),
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch m.state {
		case listView:
			switch key {
			case "q":
				return m, tea.Quit
			case "n":
				m.textinput.SetValue("")
				m.textinput.Focus()
				m.currNote = Note{}
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
				m.currNote = m.notes[m.listIndex]
				m.textarea.SetValue(m.currNote.Body)
				m.textarea.Focus()
				m.textarea.CursorEnd()
				m.state = bodyView
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

				m.currNote = Note{}
				m.state = listView
			case "esc":
				m.state = listView
			}

		}
	}
	return m, tea.Batch(cmds...)
}
