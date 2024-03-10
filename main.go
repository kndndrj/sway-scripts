package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/joshuarubin/go-sway"

	"github.com/kndndrj/sway-reflex/internal/output"
)

type direction int

const (
	directionHorizontal direction = iota
	directionVertical
)

// toSplitCmd converts direction enum to split command.
func (d direction) toSplitCmd() string {
	if d == directionVertical {
		return "splitv"
	}
	return "splith"
}

// screen represents a working unit of an output.
type screen struct {
	Width  int
	Height int

	PrefferedWindowWidth  int
	PrefferedWindowHeight int

	// which way is the top level container being split?
	Direction direction
}

// isFilled returns true if based on provided container dimensions screen is fully filled.
func (s *screen) isFilled(cwidth, cheight int) bool {
	if cwidth < s.Width {
		return false
	}
	if cheight < s.Height {
		return false
	}
	return true
}

func newScreen(o *output.Output, pr *prefferences) *screen {
	prefferedWindowWidth := (pr.PhysicalWindowWidth * o.Width) / o.PhysicalWidth
	prefferedWindowHeight := (pr.PhysicalWindowHeight * o.Height) / o.PhysicalHeight

	dir := directionHorizontal
	if o.Height-prefferedWindowHeight > o.Width-prefferedWindowWidth {
		dir = directionVertical
	}

	return &screen{
		Width:                 o.Width,
		Height:                o.Height,
		PrefferedWindowWidth:  prefferedWindowWidth,
		PrefferedWindowHeight: prefferedWindowHeight,
		Direction:             dir,
	}
}

type prefferences struct {
	// Preffered physical dimensions of windows.
	PhysicalWindowWidth  int // all physical dimensions are in [mm]
	PhysicalWindowHeight int
}

type eventHandler struct {
	sway.EventHandler
	client  sway.Client
	screens map[string]*screen

	prefferences *prefferences
}

// getScreen retrieves or initializes and then returns a screen.
func (eh *eventHandler) getScreen(ctx context.Context, outputName string) (*screen, error) {
	scr, ok := eh.screens[outputName]
	if ok {
		return scr, nil
	}

	// get all outputs from wayland
	outputs, err := output.Get()
	if err != nil {
		return nil, fmt.Errorf("output.Get: %w", err)
	}

	// add new and update existing screens
	for _, o := range outputs {
		scr = newScreen(o, eh.prefferences)
		eh.screens[o.Name] = scr
	}

	scr, ok = eh.screens[outputName]
	if !ok {
		return nil, fmt.Errorf("screen with name: %q not found", outputName)
	}

	return scr, nil
}

// findFocusedWorkspace finds the currently focused workspace.
func (eh *eventHandler) findFocusedWorkspace(ctx context.Context) (*sway.Workspace, error) {
	workspaces, err := eh.client.GetWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("eh.client.GetWorkspaces: %w", err)
	}

	for _, w := range workspaces {
		if w.Focused {
			return &w, nil
		}
	}

	// should never come to this
	return nil, errors.New("no focused workspace")
}

// getWorkspaceNode returns the node of the provided workspace.
func (eh *eventHandler) getWorkspaceNode(ctx context.Context, workspace *sway.Workspace) (*sway.Node, error) {
	t, err := eh.client.GetTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("eh.client.GetTree: %w", err)
	}

	node := t.TraverseNodes(func(n *sway.Node) bool {
		if n.Type == sway.NodeWorkspace &&
			n.Name == strconv.Itoa(int(workspace.Num)) {
			return true
		}

		return false
	})

	if node == nil {
		return nil, fmt.Errorf("workspace with number: %d not found", workspace.Num)
	}

	return node, nil
}

func filterConNodes(in []*sway.Node) []*sway.Node {
	var out []*sway.Node
	for _, n := range in {
		if n.Type == sway.NodeCon {
			out = append(out, n)
		}
	}
	return out
}

