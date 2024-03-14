package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshuarubin/go-sway"

	"github.com/kndndrj/sway-scripts/internal/core"
	"github.com/kndndrj/sway-scripts/internal/socket"
	"github.com/kndndrj/sway-scripts/scratch/scratch"
)

// eventHandler passes event messages to server.
type eventHandler struct {
	sway.EventHandler

	log    *log.Logger
	server *scratch.Server
}

func newEventHandler(logger *log.Logger, server *scratch.Server) *eventHandler {
	return &eventHandler{
		log:    logger,
		server: server,
	}
}

// Window handler gets called on window events.
func (eh *eventHandler) Window(ctx context.Context, e sway.WindowEvent) {
	err := eh.server.OnWindow(ctx)
	if err != nil {
		eh.log.Printf("OnWindow: %s", err)
	}
}

// Workspace handler gets called on workspace events.
func (eh *eventHandler) Workspace(ctx context.Context, e sway.WorkspaceEvent) {
	err := eh.server.OnWorkspace(ctx)
	if err != nil {
		eh.log.Printf("OnWorkspace: %s", err)
	}
}

// socketMessage is passed throught the unix socket.
type socketMessage struct {
	ID         string
	Definition *scratch.Definition
}

const socketName = "sway_scratch"

// mainServer is a main function for server mode.
func mainServer() error {
	ctx := context.Background()

	logger := log.New(os.Stdout, "scratch: ", log.LstdFlags)

	// check pidfile
	err := core.LockPidFile("sway_scratch")
	if err != nil {
		if errors.Is(err, core.ErrProcessAlreadyRunning) {
			return fmt.Errorf("server already running")
		}
		return fmt.Errorf("core.LockPidFile: %w", err)
	}

	client, err := sway.New(ctx)
	if err != nil {
		return fmt.Errorf("sway.New: %w", err)
	}

	// server that manages scratchpads
	server := scratch.NewServer(client, core.NewOutputCache(client), core.NewNodeNinja(client))

	// event handler for sway events
	events := newEventHandler(logger, server)

	// socket server for requests over the socket
	sock, err := socket.NewServer(logger, socketName, func(ctx context.Context, msg *socketMessage) error {
		return server.ToggleScratchpad(ctx, msg.ID, msg.Definition)
	})
	if err != nil {
		return fmt.Errorf("socket.NewServer: %w", err)
	}
	defer sock.Close()

	// start socket handler
	go func() {
		err := sock.Serve(ctx)
		if err != nil {
			logger.Fatalf("sock.Serve: %s", err)
		}
	}()

	// start event handler
	go func() {
		err = sway.Subscribe(ctx, events, sway.EventTypeWindow, sway.EventTypeWorkspace)
		if err != nil {
			logger.Fatalf("sway.Subscribe: %s", err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	return nil
}

func main() {
	cfg, err := scratch.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// server
	if cfg.AppID == "asdf" {
		err := mainServer()
		if err != nil {
			log.Fatalf("server: %s", err)
		}
		return
	}

	// clienclient
	err = socket.Invoke(socketName,
		&socketMessage{
			ID: "someid",
			Definition: &scratch.Definition{
				Position:     scratch.PositionLeft,
				Cmd:          "kitty",
				WindowWidth:  300,
				WindowHeight: 100,
			},
		})
	if err != nil {
		log.Fatalf("client: %s", err)
	}
}
