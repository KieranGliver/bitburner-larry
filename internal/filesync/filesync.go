package filesync

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sgtdi/fswatcher"
)

func Watch(path string, p *tea.Program, onEvent func(event fswatcher.WatchEvent)) {
	w, _ := fswatcher.New(
		fswatcher.WithPath(path),
		fswatcher.WithCooldown(500*time.Millisecond),
	)

	ctx := context.Background()
	go w.Watch(ctx)

	for event := range w.Events() {
		onEvent(event)
	}
}
