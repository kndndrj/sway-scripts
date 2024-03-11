package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

type config struct {
	// Preffered physical dimensions of windows.
	PhysicalWindowWidth  int // all physical dimensions are in [mm]
	PhysicalWindowHeight int

	// Default outer gaps
	DefaultGapHorizontal int // all physical dimensions are in [mm]
	DefaultGapVertical   int
}

func parseConfig() (*config, error) {
	prefferedWindowSize := flag.String("window_size", "500x300", "Preffered window size. <width>x<height> in [mm].")
	defaultGaps := flag.Int("default_gaps", 0, "Default outer gaps [px].")

	flag.Parse()

	width, height, err := parseWindowSize(*prefferedWindowSize)
	if err != nil {
		return nil, err
	}

	gaps, err := parseGaps(*defaultGaps)
	if err != nil {
		return nil, err
	}

	return &config{
		PhysicalWindowWidth:  width,
		PhysicalWindowHeight: height,

		DefaultGapHorizontal: gaps,
		DefaultGapVertical:   gaps,
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
