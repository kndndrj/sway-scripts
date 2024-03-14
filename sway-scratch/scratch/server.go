package scratch

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/joshuarubin/go-sway"

	"github.com/kndndrj/sway-scripts/internal/core"
)

type Server struct {
	log         *log.Logger
	client      sway.Client
	outputCache *core.OutputCache
	ninja       *core.NodeNinja

	scratchpads map[string]*Scratchpad

	// mutex for sway client - it freezes sometimes without this for some reason.
	mu sync.Mutex
}

func NewServer(logger *log.Logger, c sway.Client, oc *core.OutputCache, ninja *core.NodeNinja) *Server {
	return &Server{
		log:         logger,
		client:      c,
		outputCache: oc,
		ninja:       ninja,
		scratchpads: make(map[string]*Scratchpad),
	}
}

func (s *Server) findScratchpadWithPid(pid *uint32) (*Scratchpad, bool) {
	if pid == nil || *pid == 0 {
		return nil, false
	}

	p := int(*pid)

	for _, s := range s.scratchpads {
		if s.Pid == p {
			return s, true
		}
	}

	return nil, false
}

// OnWindow handler should get called on window events.
func (s *Server) OnWindow(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	focused, err := s.ninja.FindFocusedNode(ctx)
	if err != nil {
		return fmt.Errorf("s.ninja.FindFocusedNode: %w", err)
	}

	// ignore all windows without the same pid as known scratchpads
	scratchpad, ok := s.findScratchpadWithPid(focused.PID)
	if !ok {
		return nil
	}

	workspace, err := s.ninja.FindFocusedWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("s.ninja.FindFocusedWorkspace: %w", err)
	}

	out, err := s.outputCache.Get(ctx, workspace.Output)
	if err != nil {
		return fmt.Errorf("s.outputCache.Get: %w", err)
	}

	// calculate window dimensions based on prefferences and display size
	shape := scratchpad.CalculateWindowShape(out)

	err = scratchpad.Reposition(ctx, shape)
	if err != nil {
		return fmt.Errorf("s.scratchpad.Reposition: %w", err)
	}
	return nil
}

// OnWorkspace handler should get called on workspace events.
func (s *Server) OnWorkspace(ctx context.Context) error {
	s.outputCache.Invalidate()
	return nil
}

// ToggleScratchpad toggles a specified scratchpad.
func (s *Server) ToggleScratchpad(ctx context.Context, id string, def *Definition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// toggle existing scratchpad
	if sc, ok := s.scratchpads[id]; ok {
		err := sc.Toggle(ctx)
		if err != nil {
			return fmt.Errorf("sc.Toggle: %w", err)
		}
		return nil
	}

	// otherwise create a new scratchpad and toggle (open it)
	sc, err := NewScratchpad(s.log, s.client, def)
	if err != nil {
		return err
	}

	err = sc.Toggle(ctx)
	if err != nil {
		return fmt.Errorf("sc.Toggle: %w", err)
	}

	s.scratchpads[id] = sc

	return nil
}
