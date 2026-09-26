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

	needs, _, err := s.NeedsExecution(nil)
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

	needs, _, err := s.NeedsExecution(nil)
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

	needs, _, err := s.NeedsExecution(nil)
	require.NoError(t, err)
	assert.True(t, needs, "a symlink with wrong target needs execution")
}

func TestSymlink_NeedsExecutionNotASymlink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, path string)
		current string
	}{
		{
			name: "file",
			setup: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.WriteFile(path, nil, 0o600))
			},
			current: "File",
		},
		{
			name: "directory",
			setup: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.Mkdir(path, 0o700))
			},
			current: "Directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), tt.name)
			tt.setup(t, path)

			s := &Symlink{
				Path:   path,
				Target: "target",
			}

			needs, d, err := s.NeedsExecution(nil)
			require.NoError(t, err)
			assert.True(t, needs, "a non-symlink at the path needs execution")
			assert.Equal(t, "target: "+tt.current+" -> target", d.String())
		})
	}
}

func TestSymlink_ExecuteCreates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "link")
	s := &Symlink{
		Path:   path,
		Target: "target",
	}

	_, err := s.Execute(nil)
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
	_, err := s.Execute(nil)
	require.NoError(t, err)

	target, err := os.Readlink(path)
	require.NoError(t, err)
	assert.Equal(t, "target", target)
}

func TestSymlink_ExecuteReplacesDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Mkdir(path, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(path, "file"), nil, 0o600))

	s := &Symlink{
		Path:   path,
		Target: "target",
	}
	_, err := s.Execute(nil)
	if err != nil && runtime.GOOS == "windows" {
		t.Skipf("cannot create symlinks: %v", err)
	}
	require.NoError(t, err)

	target, err := os.Readlink(path)
	require.NoError(t, err)
	assert.Equal(t, "target", target)
}
