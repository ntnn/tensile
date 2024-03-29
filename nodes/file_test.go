package nodes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/require"
)

func TestFile_Validate(t *testing.T) {
	f := new(File)
	f.Target = "/"
	require.Nil(t, f.Validate())
	require.Equal(t, []string{}, f.parentDirs.ParentDirs)

	f2 := new(File)
	f2.Target = "/a/b/c"
	require.Nil(t, f2.Validate())
	require.Equal(t,
		[]string{
			tensile.FormatIdentity(tensile.Path, filepath.FromSlash("/a/b")),
			tensile.FormatIdentity(tensile.Path, filepath.FromSlash("/a")),
			tensile.FormatIdentity(tensile.Path, filepath.FromSlash("/")),
		},
		f2.parentDirs.ParentDirs,
	)

	f3 := new(File)
	f3.Target = "/a"
	require.Nil(t, f3.Validate())
	require.Equal(t,
		[]string{
			tensile.FormatIdentity(tensile.Path, filepath.FromSlash("/")),
		},
		f3.parentDirs.ParentDirs,
	)
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
