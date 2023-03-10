package nodes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_Validate(t *testing.T) {
	cases := map[string]struct {
		input, expect *File
	}{
		"/": {
			input: &File{
				Target: "/",
			},
			expect: &File{
				Target: "/",
				dirs:   []string{},
			},
		},
		"/a/b/c": {
			input: &File{
				Target: "/a/b/c",
			},
			expect: &File{
				Target: "/a/b/c",
				dirs: []string{
					"Path[/a/b]",
					"Path[/a]",
					"Path[/]",
				},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			assert.Nil(t, cas.input.Validate())
			assert.Equal(t, cas.expect, cas.input)
		})
	}
}

func TestFile_NeedsExecution(t *testing.T) {
	tempdir, rm, err := TempDir(t)
	require.Nil(t, err)
	defer rm()

	target := filepath.Join(tempdir, "testfile")

	f := &File{
		Target:  target,
		Content: "Hello, world!",
	}
	require.Nil(t, f.Validate())

	// file does not exist, hence needs creation
	shouldNeedExecution, err := f.NeedsExecution(Context(t))
	require.Nil(t, err)
	require.True(t, shouldNeedExecution)

	testf, err := os.Create(target)
	require.Nil(t, err)
	_, err = testf.WriteString(f.Content)
	require.Nil(t, err)
	require.Nil(t, testf.Close())

	shouldNotNeedExecution, err := f.NeedsExecution(Context(t))
	require.Nil(t, err)
	require.False(t, shouldNotNeedExecution)
}

func TestFile_Execute(t *testing.T) {
	tempdir, rm, err := TempDir(t)
	require.Nil(t, err)
	defer rm()

	target := filepath.Join(tempdir, "testfile")

	f := &File{
		Target:  target,
		Content: "Hello, world!",
	}
	require.Nil(t, f.Validate())

	_, err = f.Execute(Context(t))
	require.Nil(t, err)

	b, err := os.ReadFile(target)
	require.Nil(t, err)

	require.Equal(t, b, []byte(f.Content))
}
