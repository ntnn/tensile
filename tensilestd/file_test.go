package tensilestd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParentDirs(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"absolute path with multiple levels": {
			input:    "/home/user/projects/file.txt",
			expected: []string{"/home/user/projects", "/home/user", "/home", "/"},
		},
		"two level absolute path": {
			input:    "/home/file.txt",
			expected: []string{"/home", "/"},
		},
		"one level absolute path": {
			input:    "/file.txt",
			expected: []string{"/"},
		},
		"root directory": {
			input:    "/",
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := parentDirs(tc.input)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tc.expected, result, "parentDirs should return the correct parent directories")
		})
	}
}
