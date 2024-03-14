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
		return "", errors.New("no socket name passed")
	}

	// get file location
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}

	return dir + "/" + name + ".sock", nil
}

// ClearSocket forcefully removes the socket from the path.
// CAUTION!
func ClearSocket(name string) error {
	path, err := getSocketPath(name)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("os.Remove: %w", err)
	}
	return nil
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
