package nodes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackage_IsCollision(t *testing.T) {
	p := new(Package)
	p.Name = "netcat"
	p.State = Installed
	require.Nil(t, p.Validate())

	other := new(Package)
	other.Name = "netcat"
	other.State = Latest
	require.Nil(t, p.Validate())

	require.Errorf(t, p.IsCollision(nil), "other is not Package, cannot compare")

	require.Errorf(t, p.IsCollision(other), "conflicting states %q and %q", Installed, Latest)

	other.State = Installed
	require.Nil(t, p.IsCollision(other))
}

func TestPackage_Validate(t *testing.T) {
	p := new(Package)
	require.Equal(t, p.State, Undefined)
	require.Errorf(t, p.Validate(), "package name can not be empty")

	p.Name = "testpackage"
	require.Nil(t, p.Validate())
	require.Equal(t, p.State, Latest)
}
