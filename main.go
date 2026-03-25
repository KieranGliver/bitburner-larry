package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/communication"
	"github.com/KieranGliver/bitburner-larry/db"
	"github.com/KieranGliver/bitburner-larry/filesync"
	"github.com/KieranGliver/bitburner-larry/logger"
	"github.com/KieranGliver/bitburner-larry/tui"
)

func onConnect(conn *communication.BitburnerConn, p *tea.Program) {
	ctx := context.Background()

	result, err := conn.Call(ctx, "getDefinitionFile")
	if err != nil {
		p.Send(logger.Error("getDefinitionFile: " + err.Error()))
		return
	}

	// result is a JSON-quoted string — unmarshal to get the raw text
	var content string
	json.Unmarshal(result, &content)

	os.WriteFile("scripts/NetscriptDefinitions.d.ts", []byte(content), 0644)
	p.Send(logger.Info("saved NetscriptDefinitions.d.ts"))
}

func main() {
	store := &db.Store{}

	if err := store.Init(); err != nil {
		fmt.Printf("Can't init store: %v", err)
	}

	m := tui.NewModel(store)

	p := tea.NewProgram(m)

	go communication.Serve("12525", p, onConnect)
	go filesync.Watch("scripts/dist", p)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Unable to run tui: %v", err)
		os.Exit(1)
	}
}
