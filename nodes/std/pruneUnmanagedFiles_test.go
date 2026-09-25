package std

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// claimsWire returns a wire whose topology claims the [FileIdentity] of each path.
func claimsWire(t *testing.T, paths ...string) tensile.Wire {
	t.Helper()
	claims := mapTopology{}
	for _, path := range paths {
		claims[FileIdentity(path)] = FileIdentity(path)
	}
	return &tensile.DefaultWire{
		Ctx:  t.Context(),
		Topo: claims,
	}
}

// mapTopology is a [tensile.Topology] backed by a claims map.
type mapTopology map[tensile.Identity]tensile.Identity

func (m mapTopology) Claimer(identity tensile.Identity) (tensile.Identity, bool) {
	claimer, ok := m[identity]
	return claimer, ok
}

func TestPruneUnmanagedFiles_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		files    []string
		claimed  []string
	}{
		"empty dir":            {false, nil, nil},
		"all managed":          {false, []string{"a", "b"}, []string{"a", "b"}},
		"unmanaged file":       {true, []string{"a", "b"}, []string{"a"}},
		"nothing managed":      {true, []string{"a"}, nil},
		"claim without a file": {false, nil, []string{"a"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for _, name := range cas.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0o600))
			}
			claimed := make([]string, len(cas.claimed))
			for i, name := range cas.claimed {
				claimed[i] = filepath.Join(dir, name)
			}

			p := &PruneUnmanagedFiles{Dir: dir}
			needs, diff, err := p.NeedsExecution(claimsWire(t, claimed...))
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs, diff)
		})
	}
}

func TestPruneUnmanagedFiles_NeedsExecutionMissingDir(t *testing.T) {
	t.Parallel()

	p := &PruneUnmanagedFiles{Dir: filepath.Join(t.TempDir(), "missing")}
	needs, _, err := p.NeedsExecution(claimsWire(t))
	require.NoError(t, err)
	assert.False(t, needs, "a missing directory has nothing to prune")
}

func TestPruneUnmanagedFiles_NeedsExecutionNoTopology(t *testing.T) {
	t.Parallel()

	p := &PruneUnmanagedFiles{Dir: t.TempDir()}
	_, _, err := p.NeedsExecution(&tensile.DefaultWire{Ctx: t.Context()})
	require.ErrorIs(t, err, tensile.ErrNoTopology, "pruning without claim knowledge must error, not delete")
}

func TestPruneUnmanagedFiles_Execute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	managed := filepath.Join(dir, "managed")
	unmanaged := filepath.Join(dir, "unmanaged")
	unmanagedDir := filepath.Join(dir, "unmanagedDir")
	require.NoError(t, os.WriteFile(managed, nil, 0o600))
	require.NoError(t, os.WriteFile(unmanaged, nil, 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(unmanagedDir, "sub"), 0o700))

	p := &PruneUnmanagedFiles{Dir: dir}
	_, err := p.Execute(claimsWire(t, managed))
	require.NoError(t, err)

	assert.FileExists(t, managed, "a claimed file must survive pruning")
	assert.NoFileExists(t, unmanaged)
	assert.NoDirExists(t, unmanagedDir, "an unmanaged directory must be removed with its contents")
}
