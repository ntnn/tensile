//go:build !windows

package facts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOSReleaseFileWithPath(t *testing.T) {
	s, err := os.MkdirTemp("", "tensile-facts-test")
	require.Nil(t, err)
	defer func() { require.Nil(t, os.RemoveAll(s)) }()

	p := filepath.Join(s, "os-release")

	exampleData := `NAME=testdistro
PRETTY_NAME="pretty distro name"
CPE_NAME=cpe:/o:distro:name:5
ID_LIKE=distroFamily1 distroFamily2
VERSION=""
VERSION_ID=
`

	require.Nil(t, os.WriteFile(p, []byte(exampleData), 0600))

	rel, err := newOSReleaseFileWithPath(p)
	require.Nil(t, err)
	require.Equal(t, "testdistro", rel.Name)
	require.Equal(t, "pretty distro name", rel.PrettyName)
	require.Equal(t, "cpe:/o:distro:name:5", rel.CPEName)
	require.Equal(t, []string{"distroFamily1", "distroFamily2"}, rel.IDLike)
	require.Equal(t, "", rel.Version)
	require.Equal(t, "", rel.VersionID)
}
