package scratch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
)

type Position int

const (
	PositionCenter Position = iota
	PositionLeft
	PositionRight
)

// Definition defines the scratchpad.
type Definition struct {
	Position     Position
	Cmd          string
	WindowWidth  int
	WindowHeight int
}

func (d *Definition) Validate() error {
	if d.Cmd == "" {
		return errors.New("no command provided")
	}
	if d.Position < 0 || d.Position > PositionRight {
		return fmt.Errorf("invalid position: %d", d.Position)
	}
	if d.WindowWidth < 1 || d.WindowHeight < 1 {
		return fmt.Errorf("invalid window dimensions: %d x %d", d.WindowWidth, d.WindowHeight)
	}

	return nil
}

// Scratchpad represents a single scratchpad.
type Scratchpad struct {
	client sway.Client
	def    *Definition
	log    *log.Logger

	Pid int
}

func NewScratchpad(logger *log.Logger, c sway.Client, def *Definition) (*Scratchpad, error) {
	err := def.Validate()
	if err != nil {
		return nil, fmt.Errorf("def.Validate: %w", err)
	}

	return &Scratchpad{
		client: c,
		def:    def,
		log:    logger,
	}, nil
}

func (s *Scratchpad) spawnWindow(ctx context.Context) (pid int, err error) {
	// launch the program
	cmd := exec.CommandContext(ctx, "sh", "-c", s.def.Cmd)
	cmd.Stdout = wrapLogger("command out: ", s.log)
	cmd.Stderr = wrapLogger("command err: ", s.log)
	err = cmd.Start()
	if err != nil {
		return 0, fmt.Errorf("cmd.Start: %w", err)
	}

	// set the scratchpad rule for window's pid
	c := fmt.Sprintf(
		`for_window [pid=%d] move scratchpad; for_window [pid=%d] scratchpad show`,
		cmd.Process.Pid, cmd.Process.Pid,
	)
	_, err = s.client.RunCommand(ctx, c)
	if err != nil {
		return 0, fmt.Errorf("s.client.RunCommand: %w", err)
	}

	return cmd.Process.Pid, nil
}

var errNoMatchingNode = errors.New("no matching node")

func (s *Scratchpad) showScratchpad(ctx context.Context) error {
	if s.Pid == 0 {
		return errNoMatchingNode
	}

	cmd := fmt.Sprintf("[pid=%d] scratchpad show", s.Pid)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		if strings.Contains(err.Error(), "No matching node") {
			return errNoMatchingNode
		}
		return fmt.Errorf("s.client.RunCommand: %w", err)
	}

	return nil
}

// Toggle tries to toggle the scratchpad or opens a new one.
func (s *Scratchpad) Toggle(ctx context.Context) error {
	err := s.showScratchpad(ctx)
	if errors.Is(err, errNoMatchingNode) {
		pid, err := s.spawnWindow(ctx)
		if err != nil {
			return err
		}
		// update pid
		s.Pid = pid
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
	width := (eh.def.WindowWidth * out.Width) / out.PhysicalWidth
	height := (eh.def.WindowHeight * out.Height) / out.PhysicalHeight

	if height > out.Height {
		height = out.Height
	}
	y := (out.Height - height) / 2
	x := 0

	switch eh.def.Position {
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

// Reposition applies shape to scratchpad.
func (s *Scratchpad) Reposition(ctx context.Context, shape *Shape) error {
	cmd := fmt.Sprintf(
		"[pid=%d] resize set %d %d; [pid=%d] move absolute position %d %d",
		s.Pid, shape.Width, shape.Height, s.Pid, shape.X, shape.Y,
	)
	fmt.Println(cmd)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("s.client.RunCommand: %w", err)
	}

	return nil
}
