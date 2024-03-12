package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
	"github.com/kndndrj/sway-scripts/scratch/scratch"
)

type eventHandler struct {
	sway.EventHandler

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

	workspace, err := eh.ninja.FindFocusedWorkspace(ctx)
	if err != nil {
		log.Printf("eh.ninja.FindFocusedWorkspace: %s", err)
		return
	}

	out, err := eh.outputCache.Get(ctx, workspace.Output)
	if err != nil {
		log.Printf("eh.outputCache.Get: %s", err)
		return
	}

	// calculate window dimensions based on prefferences and display size
	shape := eh.summoner.CalculateWindowShape(out)

	err = eh.ninja.NodeResize(ctx, &e.Container, shape.Width, shape.Height)
	if err != nil {
		log.Printf("eh.ninja.NodeResize: %s", err)
		return
	}
	err = eh.ninja.NodeMove(ctx, &e.Container, shape.X, shape.Y)
	if err != nil {
		log.Printf("eh.ninja.NodeMove: %s", err)
		return
	}
}

func main() {
	ctx := context.Background()

	client, err := sway.New(ctx)
	if err != nil {
		log.Fatalf("sway.New: %s", err)
	}

	cfg, err := scratch.ParseConfig()
	if err != nil {
		log.Fatalf("scratch.ParseConfig: %s", err)
	}

	s := scratch.NewSummoner(client, cfg)

	// open/show scratchpad immediately
	err = s.Summon(ctx)
	if err != nil {
		log.Fatalf("s.Touch: %s", err)
	}

	// spawn a server for each scratchpad app_id only once.
	// this server then listens for events and adjusts scratchpad sizes.

	err = core.LockPidFile(fmt.Sprintf("sway_scratch_%s", cfg.AppID))
	if err != nil {
		if errors.Is(err, core.ErrProcessAlreadyRunning) {
			log.Printf("server for scratchpad with app id %q already running", cfg.AppID)
			return
		}
		log.Fatalf("LockPidFile: %s", err)
	}

	eh := &eventHandler{
		cfg:         cfg,
		outputCache: core.NewOutputCache(client),
		ninja:       core.NewNodeNinja(client),
		summoner:    s,
	}

	// start the event loop
	err = sway.Subscribe(ctx, eh, sway.EventTypeWindow)
	if err != nil {
		log.Fatalf("sway.Subscribe: %s", err)
	}
}
