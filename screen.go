package main

import (
	"github.com/kndndrj/sway-reflex/internal/core"
)

// screen represents a working unit of an output.
type screen struct {
	width  int
	height int

	prefferedWindowWidth  int
	prefferedWindowHeight int

	defaultGapHorizontal int
	defaultGapVertical   int

	// which way is the top level container being split?
	direction core.Direction
}

// IsFilled returns true if based on provided container dimensions screen is fully filled.
func (s *screen) IsFilled(cwidth, cheight int) bool {
	if cwidth < s.width {
		return false
	}
	if cheight < s.height {
		return false
	}
	return true
}

// IsFilled returns true if based on provided container dimensions screen is fully filled.
func (s *screen) Direction() core.Direction {
	return s.direction
}

// newScreen retrieves or initializes and then returns a screen.
// if no preffered height is given, width is used for both dimensions.
func newScreen(out *core.Output, cfg *config) *screen {
	// calculate pixel dimensions from actual size and prefferences
	prefferedWindowWidth := (cfg.PhysicalWindowWidth * out.Width) / out.PhysicalWidth
	prefferedWindowHeight := (cfg.PhysicalWindowHeight * out.Height) / out.PhysicalHeight

	width := out.Width - (cfg.DefaultGapHorizontal * 2)
	height := out.Height - (cfg.DefaultGapVertical * 2)

	dir := core.DirectionHorizontal
	if height-prefferedWindowHeight > width-prefferedWindowWidth {
		dir = core.DirectionVertical
	}

	return &screen{
		width:  width,
		height: height,

		prefferedWindowWidth:  prefferedWindowWidth,
		prefferedWindowHeight: prefferedWindowHeight,

		defaultGapHorizontal: cfg.DefaultGapHorizontal,
		defaultGapVertical:   cfg.DefaultGapVertical,

		direction: dir,
	}
}

// CalculateContainerDimensions adjusts top level container (aka. all windows combined) dimensions,
// so that they fit on the screen.
func (s *screen) CalculateContainerDimensions(numOfTopLevelContainers int) (width, height int) {
	if numOfTopLevelContainers < 1 {
		return 0, 0
	}

	if s.direction == core.DirectionHorizontal {
		fullWidth := s.prefferedWindowWidth * numOfTopLevelContainers

		// width fits on screen
		if fullWidth <= s.width {
			height := s.prefferedWindowHeight
			if s.height < height {
				height = s.height
			}
			return fullWidth, height
		}

		// width doesn't fit on screen and height doesn't as well
		if s.height <= s.prefferedWindowHeight {
			return s.width, s.height
		}

		// try to compensate off-screen surface with height
		offScreenSurface := (fullWidth - s.width) * s.prefferedWindowHeight
		deltaHeight := offScreenSurface / s.width

		height := s.prefferedWindowHeight + deltaHeight
		if s.height <= height {
			return s.width, s.height
		}

		return s.width, height
	}

	// else if directionVertical

	fullHeight := s.prefferedWindowHeight * numOfTopLevelContainers

	// height fits on screen
	if fullHeight <= s.height {
		width := s.prefferedWindowWidth
		if s.width < width {
			width = s.width
		}
		return width, fullHeight
	}

	// height doesn't fit on screen and width doesn't as well
	if s.width <= s.prefferedWindowWidth {
		return s.width, s.height
	}

	// try to compensate off-screen surface with width
	offScreenSurface := (fullHeight - s.height) * s.prefferedWindowWidth
	deltaWidth := offScreenSurface / s.height

	width = s.prefferedWindowWidth + deltaWidth
	if s.width <= width {
		return s.width, s.height
	}

	return width, s.height
}

// CalculateOuterGaps calculates gaps between the edge of the screen and top level container.
func (s *screen) CalculateOuterGaps(containerWidth, containerHeight int) (horizontal, vertical int) {
	widthDiff := s.width - containerWidth
	if widthDiff < 0 {
		widthDiff = 0
	}
	heightDiff := s.height - containerHeight
	if heightDiff < 0 {
		heightDiff = 0
	}

	gaph := (widthDiff / 2) + s.defaultGapHorizontal
	gapv := (heightDiff / 2) + s.defaultGapVertical

	return gaph, gapv
}
