package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joshuarubin/go-sway"

	"github.com/kndndrj/sway-reflex/internal/core"
)

type eventHandler struct {
	sway.EventHandler
	outputCache *core.OutputCache
	ninja       *core.NodeNinja

	cfg *config
}

// getScreen retrieves or initializes and then returns a screen.
func (eh *eventHandler) getScreen(ctx context.Context, outputName string) (*screen, error) {
	out, err := eh.outputCache.Get(ctx, outputName)
	if err != nil {
		return nil, fmt.Errorf("eh.outputCache.Get: %w", err)
	}

	return newScreen(out, eh.cfg), nil
}

func (eh *eventHandler) autogap(ctx context.Context, workspace *sway.Workspace) error {
	scr, err := eh.getScreen(ctx, workspace.Output)
	if err != nil {
		return fmt.Errorf("eh.getScreen: %w", err)
	}

	// get top level containers
	topLevelContainers, err := eh.ninja.WorkspaceGetTopLevelContainers(ctx, workspace)
	if err != nil {
		return fmt.Errorf("eh.ninja.WorkspaceGetTopLevelContainers: %w", err)
	}

	// calculate dimensions of the enclosing container
	cwidth, cheight := scr.CalculateContainerDimensions(len(topLevelContainers))

	// calculate and apply gaps
	hgaps, vgaps := scr.CalculateOuterGaps(cwidth, cheight)
	err = eh.ninja.ApplyOuterGaps(ctx, hgaps, vgaps)
	if err != nil {
		return fmt.Errorf("eh.ninja.ApplyOuterGaps: %w", err)
	}

	// when there is only top level container, we can set the general direction that holds
	// true for the screen.
	if len(topLevelContainers) == 1 {
		err := eh.ninja.NodeApplySplitDirection(ctx, topLevelContainers[0], scr.Direction())
		if err != nil {
			return fmt.Errorf("eh.NodeApplySplitDirection: %w", err)
		}
	}

	if scr.IsFilled(cwidth, cheight) {
		for _, c := range topLevelContainers {
			dir := eh.ninja.NodeDetermineSplitDirection(c)
			err := eh.ninja.NodeApplySplitDirection(ctx, c, dir)
			if err != nil {
				return fmt.Errorf("eh.ninja.NodeApplySplitDirection: %w", err)
			}
		}
	}

	return nil
}

func nodeExistsInSet(set []*sway.Node, n *sway.Node) bool {
	for _, ex := range set {
		if ex.ID == n.ID {
			return true
		}
	}
	return false
}

// Window handler gets called on window events.
func (eh *eventHandler) Window(ctx context.Context, e sway.WindowEvent) {
	if e.Container.Type == sway.NodeFloatingCon {
		return
	}

	workspace, err := eh.ninja.FindFocusedWorkspace(ctx)
	if err != nil {
		log.Printf("eh.ninja.FindFocusedWorkspace: %s", err)
		return
	}

	err = eh.ninja.WorkspaceFlattenChildren(ctx, workspace)
	if err != nil {
		log.Printf("eh.ninja.WorkspaceFlattenChildren: %s", err)
		return
	}

	topLevelContainers, err := eh.ninja.WorkspaceGetTopLevelContainers(ctx, workspace)
	if err != nil {
		log.Printf("eh.ninja.WorkspaceGetTopLevelContainers: %s", err)
		return
	}

	// run autogaps for toplevel containers and autotiling for
	// other nested windows.
	if nodeExistsInSet(topLevelContainers, &e.Container) || e.Change == sway.WindowClose {
		err := eh.autogap(ctx, workspace)
		if err != nil {
			log.Printf("eh.autogap: %s", err)
			return
		}
	} else {
		dir := eh.ninja.NodeDetermineSplitDirection(&e.Container)
		err = eh.ninja.NodeApplySplitDirection(ctx, &e.Container, dir)
		if err != nil {
			log.Printf("eh.ninja.NodeApplySplitDirection: %s", err)
			return
		}
	}
}

// Workspace handler gets called on workspace events.
func (eh *eventHandler) Workspace(ctx context.Context, e sway.WorkspaceEvent) {
	eh.outputCache.Invalidate()
}

func main() {
	ctx := context.Background()

	client, err := sway.New(ctx)
	if err != nil {
		log.Fatalf("sway.New: %s", err)
	}

	cfg, err := parseConfig()
	if err != nil {
		log.Fatalf("parseConfig: %s", err)
	}

	eh := &eventHandler{
		cfg:         cfg,
		outputCache: core.NewOutputCache(client),
		ninja:       core.NewNodeNinja(client),
	}

	// start the event loop
	err = sway.Subscribe(ctx, eh, sway.EventTypeWindow, sway.EventTypeWorkspace)
	if err != nil {
		log.Fatalf("sway.Subscribe: %s", err)
	}
}
