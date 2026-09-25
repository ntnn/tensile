package std

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParentDirs(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			result := parentDirs(tc.input)
			require.NotNil(t, result, "result should not be nil")
			assert.Equal(t, tc.expected, result, "parentDirs should return the correct parent directories")
		})
	}
}

func TestFile_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		file    File
	}{
		"empty state":   {false, File{Path: "/a"}},
		"present":       {false, File{Path: "/a", State: FilePresent}},
		"absent":        {false, File{Path: "/a", State: FileAbsent}},
		"unknown state": {true, File{Path: "/a", State: "gone"}},
		"missing path":  {true, File{}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			err := cas.file.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFile_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		exists   bool
		state    FileState
	}{
		"missing file":           {true, false, ""},
		"existing file":          {false, true, ""},
		"absent, missing file":   {false, false, FileAbsent},
		"absent, existing file":  {true, true, FileAbsent},
		"present, missing file":  {true, false, FilePresent},
		"present, existing file": {false, true, FilePresent},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "file")
			if cas.exists {
				require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))
			}

			f := &File{Path: path, State: cas.state}
			needs, diff, err := f.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs, diff)
		})
	}
}

func TestFile_Execute(t *testing.T) {
	t.Parallel()

	t.Run("creates a missing file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "file")
		_, err := (&File{Path: path}).Execute(nil)
		require.NoError(t, err)
		assert.FileExists(t, path)
	})

	t.Run("leaves existing content alone", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

		_, err := (&File{Path: path}).Execute(nil)
		require.NoError(t, err)

		content, err := os.ReadFile(path) //nolint:gosec // path from t.TempDir
		require.NoError(t, err)
		assert.Equal(t, "content", string(content), "empty Content must not wipe existing content")
	})

	t.Run("absent removes an existing file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

		_, err := (&File{Path: path, State: FileAbsent}).Execute(nil)
		require.NoError(t, err)
		assert.NoFileExists(t, path)
	})

	t.Run("absent tolerates a missing file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "file")
		_, err := (&File{Path: path, State: FileAbsent}).Execute(nil)
		require.NoError(t, err)
		assert.NoFileExists(t, path)
	})
}

// workIdentities builds the queue and returns all yielded node identities.
// workIdentities builds a queue from node and returns all yielded node identities.
func workIdentities(t *testing.T, node tensile.Identifier) []tensile.Identity {
	t.Helper()

	q := queue.New()
	q.Add(node)
	work, err := q.Build()
	require.NoError(t, err)

	identities := []tensile.Identity{}
	for item := range work.Chan(t.Context()) {
		require.NoError(t, item.Err)
		identities = append(identities, item.Node.Identity())
		work.MarkDone(item.Node, true)
	}
	return identities
}

func TestFile_Queue(t *testing.T) {
	t.Parallel()

	t.Run("present adds file, chmod and chown when set", func(t *testing.T) {
		t.Parallel()

		identities := workIdentities(t, &File{Path: "/a/b/c", FileMode: 0o644, Owner: "root"})
		assert.Contains(t, identities, FileIdentity("/a/b/c"))
		assert.Contains(t, identities, Chmod{Path: "/a/b/c"}.Identity())
		assert.Contains(t, identities, Chown{Path: "/a/b/c"}.Identity())
		assert.NotContains(t, identities, (&FileContent{Path: "/a/b/c"}).Identity(),
			"empty Content needs no FileContent node")
	})

	t.Run("zero mode and ownership add only the file node", func(t *testing.T) {
		t.Parallel()

		identities := workIdentities(t, &File{Path: "/a/b/c"})
		assert.Equal(t, []tensile.Identity{FileIdentity("/a/b/c")}, identities,
			"unset mode and ownership must not be managed")
	})

	t.Run("content adds a FileContent", func(t *testing.T) {
		t.Parallel()

		identities := workIdentities(t, &File{Path: "/a/b/c", Content: "content"})
		assert.Contains(t, identities, (&FileContent{Path: "/a/b/c"}).Identity())
	})

	t.Run("absent adds only the file node", func(t *testing.T) {
		t.Parallel()

		identities := workIdentities(t, &File{Path: "/a/b/c", State: FileAbsent, FileMode: 0o644, Content: "content"})
		assert.Equal(t, []tensile.Identity{FileIdentity("/a/b/c")}, identities,
			"an absent file needs no chmod/chown/content")
	})
}