// getWorkspaceTopLevelContainers returns top level containers in the provided workspace.
func (eh *eventHandler) getWorkspaceTopLevelContainers(ctx context.Context, workspace *sway.Workspace) ([]*sway.Node, error) {
	var get func(node *sway.Node) []*sway.Node
	get = func(node *sway.Node) []*sway.Node {
		nodes := filterConNodes(node.Nodes)

		if len(nodes) > 1 {
			return nodes
		}

		if len(nodes) == 1 {
			children := get(nodes[0])
			if len(children) > 0 {
				return children
			}
		}

		return nodes
	}

	node, err := eh.getWorkspaceNode(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("eh.getWorkspaceNode: %w", err)
	}

	return get(node), nil
}

// flattenWorkspaceChildren flattens children under the current workspace.
func (eh *eventHandler) flattenWorkspaceChildren(ctx context.Context, workspace *sway.Workspace) error {
	var flatten func(rootNode *sway.Node) error
	flatten = func(rootNode *sway.Node) error {
		// node without children
		if len(rootNode.Nodes) == 1 && rootNode.Type == sway.NodeCon && rootNode.Nodes[0].Type == sway.NodeCon {
			_, err := eh.client.RunCommand(ctx, fmt.Sprintf("[con_id=%d] split none", rootNode.Nodes[0].ID))
			if err != nil {
				return fmt.Errorf("eh.client.RunCommand: %w", err)
			}
		}

		for _, n := range rootNode.Nodes {
			err := flatten(n)
			if err != nil {
				return err
			}
		}

		return nil
	}

	node, err := eh.getWorkspaceNode(ctx, workspace)
	if err != nil {
		return fmt.Errorf("eh.getWorkspaceNode: %w", err)
	}

	return flatten(node)
}

// calculateContainerDimmensions adjusts top level container (all windows together) dimensions,
// so that they fit on the screen.
func calculateContainerDimmensions(scr *screen, numOfTopLevelContainers int) (width, height int) {
	if scr.Direction == directionHorizontal {
		fullWidth := scr.PrefferedWindowWidth * numOfTopLevelContainers

		// width fits on screen
		if fullWidth <= scr.Width {
			height := scr.PrefferedWindowHeight
			if scr.Height < height {
				height = scr.Height
			}
			return fullWidth, height
		}

		// width doesn't fit on screen and height doesn't as well
		if scr.Height <= scr.PrefferedWindowHeight {
			return scr.Width, scr.Height
		}

		// try to compensate off-screen surface with height
		offScreenSurface := (fullWidth - scr.Width) * scr.PrefferedWindowHeight
		deltaHeight := offScreenSurface / scr.Width

		height := scr.PrefferedWindowHeight + deltaHeight
		if scr.Height <= height {
			return scr.Width, scr.Height
		}

		return scr.Width, height
	}

	// else if directionVertical

	fullHeight := scr.PrefferedWindowHeight * numOfTopLevelContainers

	// height fits on screen
	if fullHeight <= scr.Height {
		width := scr.PrefferedWindowWidth
		if scr.Width < width {
			width = scr.Width
		}
		return width, fullHeight
	}

	// height doesn't fit on screen and width doesn't as well
	if scr.Width <= scr.PrefferedWindowWidth {
		return scr.Width, scr.Height
	}

	// try to compensate off-screen surface with width
	offScreenSurface := (fullHeight - scr.Height) * scr.PrefferedWindowWidth
	deltaWidth := offScreenSurface / scr.Height

	width = scr.PrefferedWindowWidth + deltaWidth
	if scr.Width <= width {
		return scr.Width, scr.Height
	}

	return width, scr.Height
}

// calculateOuterGaps calculates gaps between the edge of the screen and top level container.
func calculateOuterGaps(scr *screen, containerWidth, containerHeight int) (horizontal, vertical int) {
	widthDiff := scr.Width - containerWidth
	if widthDiff < 0 {
		widthDiff = 0
	}
	heightDiff := scr.Height - containerHeight
	if heightDiff < 0 {
		heightDiff = 0
	}

	return widthDiff / 2, heightDiff / 2
}

