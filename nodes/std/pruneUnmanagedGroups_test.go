package std

import (
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/shadow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneUnmanagedGroups(t *testing.T) {
	t.Parallel()

	groups := []shadow.Group{
		{Name: "root", GID: 0},
		{Name: "wheel", GID: 10},
		{Name: "alice", GID: 1000},
		{Name: "bob", GID: 1001},
		{Name: "team", GID: 2000},
		{Name: "staff", GID: 3000},
	}
	// alice is the primary group of a user and must never be pruned
	users := []shadow.User{{Name: "alice", GID: 1000}}

	cases := map[string]struct {
		node     PruneUnmanagedGroups
		expected []string
	}{
		"default skips system groups": {PruneUnmanagedGroups{}, []string{"bob", "staff"}},
		"include system skips gid 0":  {PruneUnmanagedGroups{IncludeSystem: true}, []string{"wheel", "bob", "staff"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			wire := identityWire(t, GroupIdentity("team"))
			fake := &fakeShadow{
				users:  users,
				groups: slices.Clone(groups),
			}
			cas.node.shadow = fake

			needs, d, err := cas.node.NeedsExecution(wire)
			require.NoError(t, err)
			assert.True(t, needs)
			assert.Equal(t, prunedChanges(cas.expected).String(), d.String())

			_, err = cas.node.Execute(wire)
			require.NoError(t, err)
			for _, name := range cas.expected {
				group, err := fake.Group(name)
				require.NoError(t, err)
				assert.Nil(t, group, "%s must be deleted", name)
			}

			needs, _, err = cas.node.NeedsExecution(wire)
			require.NoError(t, err)
			assert.False(t, needs, "nothing must be left to prune")
		})
	}
}

func TestPruneUnmanagedGroups_DependsOn(t *testing.T) {
	t.Parallel()

	deps, err := (&PruneUnmanagedGroups{}).DependsOn()
	require.NoError(t, err)
	assert.Equal(t, []tensile.Identity{PruneUnmanagedUsersIdentity()}, deps, "users must be pruned before their groups")
}
