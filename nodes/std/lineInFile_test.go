package std

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineInFile_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		node    LineInFile
	}{
		"valid":          {false, LineInFile{Path: "/f", Regexp: "^x=", Line: "x=1"}},
		"empty regexp":   {false, LineInFile{Path: "/f", Line: "x=1"}},
		"missing path":   {true, LineInFile{Line: "x=1"}},
		"missing line":   {true, LineInFile{Path: "/f"}},
		"invalid regexp": {true, LineInFile{Path: "/f", Regexp: "[", Line: "x=1"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			err := cas.node.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLineInFile_apply(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		content  string
		node     LineInFile
	}{
		"append to empty": {
			"x=1\n", "",
			LineInFile{Regexp: "^x=", Line: "x=1"},
		},
		"append no match": {
			"a=2\nx=1\n", "a=2\n",
			LineInFile{Regexp: "^x=", Line: "x=1"},
		},
		"replace match": {
			"x=1\n", "x=0\n",
			LineInFile{Regexp: "^x=", Line: "x=1"},
		},
		"replace last match": {
			"x=0\na=2\nx=1\n", "x=0\na=2\nx=9\n",
			LineInFile{Regexp: "^x=", Line: "x=1"},
		},
		"unchanged": {
			"x=1\n", "x=1\n",
			LineInFile{Regexp: "^x=", Line: "x=1"},
		},
		"adds trailing newline": {
			"x=1\n", "x=0",
			LineInFile{Regexp: "^x=", Line: "x=1"},
		},
		"empty regexp matches line verbatim": {
			"x=1\n", "x=1\n",
			LineInFile{Line: "x=1"},
		},
		"empty regexp appends when line missing": {
			"a=2\nx=1\n", "a=2\n",
			LineInFile{Line: "x=1"},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			result, err := cas.node.apply(cas.content)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, result)
		})
	}
}

func TestLineInFile_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		onDisk   *string
		node     LineInFile
	}{
		"missing file": {true, nil, LineInFile{Regexp: "^x=", Line: "x=1"}},
		"line present": {false, new("x=1\n"), LineInFile{Regexp: "^x=", Line: "x=1"}},
		"line differs": {true, new("x=0\n"), LineInFile{Regexp: "^x=", Line: "x=1"}},
		"line absent":  {true, new("a=2\n"), LineInFile{Regexp: "^x=", Line: "x=1"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file")
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(cas.node.Path, []byte(*cas.onDisk), 0o600))
			}

			needs, _, err := cas.node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}

func TestLineInFile_Execute(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, []byte("a=2\nx=0\n"), 0o600))

	node := LineInFile{Path: path, Regexp: "^x=", Line: "x=1"}
	_, err := node.Execute(nil)
	require.NoError(t, err)

	content, err := os.ReadFile(path) //nolint:gosec // path from t.TempDir
	require.NoError(t, err)
	assert.Equal(t, "a=2\nx=1\n", string(content))

	needs, _, err := node.NeedsExecution(nil)
	require.NoError(t, err)
	assert.False(t, needs, "execute must be idempotent")
}

func TestLineInFile_ExecuteCreate(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file")
	node := LineInFile{Path: path, Regexp: "^x=", Line: "x=1"}
	_, err := node.Execute(nil)
	require.NoError(t, err)

	content, err := os.ReadFile(path) //nolint:gosec // path from t.TempDir
	require.NoError(t, err)
	assert.Equal(t, "x=1\n", string(content))
}
