package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/internal/app"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/db"
	"github.com/KieranGliver/bitburner-larry/internal/filesync"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/KieranGliver/bitburner-larry/internal/mcpserver"
	"github.com/KieranGliver/bitburner-larry/internal/tui"
)

func main() {
	store := &db.Store{}

	if err := store.Init(); err != nil {
		fmt.Printf("Can't init store: %v", err)
	}

	m := tui.NewModel(store)

	p := tea.NewProgram(m)

	app := &app.App{P: p}
	app.Start()

	onCall := func(input, result string) {
		p.Send(logger.InfoDetail(fmt.Sprintf("[mcp] %s", input), result))
	}

	mcpSrv := mcpserver.New(onCall)

	go communication.Serve("12525", p, func(conn *communication.BitburnerConn) {
		mcpSrv.SetConn(conn)
		app.OnConnect(conn)
	})
	go mcpSrv.Serve("12526")
	go filesync.Watch("scripts/dist", p, app.OnEventDist)
	go filesync.Watch("scripts/src", p, app.OnEventSrc)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Unable to run tui: %v", err)
		os.Exit(1)
	}
}
