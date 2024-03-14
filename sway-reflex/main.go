package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joshuarubin/go-sway"

	"github.com/kndndrj/sway-scripts/internal/core"
	"github.com/kndndrj/sway-scripts/sway-reflex/reflex"
)

type eventHandler struct {
	sway.EventHandler

	log *log.Logger

	outputCache *core.OutputCache
	ninja       *core.NodeNinja

	cfg *reflex.Config
}

// getScreen retrieves or initializes and then returns a screen.
func (eh *eventHandler) getScreen(ctx context.Context, outputName string) (*reflex.Screen, error) {
	out, err := eh.outputCache.Get(ctx, outputName)
	if err != nil {
		return nil, fmt.Errorf("eh.outputCache.Get: %w", err)
	}

	return reflex.NewScreen(out, eh.cfg), nil
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
	// IMPORTANT: need to search on instead of using the window from event.
	// Events might be queued and out of sync.
	focused, err := eh.ninja.FindFocusedNode(ctx)
	if err != nil {
		eh.log.Printf("eh.ninja.FindFocusedNode: %s", err)
		return
	}

	if focused.Type != sway.NodeCon {
		return
	}

	workspace, err := eh.ninja.FindFocusedWorkspace(ctx)
	if err != nil {
		eh.log.Printf("eh.ninja.FindFocusedWorkspace: %s", err)
		return
	}

	// check if workspace is disabled
	if _, ok := eh.cfg.DisabledWorkspaces[int(workspace.Num)]; ok {
		return
	}

	err = eh.ninja.WorkspaceFlattenChildren(ctx, workspace)
	if err != nil {
		eh.log.Printf("eh.ninja.WorkspaceFlattenChildren: %s", err)
		return
	}

	topLevelContainers, err := eh.ninja.WorkspaceGetTopLevelContainers(ctx, workspace)
	if err != nil {
		eh.log.Printf("eh.ninja.WorkspaceGetTopLevelContainers: %s", err)
		return
	}

	// run autogaps for toplevel containers and autotiling for
	// other nested windows.
	if nodeExistsInSet(topLevelContainers, focused) || e.Change == sway.WindowClose {
		err := eh.autogap(ctx, workspace)
		if err != nil {
			eh.log.Printf("eh.autogap: %s", err)
			return
		}
	} else {
		dir := eh.ninja.NodeDetermineSplitDirection(focused)
		err = eh.ninja.NodeApplySplitDirection(ctx, focused, dir)
		if err != nil {
			eh.log.Printf("eh.ninja.NodeApplySplitDirection: %s", err)
			return
		}
	}
}

// Workspace handler gets called on workspace events.
func (eh *eventHandler) Workspace(ctx context.Context, e sway.WorkspaceEvent) {
	eh.outputCache.Invalidate()
}

func (eh *eventHandler) Binding(ctx context.Context, e sway.BindingEvent) {
	// disable workspaces on certain binding events
	cmd := e.Binding.Command
	if !strings.Contains(cmd, "reflex:") {
		return
	}

	workspace, err := eh.ninja.FindFocusedWorkspace(ctx)
	if err != nil {
		eh.log.Printf("eh.ninja.FindFocusedWorkspace: %s", err)
		return
	}

	enable := func() {
		delete(eh.cfg.DisabledWorkspaces, int(workspace.Num))
		err := eh.autogap(ctx, workspace)
		if err != nil {
			eh.log.Printf("eh.autogap: %s", err)
			return
		}
	}

	disable := func() {
		eh.cfg.DisabledWorkspaces[int(workspace.Num)] = struct{}{}

		err := eh.ninja.ApplyOuterGaps(ctx, eh.cfg.DefaultGapHorizontal, eh.cfg.DefaultGapVertical)
		if err != nil {
			eh.log.Printf("eh.ninja.ApplyOuterGaps: %s", err)
		}
	}

	if strings.Contains(cmd, "disable_current") {
		disable()
	} else if strings.Contains(cmd, "enable_current") {
		enable()
	} else if strings.Contains(cmd, "toggle_current") {
		if _, ok := eh.cfg.DisabledWorkspaces[int(workspace.Num)]; ok {
			enable()
		} else {
			disable()
		}
	}
}

func main() {
	logger := log.New(os.Stdout, "reflex: ", log.LstdFlags)

	// check pidfile
	err := core.LockPidFile("sway_reflex")
	if err != nil {
		if errors.Is(err, core.ErrProcessAlreadyRunning) {
			logger.Print("server already running")
			return
		}
		logger.Fatalf("LockPidFile: %s", err)
	}

	ctx := context.Background()

	client, err := sway.New(ctx)
	if err != nil {
		logger.Fatalf("sway.New: %s", err)
	}

	cfg, err := reflex.ParseConfig()
	if err != nil {
		logger.Fatalf("parseConfig: %s", err)
	}

	eh := &eventHandler{
		log:         logger,
		cfg:         cfg,
		outputCache: core.NewOutputCache(client),
		ninja:       core.NewNodeNinja(client),
	}

	// start the event loop
	err = sway.Subscribe(ctx, eh, sway.EventTypeWindow, sway.EventTypeWorkspace, sway.EventTypeBinding)
	if err != nil {
		logger.Fatalf("sway.Subscribe: %s", err)
	}
}
