package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/joshuarubin/go-sway"
)

// NodeNinja implements some useful utils for working with workspace nodes.
type NodeNinja struct {
	client sway.Client
}

// NewNodeNinja summons a new shadow of a shinobi.
func NewNodeNinja(cl sway.Client) *NodeNinja {
	return &NodeNinja{
		client: cl,
	}
}

// getWorkspaceNode returns the node of the provided workspace.
func (nn *NodeNinja) getWorkspaceNode(ctx context.Context, workspace *sway.Workspace) (*sway.Node, error) {
	t, err := nn.client.GetTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("eh.client.GetTree: %w", err)
	}

	node := t.TraverseNodes(func(n *sway.Node) bool {
		if n.Type == sway.NodeWorkspace &&
			n.Name == workspace.Name {
			return true
		}

		return false
	})

	if node == nil {
		return nil, fmt.Errorf("workspace with number: %d not found", workspace.Num)
	}

	return node, nil
}

// FindFocusedWorkspace finds the currently focused workspace.
func (nn *NodeNinja) FindFocusedWorkspace(ctx context.Context) (*sway.Workspace, error) {
	workspaces, err := nn.client.GetWorkspaces(ctx)
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

// FindFocusedNode finds the currently focused node.
func (nn *NodeNinja) FindFocusedNode(ctx context.Context) (*sway.Node, error) {
	node, err := nn.client.GetTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("eh.client.GetWorkspaces: %w", err)
	}

	return node.FocusedNode(), nil
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

// WorkspaceGetTopLevelContainers returns top level containers in the provided workspace.
func (nn *NodeNinja) WorkspaceGetTopLevelContainers(ctx context.Context, workspace *sway.Workspace) ([]*sway.Node, error) {
	node, err := nn.getWorkspaceNode(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("eh.getWorkspaceNode: %w", err)
	}

	return filterConNodes(node.Nodes), nil
}

// WorkspaceFlattenChildren flattens children under the provided workspace.
func (nn *NodeNinja) WorkspaceFlattenChildren(ctx context.Context, workspace *sway.Workspace) error {
	var flatten func(rootNode *sway.Node) error
	flatten = func(rootNode *sway.Node) error {
		// node without children
		if len(rootNode.Nodes) == 1 &&
			rootNode.Type == sway.NodeCon &&
			rootNode.Nodes[0].Type == sway.NodeCon {
			_, err := nn.client.RunCommand(ctx, fmt.Sprintf("[con_id=%d] split none", rootNode.Nodes[0].ID))
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

	node, err := nn.getWorkspaceNode(ctx, workspace)
	if err != nil {
		return fmt.Errorf("eh.getWorkspaceNode: %w", err)
	}

	return flatten(node)
}

// ApplyOuterGaps applies gaps for current workspace.
func (nn *NodeNinja) ApplyOuterGaps(ctx context.Context, horizontal, vertical int) error {
	if horizontal < 0 {
		horizontal = 0
	}
	if vertical < 0 {
		vertical = 0
	}
	cmd := fmt.Sprintf("gaps horizontal current set %d; gaps vertical current set %d", horizontal, vertical)

	_, err := nn.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// Direction represents orientation (horizontal/vertical)
type Direction int

const (
	DirectionHorizontal Direction = iota
	DirectionVertical
)

// toSplitCmd converts direction enum to split command.
func (d Direction) toLayout() string {
	if d == DirectionVertical {
		return "splitv"
	}
	return "splith"
}

// NodeDetermineSplitDirection determines split direction of the provided node based on autotile heuristics.
func (nn *NodeNinja) NodeDetermineSplitDirection(node *sway.Node) Direction {
	if node.Rect.Height > node.Rect.Width {
		return DirectionVertical
	}

	return DirectionHorizontal
}

// NodeApplySplitDirection applies split direction for a specific node.
func (nn *NodeNinja) NodeApplySplitDirection(ctx context.Context, node *sway.Node, dir Direction) error {
	ly := dir.toLayout()
	if node.Orientation == ly {
		return nil
	}

	cmd := fmt.Sprintf("[con_id=%d] %s", node.ID, ly)

	_, err := nn.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}
