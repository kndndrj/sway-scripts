package scratch

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
)

// Scratchpad represents a single scratchpad.
type Scratchpad struct {
	client sway.Client
	cfg    *Config
}

func NewSummoner(c sway.Client, cfg *Config) *Scratchpad {
	return &Scratchpad{
		client: c,
		cfg:    cfg,
	}
}

// MoveToScratchpad configures the scratchpad as scratchpad.
func (s *Scratchpad) MoveToScratchpad(ctx context.Context) error {
	cmd := fmt.Sprintf("[app_id=%q] move scratchpad; [app_id=%q] scratchpad show", s.cfg.AppID, s.cfg.AppID)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

func (s *Scratchpad) spawnWindow(ctx context.Context) error {
	command, err := s.cfg.GenTermCmd(s.cfg.AppID)
	if err != nil {
		return fmt.Errorf("h.cfg.GenTermCmd: %w", err)
	}

	// launch the program
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Start: %w", err)
	}

	return nil
}

func (s *Scratchpad) showScratchpad(ctx context.Context) error {
	cmd := fmt.Sprintf("[app_id=%q] scratchpad show", s.cfg.AppID)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// Summon tries to toggle the scratchpad or opens a new one.
func (s *Scratchpad) Summon(ctx context.Context) error {
	err := s.showScratchpad(ctx)
	if err != nil && strings.Contains(err.Error(), "No matching node") {
		err := s.spawnWindow(ctx)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// Shape describes scratchpad size and position on screen.
type Shape struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (eh *Scratchpad) CalculateWindowShape(out *core.Output) *Shape {
	width := (eh.cfg.WindowWidth * out.Width) / out.PhysicalWidth
	height := (eh.cfg.WindowHeight * out.Height) / out.PhysicalHeight

	if height > out.Height {
		height = out.Height
	}
	y := (out.Height - height) / 2
	x := 0

	switch eh.cfg.Position {
	case PositionCenter:
		if width > out.Width {
			width = out.Width
		}
		x = (out.Width - width) / 2
	case PositionRight:
		if width > (out.Width / 2) {
			width = (out.Width / 2)
		}
		x = out.Width / 2
	case PositionLeft:
		if width > (out.Width / 2) {
			width = (out.Width / 2)
		}
		x = (out.Width / 2) - width
	}

	return &Shape{
		X:      x + out.X,
		Y:      y + out.Y,
		Width:  width,
		Height: height,
	}
}

// Resize applies dimensions to specified node.
func (s *Scratchpad) Resize(ctx context.Context, width, height int) error {
	cmd := fmt.Sprintf("[app_id=%q] resize set %d %d", s.cfg.AppID, width, height)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// Move moves the specified node to the position
func (s *Scratchpad) Move(ctx context.Context, x, y int) error {
	cmd := fmt.Sprintf("[app_id=%q] move absolute position %d %d", s.cfg.AppID, x, y)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}
