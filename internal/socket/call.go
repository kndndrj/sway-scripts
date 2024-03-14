package socket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
)

func getSocketPath(name string) (string, error) {
	if name == "" {
		return "", errors.New("no pidfile name passed")
	}

	// get file location
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}

	return dir + "/" + name + ".sock", nil
}

func Invoke(socketName string, msg any) error {
	path, err := getSocketPath(socketName)
	if err != nil {
		return err
	}

	c, err := net.Dial("unix", path)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	_, err = c.Write(b)
	if err != nil {
		return fmt.Errorf("c.Write: %w", err)
	}

	return nil
}
