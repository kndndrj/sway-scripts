package scratch

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Subcommand int

const (
	SubcommandUnknown Subcommand = iota
	SubcommandServe
	SubcommandCall
)

func SubcommandFromString(s string) Subcommand {
	switch s {
	case "serve":
		return SubcommandServe
	case "call":
		return SubcommandCall
	}
	return SubcommandUnknown
}

func (s Subcommand) String() string {
	switch s {
	case SubcommandServe:
		return "serve"
	case SubcommandCall:
		return "call"
	}
	return "unknown"
}

func GetSubcommand() (Subcommand, error) {
	if len(os.Args) < 2 {
		return 0, errors.New("expected a subcommand")
	}

	subcommand := SubcommandFromString(os.Args[1])
	if subcommand == SubcommandUnknown {
		return 0, fmt.Errorf("unknown subcommand: %q", os.Args[1])
	}

	return subcommand, nil
}

type CallConfig struct {
	ID           string
	Position     Position
	Cmd          string
	WindowWidth  int
	WindowHeight int
}

func ParseCallFlags() (*CallConfig, error) {
	// third argument is a window command
	if len(os.Args) < 3 || os.Args[2][0] == '-' {
		return nil, errors.New("expected a window command - example: kitty")
	}

	cmd := os.Args[2]

	const placeholderID = "<cmd>_<position>_<window_size>"

	subcmd := flag.NewFlagSet(SubcommandCall.String(), flag.ExitOnError)
	idFlag := subcmd.String("id", placeholderID, "Unique id to be used by the scratchpad.")
	positionFlag := subcmd.String("position", "center", "Position of scratchpad. Valid are: left, right, center.")
	windowSizeFlag := subcmd.String("window_size", "200x100", "Preffered window size. <width>x<height> in [mm].")

	err := subcmd.Parse(os.Args[3:])
	if err != nil {
		return nil, err
	}

	width, height, err := parseWindowSize(*windowSizeFlag)
	if err != nil {
		return nil, err
	}

	pos, err := parsePosition(*positionFlag)
	if err != nil {
		return nil, err
	}

	id := *idFlag
	if *idFlag == "" || *idFlag == placeholderID {
		id = fmt.Sprintf("%s_%d_%dx%d", cmd, pos, width, height)
	}

	return &CallConfig{
		ID:           id,
		Position:     pos,
		Cmd:          cmd,
		WindowWidth:  width,
		WindowHeight: height,
	}, nil
}

func parsePosition(in string) (Position, error) {
	input := strings.ToLower(in)

	switch input {
	case "center":
		return PositionCenter, nil
	case "left":
		return PositionLeft, nil
	case "right":
		return PositionRight, nil
	}

	return 0, fmt.Errorf("invalid position flag: %q", in)
}

func parseWindowSize(in string) (w, h int, err error) {
	input := strings.ToLower(in)

	sp := strings.Split(input, "x")
	if len(sp) != 2 {
		return 0, 0, fmt.Errorf("invlid window size format: %q, should be: <widhth>x<height>", in)
	}

	w, err = strconv.Atoi(sp[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width parameter: %q - not a number", sp[0])
	}

	h, err = strconv.Atoi(sp[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height parameter: %q - not a number", sp[1])
	}

	if w < 1 && h < 1 {
		return 0, 0, fmt.Errorf("invalid window size parameter: %q - widht and height should be positive integers", in)
	}

	return w, h, nil
}
