package tensilestd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChmod_NeedsExecutionMissing(t *testing.T) {
	t.Parallel()

	c := Chmod{
		Path:     filepath.Join(t.TempDir(), "missing"),
		FileMode: 0o644,
	}

	_, err := c.NeedsExecution(nil)
	assert.Error(t, err, "a missing path should be an error")
}

func TestChmod_ExecuteCycle(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	// restore write bit so TempDir cleanup can remove the file on windows
	t.Cleanup(func() {
		_ = os.Chmod(path, 0o600)
	})

	// 0o444 drops the owner write bit
	c := Chmod{
		Path:     path,
		FileMode: 0o444,
	}

	needs, err := c.NeedsExecution(nil)
	require.NoError(t, err)
	require.True(t, needs, "a file with wrong mode needs execution")

	require.NoError(t, c.Execute(nil))

	needs, err = c.NeedsExecution(nil)
	require.NoError(t, err)
	assert.False(t, needs, "a file with matching mode needs no execution")
}

func TestChmod_NeedsExecutionDirTypeBits(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "dir")
	require.NoError(t, os.Mkdir(path, 0o700))

	// ModeDir in FileMode must not cause drift, should be filtered
	c := Chmod{
		Path:     path,
		FileMode: 0o700 | os.ModeDir,
	}

	needs, err := c.NeedsExecution(nil)
	require.NoError(t, err)
	assert.False(t, needs, "type bits must not be compared")
}

func TestChmod_NeedsExecutionWrongMode(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	c := Chmod{
		Path:     path,
		FileMode: 0o640,
	}

	needs, err := c.NeedsExecution(nil)
	require.NoError(t, err)
	assert.True(t, needs, "differing permission bits need execution")
}
