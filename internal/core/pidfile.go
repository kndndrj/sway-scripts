package core

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
)

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

	file := dir + "/" + fileName + ".pid"

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
	f, err := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	_, err = f.Write([]byte(fmt.Sprint(os.Getpid())))
	if err != nil {
		return fmt.Errorf("f.Write: %w", err)
	}

	return nil
}
