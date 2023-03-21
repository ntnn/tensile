package nodes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/testutils"
	"github.com/stretchr/testify/require"
)

func TestFile_Validate(t *testing.T) {
	f := new(File)
	f.Target = "/"
	require.Nil(t, f.Validate())
	require.Equal(t, []string{}, f.dirs)

	f2 := new(File)
	f2.Target = "/a/b/c"
	require.Nil(t, f2.Validate())
	require.Equal(t,
		[]string{
			tensile.FormatIdentitierParts(tensile.Path, filepath.FromSlash("/a/b")),
			tensile.FormatIdentitierParts(tensile.Path, filepath.FromSlash("/a")),
			tensile.FormatIdentitierParts(tensile.Path, filepath.FromSlash("/")),
		},
		f2.dirs,
	)

	f3 := new(File)
	f3.Target = "/a"
	require.Nil(t, f3.Validate())
	require.Equal(t,
		[]string{
			tensile.FormatIdentitierParts(tensile.Path, filepath.FromSlash("/")),
		},
		f3.dirs,
	)
}

func TestFile_NeedsExecution(t *testing.T) {
	tempdir, rm, err := testutils.TempDir(t)
	require.Nil(t, err)
	defer rm()

	target := filepath.Join(tempdir, "testfile")

	f := &File{
		Target:  target,
		Content: "Hello, world!",
	}
	require.Nil(t, f.Validate())

	// file does not exist, hence needs creation
	shouldNeedExecution, err := f.NeedsExecution(testutils.Context(t))
	require.Nil(t, err)
	require.True(t, shouldNeedExecution)

	testf, err := os.Create(target)
	require.Nil(t, err)
	_, err = testf.WriteString(f.Content)
	require.Nil(t, err)
	require.Nil(t, testf.Close())

	shouldNotNeedExecution, err := f.NeedsExecution(testutils.Context(t))
	require.Nil(t, err)
	require.False(t, shouldNotNeedExecution)
}

func TestFile_Execute(t *testing.T) {
	tempdir, rm, err := testutils.TempDir(t)
	require.Nil(t, err)
	defer rm()

	target := filepath.Join(tempdir, "testfile")

	f := &File{
		Target:  target,
		Content: "Hello, world!",
	}
	require.Nil(t, f.Validate())

	_, err = f.Execute(testutils.Context(t))
	require.Nil(t, err)

	b, err := os.ReadFile(target)
	require.Nil(t, err)

	require.Equal(t, b, []byte(f.Content))
}
