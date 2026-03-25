package filesync

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/logger"
	"github.com/sgtdi/fswatcher"
)

func Watch(path string, p *tea.Program) {
	// Create watcher with debouncing that watches the current working directory
	w, _ := fswatcher.New(
		fswatcher.WithPath(path),
		fswatcher.WithCooldown(200*time.Millisecond),
	)

	ctx := context.Background()
	go w.Watch(ctx)
	p.Send(logger.Info("fswatcher started, change a file in watcher dir"))

	// Process clean, debounced events
	for event := range w.Events() {
		var types, flags []string
		// Loop through types and flags
		for _, t := range event.Types {
			types = append(types, t.String())
		}
		for _, f := range event.Flags {
			flags = append(flags, f)
		}
		p.Send(logger.Info(fmt.Sprintf("File changed: %s %v %v\n", event.Path, types, flags)))
	}
}
