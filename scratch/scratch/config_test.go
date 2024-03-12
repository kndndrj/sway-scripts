package scratch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTermCmd(t *testing.T) {
	appID := "example_app_id"

	testCases := []struct {
		input       string
		expected    string
		expectedErr error
	}{
		{
			input:       "kitty --class {{ app_id }}",
			expected:    "kitty --class " + appID,
			expectedErr: nil,
		},
		{
			input:       "kitty",
			expected:    "",
			expectedErr: errTermCmdNoAppID,
		},
		{
			input:       "",
			expected:    "",
			expectedErr: errTermCmdEmpty,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			r := require.New(t)

			gen, err := parseTermCmd(tc.input)
			if tc.expectedErr != nil {
				r.NotNil(err, "Expected error, got nil")
				r.Contains(err.Error(), tc.expectedErr.Error(), "Expected errors differ")
				return
			} else {
				r.NoError(err)
			}

			cmd, err := gen(appID)
			r.NoError(err)

			r.Equal(tc.expected, cmd)
		})
	}
}
