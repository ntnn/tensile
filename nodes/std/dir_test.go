package std

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipOnWindows skips tests that assert Unix permission bits.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("permission bits not meaningful on windows")
	}
}

func TestDir_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mode     os.FileMode
		expected os.FileMode
	}{
		"empty mode defaults":              {0, DefaultDirMode},
		"explicit mode is not overwritten": {0o700 | os.ModeDir, 0o700 | os.ModeDir},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			d := &Dir{
				Path:     "/some/dir",
				FileMode: cas.mode,
			}
			require.NoError(t, d.Validate(nil))
			assert.Equal(t, cas.expected, d.FileMode)
			assert.Equal(t, d.Path, d.Chmod.Path, "path should propagate to Chmod")
			assert.Equal(t, d.Path, d.Chown.Path, "path should propagate to Chown")
		})
	}
}

func TestDir_NeedsExecutionMissing(t *testing.T) {
	t.Parallel()

	d := &Dir{
		Path: filepath.Join(t.TempDir(), "missing"),
	}
	require.NoError(t, d.Validate(nil))

	needs, err := d.NeedsExecution(nil)
	require.NoError(t, err)
	assert.True(t, needs, "a missing directory needs execution")
}

func TestDir_NeedsExecutionMatchingMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "dir")
	require.NoError(t, os.Mkdir(path, 0o700))

	d := &Dir{
		Path:     path,
		FileMode: 0o700 | os.ModeDir,
	}
	require.NoError(t, d.Validate(nil))

	needs, err := d.NeedsExecution(nil)
	require.NoError(t, err)
	assert.False(t, needs, "an existing directory with matching mode needs no execution")
}

func TestDir_NeedsExecutionWrongMode(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	path := filepath.Join(t.TempDir(), "dir")
	require.NoError(t, os.Mkdir(path, 0o700))

	d := &Dir{Path: path}
	require.NoError(t, d.Validate(nil))

	needs, err := d.NeedsExecution(nil)
	require.NoError(t, err)
	assert.True(t, needs, "an existing directory with wrong mode needs execution")
}

func TestDir_NeedsExecutionNotADir(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o600))

	d := &Dir{Path: path}
	require.NoError(t, d.Validate(nil))

	_, err := d.NeedsExecution(nil)
	assert.Error(t, err, "a non-directory at the path should be an error")
}

func TestDir_ExecuteCreatesNested(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "a", "b", "c")
	d := &Dir{
		Path:     path,
		FileMode: 0o700 | os.ModeDir,
	}
	require.NoError(t, d.Validate(nil))
	require.NoError(t, d.Execute(nil))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	}
}

func TestDir_ExecuteFixesMode(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	path := filepath.Join(t.TempDir(), "dir")
	require.NoError(t, os.Mkdir(path, 0o700))
	d := &Dir{Path: path}
	require.NoError(t, d.Validate(nil))
	require.NoError(t, d.Execute(nil))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}
