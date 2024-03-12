package reflex

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	// Preffered physical dimensions of windows in [mm].
	PhysicalWindowWidth  int
	PhysicalWindowHeight int

	// Default outer gaps in [px].
	DefaultGapHorizontal int
	DefaultGapVertical   int

	// List of disabled workspaces.
	DisabledWorkspaces map[int]struct{}
	// List of disabled app_ids.
	DisabledAppIDs map[string]struct{}
}

func ParseConfig() (*Config, error) {
	prefferedWindowSize := flag.String("window_size", "500x300", "Preffered window size. <width>x<height> in [mm].")
	defaultGaps := flag.Int("default_gaps", 0, "Default outer gaps [px].")
	disabledWorkspaces := flag.String("disable_workspaces", "", "Comma-seperated list of workspace numbers to disable.")
	disabledAppIDsFlag := flag.String("disable_app_ids", "", "Comma-seperated list of app_ids to disable.")

	flag.Parse()

	width, height, err := parseWindowSize(*prefferedWindowSize)
	if err != nil {
		return nil, err
	}

	gaps, err := parseGaps(*defaultGaps)
	if err != nil {
		return nil, err
	}

	disabledWss, err := parseDisabledWorkspaces(*disabledWorkspaces)
	if err != nil {
		return nil, err
	}

	disabledAppIDs, err := parseDisabledAppIDs(*disabledAppIDsFlag)
	if err != nil {
		return nil, err
	}

	return &Config{
		PhysicalWindowWidth:  width,
		PhysicalWindowHeight: height,

		DefaultGapHorizontal: gaps,
		DefaultGapVertical:   gaps,

		DisabledWorkspaces: disabledWss,

		DisabledAppIDs: disabledAppIDs,
	}, nil
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

func parseGaps(in int) (int, error) {
	if in < 0 {
		return 0, fmt.Errorf(`invalid default gaps parameter: "%d" - should be a positive integer`, in)
	}

	return in, nil
}

func parseDisabledWorkspaces(in string) (map[int]struct{}, error) {
	if in == "" {
		return make(map[int]struct{}), nil
	}

	sp := strings.Split(in, ",")

	ret := make(map[int]struct{}, len(sp))
	for _, s := range sp {
		w, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid workspace number: %q - not a number", s)
		}
		ret[w] = struct{}{}
	}

	return ret, nil
}

func parseDisabledAppIDs(in string) (map[string]struct{}, error) {
	if in == "" {
		return make(map[string]struct{}), nil
	}

	sp := strings.Split(in, ",")

	ret := make(map[string]struct{}, len(sp))
	for _, s := range sp {
		ret[s] = struct{}{}
	}

	return ret, nil
}
