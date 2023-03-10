package nodes

import (
	"fmt"
	"os"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/require"
)

func init() {
	tensile.SetDebugLog()
}

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

func Context(t *testing.T) tensile.Context {
	ctx, err := tensile.NewContext(nil, nil, nil)
	require.Nil(t, err)
	return ctx
}
