package testutils

import (
	"fmt"
	"os"
	"testing"
)

// TempDir created a temporary directory and returns the path an
// a function to remove the directory.
func TempDir(t *testing.T) (string, func(), error) {
	s, err := os.MkdirTemp("", "tensiletest")
	if err != nil {
		return "", nil, fmt.Errorf("error creating test directory: %w", err)
	}

	fn := func() {
		if err := os.RemoveAll(s); err != nil {
			t.Errorf("error removing test directory %q: %v", s, err)
		}
	}

	return s, fn, nil
}
