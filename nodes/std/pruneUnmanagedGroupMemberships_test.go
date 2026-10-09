package std

import (
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/shadow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneUnmanagedGroupMemberships(t *testing.T) {
	t.Parallel()

	groups := []shadow.Group{
		{Name: "wheel", GID: 10, Members: []string{"alice", "bob"}},
		{Name: "team", GID: 2000, Members: []string{"alice", "bob", "carol"}},
	}

	cases := map[string]struct {
		node     PruneUnmanagedGroupMemberships
		wantDiff string
		expected []shadow.Group
	}{
		"default skips system groups": {
			PruneUnmanagedGroupMemberships{},
			"team/bob: present -> (absent)\nteam/carol: present -> (absent)",
			[]shadow.Group{
				{Name: "wheel", GID: 10, Members: []string{"alice", "bob"}},
				{Name: "team", GID: 2000, Members: []string{"alice"}},
			},
		},
		"include system": {
			PruneUnmanagedGroupMemberships{IncludeSystem: true},
			"wheel/alice: present -> (absent)\nwheel/bob: present -> (absent)\n" +
				"team/bob: present -> (absent)\nteam/carol: present -> (absent)",
			[]shadow.Group{
				{Name: "wheel", GID: 10, Members: []string{}},
				{Name: "team", GID: 2000, Members: []string{"alice"}},
			},
		},
		"single group": {
			PruneUnmanagedGroupMemberships{Group: "wheel", IncludeSystem: true},
			"wheel/alice: present -> (absent)\nwheel/bob: present -> (absent)",
			[]shadow.Group{
				{Name: "wheel", GID: 10, Members: []string{}},
				{Name: "team", GID: 2000, Members: []string{"alice", "bob", "carol"}},
			},
		},
		"single system group without include system": {
			PruneUnmanagedGroupMemberships{Group: "wheel"},
			"",
			[]shadow.Group{
				{Name: "wheel", GID: 10, Members: []string{"alice", "bob"}},
				{Name: "team", GID: 2000, Members: []string{"alice", "bob", "carol"}},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			wire := identityWire(t, GroupMembershipIdentity("alice", "team"))
			fake := &fakeShadow{groups: cloneGroups(groups)}
			cas.node.shadow = fake

			needs, d, err := cas.node.NeedsExecution(wire)
			require.NoError(t, err)
			if cas.wantDiff == "" {
				assert.False(t, needs)
			} else {
				assert.True(t, needs)
				assert.Equal(t, cas.wantDiff, d.String())
			}

			_, err = cas.node.Execute(wire)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, fake.groups)
		})
	}
}

func TestPruneUnmanagedGroupMemberships_DependsOn(t *testing.T) {
	t.Parallel()

	deps, err := (&PruneUnmanagedGroupMemberships{}).DependsOn()
	require.NoError(t, err)
	assert.Empty(t, deps)

	deps, err = (&PruneUnmanagedGroupMemberships{Group: "g"}).DependsOn()
	require.NoError(t, err)
	assert.Equal(t, []tensile.Identity{GroupIdentity("g")}, deps)
}

// cloneGroups deep copies groups including their member lists.
func cloneGroups(groups []shadow.Group) []shadow.Group {
	cloned := make([]shadow.Group, len(groups))
	for i, group := range groups {
		cloned[i] = group
		cloned[i].Members = slices.Clone(group.Members)
	}
	return cloned
}
