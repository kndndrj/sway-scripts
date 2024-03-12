package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
	"github.com/kndndrj/sway-scripts/scratch/scratch"
)

type spawner struct {
	client sway.Client
	cfg    *scratch.Config
}

func (s *spawner) spawnWindow(ctx context.Context) error {
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

func (s *spawner) showScratchpad(ctx context.Context) error {
	cmd := fmt.Sprintf("[app_id=%q] scratchpad show", s.cfg.AppID)

	_, err := s.client.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eh.client.RunCommand: %w", err)
	}

	return nil
}

// Touch tries to toggle the scratchpad or opens a new one.
func (s *spawner) Touch(ctx context.Context) error {
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

type eventHandler struct {
	sway.EventHandler

	client      sway.Client
	cfg         *scratch.Config
	outputCache *core.OutputCache
	ninja       *core.NodeNinja
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
	width, height := eh.calculateWindowDimensions(out)
	x, y := eh.calculateWindowPosition(out, width, height)

	err = eh.ninja.NodeResize(ctx, &e.Container, width, height)
	if err != nil {
		log.Printf("eh.ninja.NodeResize: %s", err)
		return
	}
	err = eh.ninja.NodeMove(ctx, &e.Container, x, y)
	if err != nil {
		log.Printf("eh.ninja.NodeMove: %s", err)
		return
	}
}

func (eh *eventHandler) calculateWindowDimensions(out *core.Output) (width, height int) {
	width = (eh.cfg.WindowWidth * out.Width) / out.PhysicalWidth
	height = (eh.cfg.WindowHeight * out.Height) / out.PhysicalHeight

	if height > out.Height {
		height = out.Height
	}

	switch eh.cfg.Position {
	case scratch.PositionCenter:
		if width > out.Width {
			width = out.Width
		}
		return width, height
	case scratch.PositionRight, scratch.PositionLeft:
		if width > (out.Width / 2) {
			width = (out.Width / 2)
		}
		return width, height
	}

	return width, height
}

func (eh *eventHandler) calculateWindowPosition(out *core.Output, width, height int) (x, y int) {
	y = (out.Height - height) / 2
	switch eh.cfg.Position {
	case scratch.PositionCenter:
		x = (out.Width - width) / 2
		return x, y
	case scratch.PositionRight:
		x = out.Width / 2
		return x, y
	case scratch.PositionLeft:
		x = (out.Width / 2) - width
		return x, y
	}

	return x, y
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

	s := &spawner{
		client: client,
		cfg:    cfg,
	}

	// open/show scratchpad immediately
	err = s.Touch(ctx)
	if err != nil {
		log.Fatalf("s.Touch: %s", err)
	}

	// spawn a server for each scratchpad app_id only once.
	// this server then listens for events and adjusts scratchpad sizes.

	err = LockPidFile(fmt.Sprintf("sway_scratch_%s", cfg.AppID))
	if err != nil {
		if errors.Is(err, ErrProcessAlreadyRunning) {
			log.Printf("server for scratchpad with app id %q already running", cfg.AppID)
			return
		}
		log.Fatalf("LockPidFile: %s", err)
	}

	eh := &eventHandler{
		client:      client,
		cfg:         cfg,
		outputCache: core.NewOutputCache(client),
		ninja:       core.NewNodeNinja(client),
	}

	// start the event loop
	err = sway.Subscribe(ctx, eh, sway.EventTypeWindow)
	if err != nil {
		log.Fatalf("sway.Subscribe: %s", err)
	}
}

var ErrProcessAlreadyRunning = errors.New("process is already running")

// LockPidFile writes process's pid into the provided file.
// If the file already exists and the process with that pid is alive,
// the function returns an ErrProcessAlreadyRunning error.
func LockPidFile(fileName string) error {
	if fileName == "" {
		return errors.New("no pidfile name passed")
	}

	// get file location
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}

	file := dir + "/" + fileName

	// check if file exists
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		// file does not exist
		return writePidFile(file)
	} else if err != nil {
		return fmt.Errorf("os.Stat: %w", err)
	}

	// file exists

	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		return fmt.Errorf("strconv.Atoi: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("os.FindProcess: %w", err)
	}

	err = process.Signal(syscall.Signal(0))
	if !errors.Is(err, os.ErrProcessDone) {
		// process is already running
		return ErrProcessAlreadyRunning
	}

	// process done, reuse the file
	return writePidFile(file)
}

func writePidFile(file string) error {
	// If the file doesn't exist, create it, otherwise overwrite it
	f, err := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	_, err = f.Write([]byte(fmt.Sprint(os.Getpid())))
	if err != nil {
		return fmt.Errorf("f.Write: %w", err)
	}

	return nil
}
