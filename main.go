package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	larcmd "github.com/KieranGliver/bitburner-larry/cmd"
	"github.com/KieranGliver/bitburner-larry/internal/app"
	"github.com/KieranGliver/bitburner-larry/internal/brain"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/db"
	"github.com/KieranGliver/bitburner-larry/internal/filesync"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/KieranGliver/bitburner-larry/internal/mcp"
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

	var scanCancel context.CancelFunc

	onColReady := func(conn *communication.BitburnerConn) {
		if scanCancel != nil {
			scanCancel()
		}
		var scanCtx context.Context
		scanCtx, scanCancel = context.WithCancel(context.Background())
		if cracked, _, err := larcmd.DoCrack(conn); err != nil {
			p.Send(logger.Warn("crack: " + err.Error()))
		} else if len(cracked) > 0 {
			p.Send(logger.Info(fmt.Sprintf("cracked %d servers: %v", len(cracked), cracked)))
		}
		if w, err := larcmd.DoScan(conn); err != nil {
			p.Send(logger.Warn("initial scan: " + err.Error()))
		} else {
			p.Send(w)
		}
		larcmd.RunScanner(conn, scanCtx, 5*time.Second, func(w *brain.World) {
			p.Send(w)
		})
	}

	go communication.Serve("12525", p, func(conn *communication.BitburnerConn) {
		if scanCancel != nil {
			scanCancel()
			scanCancel = nil
		}
		mcpSrv.SetConn(conn)
		app.OnConnect(conn)
	}, onColReady)
	go mcpSrv.Serve("12526")
	go filesync.Watch("scripts/dist", p, app.OnEventDist)
	go filesync.Watch("scripts/src", p, app.OnEventSrc)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Unable to run tui: %v", err)
		os.Exit(1)
	}
}
