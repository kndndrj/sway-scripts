package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshuarubin/go-sway"
	"github.com/kndndrj/sway-scripts/internal/core"
	"github.com/kndndrj/sway-scripts/scratch/scratch"
)

// eventHandler passes event messages to server.
type eventHandler struct {
	sway.EventHandler

	log    *log.Logger
	server *scratch.Server
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

// socketHandler passes messages from and to the unix socket between server and client.
type socketHandler struct {
	server *scratch.Server
	log    *log.Logger
}

type scratchMessage struct {
	ID         string
	Definition *scratch.Definition
}

func (sh *socketHandler) decodeJson(reader io.Reader) (*scratchMessage, error) {
	decoder := json.NewDecoder(reader)

	ret := new(scratchMessage)
	err := decoder.Decode(ret)
	if err != nil {
		return nil, fmt.Errorf("decoder.Decode: %w", err)
	}

	return ret, nil
}

func (sh *socketHandler) Close() error {
	return os.Remove("/tmp/echo.sock")
}

func (sh *socketHandler) Serve(ctx context.Context) error {
	l, err := net.Listen("unix", "/tmp/echo.sock")
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fd, err := l.Accept()
		if err != nil {
			sh.log.Printf("l.Accept: %s", err)
			continue
		}

		message, err := sh.decodeJson(fd)
		if err != nil {
			log.Print(err)
			continue
		}

		err = sh.server.ToggleScratchpad(ctx, message.ID, message.Definition)
		if err != nil {
			sh.log.Printf("sh.server.ToggleScratchpad: %s", err)
			continue
		}
	}
}

func sendMessage(id string, def *scratch.Definition) error {
	c, err := net.Dial("unix", "/tmp/echo.sock")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	message := &scratchMessage{
		ID:         id,
		Definition: def,
	}

	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	_, err = c.Write(b)
	if err != nil {
		return fmt.Errorf("c.Write: %w", err)
	}

	return nil
}

// mainserver is a main function for server mode.
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

	server := scratch.NewServer(client, core.NewOutputCache(client), core.NewNodeNinja(client))

	eh := &eventHandler{
		log:    logger,
		server: server,
	}

	sh := &socketHandler{
		server: server,
		log:    logger,
	}

	defer sh.Close()

	// start socket handler
	go func() {
		err := sh.Serve(ctx)
		if err != nil {
			logger.Fatalf("sh.Serve: %s", err)
		}
	}()

	// start event handler
	go func() {
		err = sway.Subscribe(ctx, eh, sway.EventTypeWindow, sway.EventTypeWorkspace)
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
	err = sendMessage("srat", &scratch.Definition{
		Position:     scratch.PositionLeft,
		Cmd:          "kitty",
		WindowWidth:  300,
		WindowHeight: 100,
	})
	if err != nil {
		log.Fatalf("client: %s", err)
	}
}
