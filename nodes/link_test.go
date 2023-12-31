package nodes

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLink_NeedsExecution(t *testing.T) {
	for _, linkType := range []LinkType{Softlink, Hardlink} {
		t.Run(string(linkType), func(t *testing.T) {
			tempdir, rm, err := TempDir(t)
			require.Nil(t, err)
			defer rm()

			// test file being linked to
			source := &File{
				Target:  filepath.Join(tempdir, "sourceFile"),
				Content: "Hello, world!",
			}
			require.Nil(t, source.Validate())

			// test target
			target := &Link{
				Target: filepath.Join(tempdir, "targetSymlink"),
				Source: source.Target,
				Type:   linkType,
			}
			require.Nil(t, target.Validate())

			shouldNeedExecution, err := target.NeedsExecution(Context(t))
			// should return execution necessity as source and target do
			// not exist
			require.Nil(t, err)
			require.True(t, shouldNeedExecution)

			// create source file
			_, err = source.Execute(Context(t))
			require.Nil(t, err)

			shouldNeedExecution2, err := target.NeedsExecution(Context(t))
			// should return execution necessity as target does not
			// exist
			require.Nil(t, err)
			require.True(t, shouldNeedExecution2)
		})
	}
}

func TestLink_Execute(t *testing.T) {
	for _, linkType := range []LinkType{Softlink, Hardlink} {
		t.Run(string(linkType), func(t *testing.T) {
			tempdir, rm, err := TempDir(t)
			require.Nil(t, err)
			defer rm()

			// test file being linked to
			source := &File{
				Target:  filepath.Join(tempdir, "sourceFile"),
				Content: "Hello, world!",
			}
			require.Nil(t, source.Validate())
			_, err = source.Execute(Context(t))
			require.Nil(t, err)

			// test target
			target := &Link{
				Target: filepath.Join(tempdir, "targetSymlink"),
				Source: source.Target,
				Type:   linkType,
			}
			require.Nil(t, target.Validate())

			_, err = target.Execute(Context(t))
			require.Nil(t, err)

			// double check with NeedsExecution
			b, err := target.NeedsExecution(Context(t))
			require.Nil(t, err)
			require.False(t, b)
		})
	}
}
