package app

import (
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
)

type App struct {
	P       *tea.Program
	State   *AppState
	syncing atomic.Bool
}
