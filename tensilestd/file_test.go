package tensilestd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParentDirs(t *testing.T) {
	// Use a root that works on any OS (e.g. "/" on Unix, "C:\" on Windows).
	root := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)

	tests := map[string]struct {
		input    string
		expected []string
	}{
		"absolute path with multiple levels": {
			input: filepath.Join(root, "home", "user", "projects", "file.txt"),
			expected: []string{
				filepath.Join(root, "home", "user", "projects"),
				filepath.Join(root, "home", "user"),
				filepath.Join(root, "home"),
				root,
			},
		},
		"two level absolute path": {
			input: filepath.Join(root, "home", "file.txt"),
			expected: []string{
				filepath.Join(root, "home"),
				root,
			},
		},
		"one level absolute path": {
			input:    filepath.Join(root, "file.txt"),
			expected: []string{root},
		},
		"root directory": {
			input:    root,
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
