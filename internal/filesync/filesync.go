package filesync

import (
	"context"
	"time"

	"github.com/sgtdi/fswatcher"
)

func Watch(path string, onError func(string), onEvent func(event fswatcher.WatchEvent)) {
	w, err := fswatcher.New(
		fswatcher.WithPath(path),
		fswatcher.WithCooldown(500*time.Millisecond),
	)
	if err != nil {
		onError("filesync " + path + ": " + err.Error())
		return
	}

	ctx := context.Background()
	go w.Watch(ctx)

	for event := range w.Events() {
		onEvent(event)
	}
}
