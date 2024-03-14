package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

// Server listens for requests on the socket and invokes the
// provided callback on eash message.
type Server[MSG any] struct {
	log        *log.Logger
	cb         func(context.Context, *MSG) error
	socketPath string
}

func NewServer[MSG any](logger *log.Logger, socketName string, cb func(context.Context, *MSG) error) (*Server[MSG], error) {
	path, err := getSocketPath(socketName)
	if err != nil {
		return nil, err
	}

	return &Server[MSG]{
		log:        logger,
		cb:         cb,
		socketPath: path,
	}, nil
}

func (s *Server[MSG]) decodeJson(reader io.Reader) (*MSG, error) {
	decoder := json.NewDecoder(reader)

	ret := new(MSG)
	err := decoder.Decode(ret)
	if err != nil {
		return nil, fmt.Errorf("decoder.Decode: %w", err)
	}

	return ret, nil
}

func (s *Server[MSG]) Close() error {
	return os.Remove(s.socketPath)
}

func (s *Server[MSG]) Serve(ctx context.Context) error {
	l, err := net.Listen("unix", s.socketPath)
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
			s.log.Printf("l.Accept: %s", err)
			continue
		}

		message, err := s.decodeJson(fd)
		if err != nil {
			log.Print(err)
			continue
		}
		if message == nil {
			s.log.Printf("empty socket message")
			continue
		}

		err = s.cb(ctx, message)
		if err != nil {
			s.log.Printf("s.cb: %s", err)
			continue
		}
	}
}
