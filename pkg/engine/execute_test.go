package engine

import (
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedBackend_Load(t *testing.T) {
	t.Parallel()

	node := tensile.AsIdentity("test", "name", "node")
	dep := tensile.AsIdentity("test", "name", "dep")
	other := tensile.AsIdentity("test", "name", "other")

	backend := storage.NewDefaultBackend[tensile.Identity]()
	require.NoError(t, backend.Store(dep, 42))
	require.NoError(t, backend.Store(other, 7))

	scoped := &scopedBackend{
		backend: backend,
		node:    node,
		deps:    []tensile.Identity{dep},
	}

	got, err := scoped.Load(dep)
	require.NoError(t, err)
	assert.Equal(t, 42, got)

	_, err = scoped.Load(other)
	require.ErrorContains(t, err, "not a dependency", "undeclared dependency must error")

	require.ErrorContains(t, scoped.Store(dep, 1), "read-only", "nodes must not store outputs directly")
}
