package app

import (
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/KieranGliver/bitburner-larry/internal/world"
)

type AppState struct {
	mu    sync.RWMutex
	conn  *communication.BitburnerConn
	world *world.World
	send  func(tea.Msg)
	brain *Brain
}

func (s *AppState) SetSend(fn func(tea.Msg)) {
	s.send = fn
}

func (s *AppState) SetConn(c *communication.BitburnerConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn = c
	if s.send != nil {
		if c != nil {
			s.send(communication.BitburnerConnected{Conn: c})
		} else {
			s.send(communication.BitburnerDisconnected{})
		}
	}
}

func (s *AppState) Conn() *communication.BitburnerConn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn
}

func (s *AppState) SetWorld(w *world.World) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.world = w
	if s.send != nil {
		s.send(w)
	}
}

func (s *AppState) World() *world.World {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.world
}

func (s *AppState) ensureBrain() {
	if s.brain == nil {
		s.brain = NewBrain(rankByTargetList, func(level logger.Level, summary string) {
			s.send(logger.NewLog(level, summary))
		})
	}
}

func (s *AppState) BrainStart() {
	s.ensureBrain()
	s.brain.start(s)
}

func (s *AppState) BrainStop() {
	s.ensureBrain()
	s.brain.stop()
}

func (s *AppState) BrainTick() {
	s.ensureBrain()
	s.brain.tick(s)
}

func (s *AppState) BrainRunning() bool {
	return s.brain != nil && s.brain.Running()
}
