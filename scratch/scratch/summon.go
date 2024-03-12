package scratch

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
)

// Scratchpad Summoner is used to cast spells around scratchpads.
type Summoner struct {
	client sway.Client
	cfg    *Config
}

func NewSummoner(c sway.Client, cfg *Config) *Summoner {
	return &Summoner{
		client: c,
		cfg:    cfg,
	}
}

func (s *Summoner) spawnWindow(ctx context.Context) error {
	command, err := s.cfg.GenTermCmd(s.cfg.AppID)
	if err != nil {
		return fmt.Errorf("h.cfg.GenTermCmd: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Start: %w", err)
	}

	return nil
}

func (s *Summoner) showScratchpad(ctx context.Context) error {
	cmd := fmt.Sprintf("[app_id=%q] scratchpad show", s.cfg.AppID)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// Summon tries to toggle the scratchpad or opens a new one.
func (s *Summoner) Summon(ctx context.Context) error {
	err := s.showScratchpad(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "No matching node") {
			err := s.spawnWindow(ctx)
			if err != nil {
				return err
			}
			return nil
		}
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

func (eh *Summoner) CalculateWindowShape(out *core.Output) *Shape {
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
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}