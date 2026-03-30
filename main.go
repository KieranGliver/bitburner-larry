package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	larcmd "github.com/KieranGliver/bitburner-larry/cmd"
	"github.com/KieranGliver/bitburner-larry/internal/app"
	col "github.com/KieranGliver/bitburner-larry/internal/col"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/filesync"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	mcpserver "github.com/KieranGliver/bitburner-larry/internal/mcp"
	"github.com/KieranGliver/bitburner-larry/internal/tui"
	"github.com/KieranGliver/bitburner-larry/internal/world"
)

func main() {

	appState := &app.AppState{}

	runCmd := func(input string) string {
		return larcmd.ExecuteCommand(input, appState)
	}

	m := tui.NewModel(runCmd)
	defer m.Close()

	p := tea.NewProgram(m)

	appState.SetSend(p.Send)
	app := &app.App{P: p, State: appState}

	app.Start()

	var scanCancel context.CancelFunc

	onColReady := func(conn *communication.BitburnerConn) {
		if scanCancel != nil {
			scanCancel()
		}
		var scanCtx context.Context
		scanCtx, scanCancel = context.WithCancel(context.Background())
		if cracked, _, err := col.DoCrack(conn, nil); err != nil {
			p.Send(logger.Warn("crack: " + err.Error()))
		} else if len(cracked) > 0 {
			p.Send(logger.Info(fmt.Sprintf("cracked %d servers: %v", len(cracked), cracked)))
		}
		if w, err := col.DoScan(conn, ""); err != nil {
			p.Send(logger.Warn("initial scan: " + err.Error()))
		} else {
			appState.SetWorld(w)
		}
		col.RunScanner(conn, scanCtx, 5*time.Second, func(w *world.World) {
			appState.SetWorld(w)
		})
	}

	go communication.Serve("12525", p, func(conn *communication.BitburnerConn) {
		if scanCancel != nil {
			scanCancel()
			scanCancel = nil
		}
		app.OnConnect(conn)
	}, func() {
		appState.SetConn(nil)
	}, onColReady)

	onCall := func(input, result string) {
		p.Send(logger.InfoDetail(fmt.Sprintf("[mcp] %s", input), result))
	}
	mcpSrv := mcpserver.New(onCall, appState)
	go mcpSrv.Serve("12526")

	onError := func(s string) {
		p.Send(logger.Error(s))
	}
	go filesync.Watch("scripts/dist", onError, app.OnEventDist)
	go filesync.Watch("scripts/src", onError, app.OnEventSrc)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Unable to run tui: %v", err)
		os.Exit(1)
	}
}
