package scratch

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

type Config struct {
	AppID        string
	Position     Position
	GenTermCmd   func(appID string) (cmd string, err error)
	WindowWidth  int
	WindowHeight int
}

func ParseConfig() (*Config, error) {
	appIDFlag := flag.String("app_id", "scratchpad", "App id to be used by the scratchpad.")
	positionFlag := flag.String("position", "center", "Position of scratchpad. Valid are: left, right, center.")
	termCmdFlag := flag.String("term", "kitty --class {{ app_id }}", "Command to be used for opening the scratchpad. It MUST contain the flag to pass an app id to it.")
	windowSizeFlag := flag.String("window_size", "200x100", "Preffered window size. <width>x<height> in [mm].")

	flag.Parse()

	width, height, err := parseWindowSize(*windowSizeFlag)
	if err != nil {
		return nil, err
	}

	pos, err := parsePosition(*positionFlag)
	if err != nil {
		return nil, err
	}

	appID, err := parseAppID(*appIDFlag)
	if err != nil {
		return nil, err
	}

	genCmd, err := parseTermCmd(*termCmdFlag)
	if err != nil {
		return nil, err
	}

	return &Config{
		AppID:        appID,
		Position:     pos,
		GenTermCmd:   genCmd,
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

func parseAppID(appID string) (string, error) {
	if appID == "" {
		return "", errors.New("app_id not provided.")
	}

	return appID, nil
}

var (
	errTermCmdNoAppID = errors.New("no way to specify an app_id in the provided term command. Need a flag with {{ app_id }}.")
	errTermCmdEmpty   = errors.New("no command for terminal provided")
)

func parseTermCmd(cmd string) (func(string) (string, error), error) {
	if cmd == "" {
		return nil, errTermCmdEmpty
	}

	if !strings.Contains(cmd, "app_id") {
		return nil, errTermCmdNoAppID
	}

	return expandCmd(cmd), nil
}

func expandCmd(cmd string) func(string) (string, error) {
	return func(appID string) (string, error) {
		tmpl, err := template.New("expand_term_cmd").
			Funcs(template.FuncMap{
				"app_id": func() string {
					return appID
				},
			}).
			Parse(cmd)
		if err != nil {
			return "", fmt.Errorf("template.Parse: %w", err)
		}

		var out bytes.Buffer
		err = tmpl.Execute(&out, nil)
		if err != nil {
			return "", err
		}

		return out.String(), nil
	}
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
