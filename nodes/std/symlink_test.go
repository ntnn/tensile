package std

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustSymlink creates a symlink or skips when the platform denies it, e.g. windows without developer mode.
func mustSymlink(t *testing.T, target, path string) {
	t.Helper()
	err := os.Symlink(target, path)
	if err != nil && runtime.GOOS == "windows" {
		t.Skipf("cannot create symlinks: %v", err)
	}
	require.NoError(t, err)
}

func TestSymlink_NeedsExecutionMissing(t *testing.T) {
	t.Parallel()

	s := &Symlink{
		Path:   filepath.Join(t.TempDir(), "missing"),
		Target: "target",
	}

	needs, err := s.NeedsExecution(nil)
	require.NoError(t, err)
	assert.True(t, needs, "a missing symlink needs execution")
}

func TestSymlink_NeedsExecutionMatching(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "link")
	mustSymlink(t, "target", path)

	s := &Symlink{
		Path:   path,
		Target: "target",
	}

	needs, err := s.NeedsExecution(nil)
	require.NoError(t, err)
	assert.False(t, needs, "a symlink with matching target needs no execution")
}

func TestSymlink_NeedsExecutionWrongTarget(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "link")
	mustSymlink(t, "other", path)

	s := &Symlink{
		Path:   path,
		Target: "target",
	}

	needs, err := s.NeedsExecution(nil)
	require.NoError(t, err)
	assert.True(t, needs, "a symlink with wrong target needs execution")
}

func TestSymlink_NeedsExecutionNotASymlink(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	s := &Symlink{
		Path:   path,
		Target: "target",
	}

	_, err := s.NeedsExecution(nil)
	assert.Error(t, err, "a non-symlink at the path should be an error")
}

func TestSymlink_ExecuteCreates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "link")
	s := &Symlink{
		Path:   path,
		Target: "target",
	}

	err := s.Execute(nil)
	if err != nil && runtime.GOOS == "windows" {
		t.Skipf("cannot create symlinks: %v", err)
	}
	require.NoError(t, err)

	target, err := os.Readlink(path)
	require.NoError(t, err)
	assert.Equal(t, "target", target)
}

func TestSymlink_ExecuteReplaces(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "link")
	mustSymlink(t, "other", path)

	s := &Symlink{
		Path:   path,
		Target: "target",
	}
	require.NoError(t, s.Execute(nil))

	target, err := os.Readlink(path)
	require.NoError(t, err)
	assert.Equal(t, "target", target)
}
