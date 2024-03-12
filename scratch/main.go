package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
	"github.com/kndndrj/sway-scripts/scratch/scratch"
)

type eventHandler struct {
	sway.EventHandler

	log *log.Logger

	cfg         *scratch.Config
	outputCache *core.OutputCache
	ninja       *core.NodeNinja
	summoner    *scratch.Summoner
}

// Window handler gets called on window events.
func (eh *eventHandler) Window(ctx context.Context, e sway.WindowEvent) {
	// ignore close changes
	if e.Change == sway.WindowClose {
		return
	}

	// ignore all app ids other than the one provided
	if e.Container.AppID == nil || *e.Container.AppID != eh.cfg.AppID {
		return
	}

	// move to sway scratchpad the first time running
	if e.Change == sway.WindowNew {
		err := eh.summoner.MoveToScratchpad(ctx)
		if err != nil {
			eh.log.Printf("eh.summoner.Float: %s", err)
			return
		}
	}

	workspace, err := eh.ninja.FindFocusedWorkspace(ctx)
	if err != nil {
		eh.log.Printf("eh.ninja.FindFocusedWorkspace: %s", err)
		return
	}

	out, err := eh.outputCache.Get(ctx, workspace.Output)
	if err != nil {
		eh.log.Printf("eh.outputCache.Get: %s", err)
		return
	}

	// calculate window dimensions based on prefferences and display size
	shape := eh.summoner.CalculateWindowShape(out)

	err = eh.summoner.Resize(ctx, shape.Width, shape.Height)
	if err != nil {
		eh.log.Printf("eh.summoner.Resize: %s", err)
		return
	}
	err = eh.summoner.Move(ctx, shape.X, shape.Y)
	if err != nil {
		eh.log.Printf("eh.summoner.Move: %s", err)
		return
	}
}

func main() {
	ctx := context.Background()

	logger := log.New(os.Stdout, "scratch:", log.LstdFlags)

	client, err := sway.New(ctx)
	if err != nil {
		logger.Fatalf("sway.New: %s", err)
	}

	cfg, err := scratch.ParseConfig()
	if err != nil {
		logger.Fatalf("scratch.ParseConfig: %s", err)
	}

	s := scratch.NewSummoner(client, cfg)

	// open/show scratchpad immediately
	err = s.Summon(ctx)
	if err != nil {
		logger.Fatalf("s.Touch: %s", err)
	}

	// spawn a server for each scratchpad app_id only once.
	// this server then listens for events and adjusts scratchpad sizes.
	err = core.LockPidFile(fmt.Sprintf("sway_scratch_%s", cfg.AppID))
	if err != nil {
		if errors.Is(err, core.ErrProcessAlreadyRunning) {
			return
		}
		logger.Fatalf("LockPidFile: %s", err)
	}

	eh := &eventHandler{
		log:         logger,
		cfg:         cfg,
		outputCache: core.NewOutputCache(client),
		ninja:       core.NewNodeNinja(client),
		summoner:    s,
	}

	// start the event loop
	err = sway.Subscribe(ctx, eh, sway.EventTypeWindow)
	if err != nil {
		logger.Fatalf("sway.Subscribe: %s", err)
	}
}
