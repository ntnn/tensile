package std

import (
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/shadow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identityWire returns a wire whose topology claims each identity.
func identityWire(t *testing.T, identities ...tensile.Identity) tensile.Wire {
	t.Helper()
	claims := mapTopology{}
	for _, identity := range identities {
		claims[identity] = identity
	}
	return &tensile.DefaultWire{
		Ctx:  t.Context(),
		Topo: claims,
	}
}

func TestPruneUnmanagedUsers(t *testing.T) {
	t.Parallel()

	users := []shadow.User{
		{Name: "root", UID: 0},
		{Name: "daemon", UID: 2},
		{Name: "carol", UID: 500},
		{Name: "alice", UID: 1000},
		{Name: "bob", UID: 1001},
		{Name: "nobody", UID: 65534},
	}
	low := 500

	cases := map[string]struct {
		node     PruneUnmanagedUsers
		expected []string
	}{
		"default skips system users": {PruneUnmanagedUsers{}, []string{"bob"}},
		"include system skips root": {
			PruneUnmanagedUsers{IncludeSystem: true},
			[]string{"daemon", "carol", "bob", "nobody"},
		},
		"custom min": {PruneUnmanagedUsers{IDMin: &low}, []string{"carol", "bob"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			wire := identityWire(t, UserIdentity("alice"))
			fake := &fakeShadow{users: slices.Clone(users)}
			cas.node.shadow = fake

			needs, d, err := cas.node.NeedsExecution(wire)
			require.NoError(t, err)
			assert.True(t, needs)
			assert.Equal(t, prunedChanges(cas.expected).String(), d.String())

			_, err = cas.node.Execute(wire)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, fake.removedHomes, "pruned users must be deleted with their home")

			needs, _, err = cas.node.NeedsExecution(wire)
			require.NoError(t, err)
			assert.False(t, needs, "nothing must be left to prune")
		})
	}
}
