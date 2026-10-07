package systemd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mapTopology is a [tensile.Topology] backed by a claims map.
type mapTopology map[tensile.Identity]tensile.Identity

func (m mapTopology) Claimer(identity tensile.Identity) (tensile.Identity, bool) {
	claimer, ok := m[identity]
	return claimer, ok
}

// unitsWire returns a wire whose topology holds the claims of units.
func unitsWire(t *testing.T, units ...*Unit) tensile.Wire {
	t.Helper()
	claims := mapTopology{}
	for _, u := range units {
		conflicts, err := u.Conflicts()
		require.NoError(t, err)
		for _, conflict := range conflicts {
			claims[conflict] = u.Identity()
		}
	}
	return &tensile.DefaultWire{
		Ctx:  t.Context(),
		Topo: claims,
	}
}

func TestPruneDropIns_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&PruneDropIns{Name: "foo.service"}).Validate(nil))
	require.Error(t, (&PruneDropIns{}).Validate(nil))
}

func TestPruneDropIns_Path(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		filepath.FromSlash("/etc/systemd/system/foo.service.d"),
		(&PruneDropIns{Name: "foo.service"}).Path(),
	)
	assert.Equal(t,
		filepath.FromSlash("/x/foo.service.d"),
		(&PruneDropIns{Name: "foo.service", Dir: "/x"}).Path(),
	)
}

func TestPruneDropIns_Execute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	managed := &Unit{Name: "foo.service", DropIn: "10-managed", Dir: dir}
	prune := &PruneDropIns{Name: "foo.service", Dir: dir}
	unmanaged := filepath.Join(prune.Path(), "20-unmanaged.conf")
	otherUnit := filepath.Join(dir, "bar.service.d", "20-unmanaged.conf")
	for _, path := range []string{managed.Path(), unmanaged, otherUnit} {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, nil, 0o600))
	}
	wire := unitsWire(t, managed)

	needs, d, err := prune.NeedsExecution(wire)
	require.NoError(t, err)
	require.True(t, needs)
	require.NotNil(t, d)
	assert.Equal(t, "20-unmanaged.conf: present -> (absent)", d.String())

	_, err = prune.Execute(wire)
	require.NoError(t, err)

	assert.FileExists(t, managed.Path(), "a drop-in managed by a Unit must survive pruning")
	assert.NoFileExists(t, unmanaged)
	assert.FileExists(t, otherUnit, "drop-ins of other units must not be pruned")

	needs, _, err = prune.NeedsExecution(wire)
	require.NoError(t, err)
	assert.False(t, needs, "execute must be idempotent")
}

func TestPruneDropIns_DependsOn(t *testing.T) {
	t.Parallel()

	prune := &PruneDropIns{Name: "foo.service"}
	deps, err := prune.DependsOn()
	require.NoError(t, err)
	assert.Contains(t, deps, std.FileIdentity(prune.Path()))
}