// applyOuterGaps applies gaps for current workspace.
func (eh *eventHandler) applyOuterGaps(ctx context.Context, horizontal, vertical int) error {
	if horizontal < 0 {
		horizontal = 0
	}
	if vertical < 0 {
		vertical = 0
	}
	cmd := fmt.Sprintf("gaps horizontal current set %d; gaps vertical current set %d", horizontal, vertical)

	_, err := eh.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// applySplitDirection applies split direction for a specific node.
func (eh *eventHandler) applySplitDirection(ctx context.Context, node *sway.Node, dir direction) error {
	cmd := fmt.Sprintf("[con_id=%d] %s", node.ID, dir.toSplitCmd())

	_, err := eh.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// determineSplitDirection determines split direction of the provided node based on autotile heuristics.
func determineSplitDirection(node *sway.Node) direction {
	if node.Rect.Height > node.Rect.Width {
		return directionVertical
	}

	return directionHorizontal
}

func (eh *eventHandler) autogap(ctx context.Context, workspace *sway.Workspace) error {
	scr, err := eh.getScreen(ctx, workspace.Output)
	if err != nil {
		return fmt.Errorf("eh.getScreen: %w", err)
	}

	// get top level containers
	topLevelContainers, err := eh.getWorkspaceTopLevelContainers(ctx, workspace)
	if err != nil {
		return fmt.Errorf("eh.getWorkspaceTopLevelContainers: %w", err)
	}

	// calculate dimensions of the enclosing container
	cwidth, cheight := calculateContainerDimmensions(scr, len(topLevelContainers))

	// calculate and apply gaps
	hgaps, vgaps := calculateOuterGaps(scr, cwidth, cheight)
	err = eh.applyOuterGaps(ctx, hgaps, vgaps)
	if err != nil {
		return fmt.Errorf("eh.calculateOuterGaps: %w", err)
	}

	// when there is only top level container, we can set the general direction that holds
	// true for the screen.
	if len(topLevelContainers) == 1 {
		err := eh.applySplitDirection(ctx, topLevelContainers[0], scr.Direction)
		if err != nil {
			return fmt.Errorf("eh.applySplitDirection: %w", err)
		}
	}

	if scr.isFilled(cwidth, cheight) {
		for _, c := range topLevelContainers {
			dir := determineSplitDirection(c)
			err := eh.applySplitDirection(ctx, c, dir)
			if err != nil {
				return fmt.Errorf("eh.applySplitDirection: %w", err)
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

	workspace, err := eh.findFocusedWorkspace(ctx)
	if err != nil {
		log.Printf("eh.findFocusedWorkspace: %s", err)
		return
	}

	err = eh.flattenWorkspaceChildren(ctx, workspace)
	if err != nil {
		log.Printf("eh.flattenWorkspaceChildren: %s", err)
		return
	}

	// get top level containers
	topLevelContainers, err := eh.getWorkspaceTopLevelContainers(ctx, workspace)
	if err != nil {
		log.Printf("eh.getWorkspaceTopLevelContainers: %s", err)
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
		dir := determineSplitDirection(&e.Container)
		err = eh.applySplitDirection(ctx, &e.Container, dir)
		if err != nil {
			log.Printf("eh.applySplitDirection: %s", err)
			return
		}

	}
}

func main() {
	ctx := context.Background()

	client, err := sway.New(ctx)
	if err != nil {
		log.Fatalf("sway.New: %s", err)
	}

	eh := &eventHandler{
		client: client,
		prefferences: &prefferences{
			PhysicalWindowWidth:  200,
			PhysicalWindowHeight: 100,
		},
		screens: make(map[string]*screen),
	}

	// start the event loop
	err = sway.Subscribe(ctx, eh, sway.EventTypeWindow, sway.EventTypeMode)
	if err != nil {
		log.Fatalf("sway.Subscribe: %s", err)
	}
}
