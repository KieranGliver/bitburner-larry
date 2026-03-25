package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/communication"
	"github.com/KieranGliver/bitburner-larry/db"
	"github.com/KieranGliver/bitburner-larry/tui"
)

func main() {
	store := &db.Store{}

	if err := store.Init(); err != nil {
		fmt.Printf("Can't init store: %v", err)
	}

	m := tui.NewModel(store)

	p := tea.NewProgram(m)

	go communication.Serve("12525", p)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Unable to run tui: %v", err)
		os.Exit(1)
	}
}
