package shadow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gosec // passwd fixture, no credentials
const testPasswd = `root:x:0:0:root:/root:/bin/bash
# comment
alice:x:1000:1000:Alice:/home/alice:/bin/zsh
bob:x:1001:1001::/home/bob:/bin/sh
orphan:x:1002:4242::/home/orphan:/bin/sh
broken:x:abc:1000::/home/broken:/bin/sh

+nisuser::::::
+::::::
`

const testGroup = `root:x:0:
wheel:x:10:alice,bob
alice:x:1000:
bob:x:1001:
empty:x:2000:
# comment
+nisgroup:::
`

// testFiles returns [files] reading testPasswd and testGroup.
func testFiles(t *testing.T) files {
	t.Helper()
	dir := t.TempDir()
	f := files{
		passwdPath: filepath.Join(dir, "passwd"),
		groupPath:  filepath.Join(dir, "group"),
	}
	require.NoError(t, os.WriteFile(f.passwdPath, []byte(testPasswd), 0o600))
	require.NoError(t, os.WriteFile(f.groupPath, []byte(testGroup), 0o600))
	return f
}

func TestFiles_Users(t *testing.T) {
	t.Parallel()

	users, err := testFiles(t).Users()
	require.NoError(t, err)

	expected := []User{
		{
			Name:  "root",
			UID:   0,
			GID:   0,
			Group: "root",
			Home:  "/root",
			Shell: "/bin/bash",
		},
		{
			Name:   "alice",
			UID:    1000,
			GID:    1000,
			Group:  "alice",
			Groups: []string{"wheel"},
			Home:   "/home/alice",
			Shell:  "/bin/zsh",
		},
		{
			Name:   "bob",
			UID:    1001,
			GID:    1001,
			Group:  "bob",
			Groups: []string{"wheel"},
			Home:   "/home/bob",
			Shell:  "/bin/sh",
		},
		{
			Name:  "orphan",
			UID:   1002,
			GID:   4242,
			Group: "",
			Home:  "/home/orphan",
			Shell: "/bin/sh",
		},
		{
			// moby/sys/user parses malformed IDs as 0
			Name:  "broken",
			UID:   0,
			GID:   1000,
			Group: "alice",
			Home:  "/home/broken",
			Shell: "/bin/sh",
		},
	}
	assert.Equal(t, expected, users, "comments and NIS compat entries must be skipped")
}

func TestFiles_Groups(t *testing.T) {
	t.Parallel()

	groups, err := testFiles(t).Groups()
	require.NoError(t, err)

	expected := []Group{
		{Name: "root", GID: 0, Members: []string{}},
		{Name: "wheel", GID: 10, Members: []string{"alice", "bob"}},
		{Name: "alice", GID: 1000, Members: []string{}},
		{Name: "bob", GID: 1001, Members: []string{}},
		{Name: "empty", GID: 2000, Members: []string{}},
	}
	assert.Equal(t, expected, groups, "comments and NIS compat entries must be skipped")
}

func TestFiles_User(t *testing.T) {
	t.Parallel()

	f := testFiles(t)

	t.Run("existing user", func(t *testing.T) {
		t.Parallel()
		entry, err := f.User("alice")
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, 1000, entry.UID)
		assert.Equal(t, []string{"wheel"}, entry.Groups)
	})

	t.Run("missing user returns nil", func(t *testing.T) {
		t.Parallel()
		entry, err := f.User("nobody")
		require.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("NIS compat entry is not a user", func(t *testing.T) {
		t.Parallel()
		entry, err := f.User("+nisuser")
		require.NoError(t, err)
		assert.Nil(t, entry)
	})
}

func TestFiles_Group(t *testing.T) {
	t.Parallel()

	f := testFiles(t)

	t.Run("existing group", func(t *testing.T) {
		t.Parallel()
		entry, err := f.Group("wheel")
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, 10, entry.GID)
		assert.Equal(t, []string{"alice", "bob"}, entry.Members)
	})

	t.Run("missing group returns nil", func(t *testing.T) {
		t.Parallel()
		entry, err := f.Group("nogroup")
		require.NoError(t, err)
		assert.Nil(t, entry)
	})
}

func TestFiles_missingFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f := files{
		passwdPath: filepath.Join(dir, "passwd"),
		groupPath:  filepath.Join(dir, "group"),
	}

	_, err := f.Users()
	require.Error(t, err, "missing passwd must not read as no users")

	_, err = f.Groups()
	require.Error(t, err, "missing group must not read as no groups")
}
