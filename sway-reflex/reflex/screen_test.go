package reflex

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kndndrj/sway-scripts/internal/core"
)

func TestNewScreen(t *testing.T) {
	r := require.New(t)

	out := &core.Output{
		Name:           "output-1",
		Width:          2000,
		Height:         1000,
		PhysicalWidth:  200,
		PhysicalHeight: 100,
	}

	cfg := &Config{
		PhysicalWindowWidth:  100,
		PhysicalWindowHeight: 50,
		DefaultGapHorizontal: 5,
		DefaultGapVertical:   5,
		DisabledWorkspaces:   map[int]struct{}{},
	}

	scr := NewScreen(out, cfg)

	r.Equal(1990, scr.width) // output minus gaps
	r.Equal(990, scr.height)

	r.Equal(1000, scr.prefferedWindowWidth)
	r.Equal(500, scr.prefferedWindowHeight)

	r.Equal(scr.direction, core.DirectionHorizontal)
}

func TestScreen_CalculateDimensionsAndGaps(t *testing.T) {
	testCases := []struct {
		comment string

		screenWidth  int
		screenHeight int

		prefferedWindowWidth  int
		prefferedWindowHeight int

		numberOfWindows int

		expectedWidth  int
		expectedHeight int
	}{
		{
			// +-----------------------------------------------+
			// |                                               |
			// |                                               |
			// |                                               |
			// |                                               |
			// |                                               |
			// |                                               |
			// |                                               |
			// |                                               |
			// |                                               |
			// +-----------------------------------------------+
			comment:               "no open windows",
			screenWidth:           2000,
			screenHeight:          1000,
			prefferedWindowWidth:  500,
			prefferedWindowHeight: 250,
			numberOfWindows:       0,

			expectedWidth:  0,
			expectedHeight: 0,
		},

		{
			// +-----------------------------------------------+
			// |                                               |
			// |                                               |
			// |                +-------------+                |
			// |                |             |                |
			// |                |             |                |
			// |                |             |                |
			// |                +-------------+                |
			// |                                               |
			// |                                               |
			// +-----------------------------------------------+
			comment:               "horizontal: one window that fits",
			screenWidth:           2000,
			screenHeight:          1000,
			prefferedWindowWidth:  500,
			prefferedWindowHeight: 250,
			numberOfWindows:       1,

			expectedWidth:  500,
			expectedHeight: 250,
		},
		{
			// +-----------------------------------------------+
			// |                                               |
			// |                                               |
			// |  +-------------+-------------+-------------+  |
			// |  |             |             |             |  |
			// |  |             |             |             |  |
			// |  |             |             |             |  |
			// |  +-------------+-------------+-------------+  |
			// |                                               |
			// |                                               |
			// +-----------------------------------------------+
			comment:               "horizontal: three windows that fit",
			screenWidth:           2000,
			screenHeight:          1000,
			prefferedWindowWidth:  500,
			prefferedWindowHeight: 250,
			numberOfWindows:       3,

			expectedWidth:  1500,
			expectedHeight: 250,
		},
		{
			// +-----------------------------------------------+
			// |                                               |
			// +--------+---------+---------+---------+--------+
			// |        |         |         |         |        |
			// |        |         |         |         |        |
			// |        |         |         |         |        |
			// |        |         |         |         |        |
			// |        |         |         |         |        |
			// +--------+---------+---------+---------+--------+
			// |                                               |
			// +-----------------------------------------------+
			comment:               "horizontal: five windows that partially fit",
			screenWidth:           2000,
			screenHeight:          1000,
			prefferedWindowWidth:  500,
			prefferedWindowHeight: 250,
			numberOfWindows:       5,

			expectedWidth:  2000,
			expectedHeight: 312,
		},
		{
			// +---+---+---+---+---+---+---+---+---+---+---+---+
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// |   |   |   |   |   |   |   |   |   |   |   |   |
			// +---+---+---+---+---+---+---+---+---+---+---+---+
			comment:               "horizontal: twenty windows that don't fit",
			screenWidth:           2000,
			screenHeight:          1000,
			prefferedWindowWidth:  500,
			prefferedWindowHeight: 250,
			numberOfWindows:       20,

			expectedWidth:  2000,
			expectedHeight: 1000,
		},
		{
			// +-------------------+
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// |  +-------------+  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  +-------------+  |
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// |                   |
			// +-------------------+
			comment:               "vertical: one window that fits",
			screenWidth:           1000,
			screenHeight:          2000,
			prefferedWindowWidth:  700,
			prefferedWindowHeight: 500,
			numberOfWindows:       1,

			expectedWidth:  700,
			expectedHeight: 500,
		},
		{
			// +-------------------+
			// |                   |
			// |  +-------------+  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  +-------------+  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  +-------------+  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  |             |  |
			// |  +-------------+  |
			// |                   |
			// +-------------------+
			comment:               "vertical: three windows that fit",
			screenWidth:           1000,
			screenHeight:          2000,
			prefferedWindowWidth:  700,
			prefferedWindowHeight: 500,
			numberOfWindows:       3,

			expectedWidth:  700,
			expectedHeight: 1500,
		},
		{
			// +-+---------------+-+
			// | |               | |
			// | |               | |
			// | |               | |
			// | +---------------+ |
			// | |               | |
			// | |               | |
			// | |               | |
			// | +---------------+ |
			// | |               | |
			// | |               | |
			// | |               | |
			// | +---------------+ |
			// | |               | |
			// | |               | |
			// | |               | |
			// | +---------------+ |
			// | |               | |
			// | |               | |
			// | |               | |
			// +-+---------------+-+
			comment:               "vertical: five windows that partially fit",
			screenWidth:           1000,
			screenHeight:          2000,
			prefferedWindowWidth:  700,
			prefferedWindowHeight: 500,
			numberOfWindows:       5,

			expectedWidth:  875,
			expectedHeight: 2000,
		},
		{
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			// |                   |
			// |                   |
			// +-------------------+
			comment:               "vertical: seven windows that don't fit",
			screenWidth:           1000,
			screenHeight:          2000,
			prefferedWindowWidth:  700,
			prefferedWindowHeight: 500,
			numberOfWindows:       7,

			expectedWidth:  1000,
			expectedHeight: 2000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.comment, func(t *testing.T) {
			dir := core.DirectionHorizontal
			if tc.screenHeight-tc.prefferedWindowHeight > tc.screenWidth-tc.prefferedWindowWidth {
				dir = core.DirectionVertical
			}

			scr := &Screen{
				width:                 tc.screenWidth,
				height:                tc.screenHeight,
				prefferedWindowWidth:  tc.prefferedWindowWidth,
				prefferedWindowHeight: tc.prefferedWindowHeight,
				defaultGapHorizontal:  0,
				defaultGapVertical:    0,
				direction:             dir,
			}

			width, height := scr.CalculateContainerDimensions(tc.numberOfWindows)
			t.Log(width, height)

			require.Equal(t, tc.expectedWidth, width, "Expected and actual widths differ.")
			require.Equal(t, tc.expectedHeight, height, "Expected and actual heights differ.")

			gapsh, gapsv := scr.CalculateOuterGaps(width, height)

			require.Equal(t, (tc.screenWidth-tc.expectedWidth)/2, gapsh, "Expected and actual horizontal gaps differ.")
			require.Equal(t, (tc.screenHeight-tc.expectedHeight)/2, gapsv, "Expected and actual vertical gaps differ.")
		})
	}
}
