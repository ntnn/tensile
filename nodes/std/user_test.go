package std

import (
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/shadow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// membershipWire returns a wire whose topology claims the membership of user in each group by claimer.
func membershipWire(t *testing.T, claimer tensile.Identity, user string, groups ...string) tensile.Wire {
	t.Helper()
	claims := mapTopology{}
	for _, group := range groups {
		claims[GroupMembershipIdentity(user, group)] = claimer
	}
	return &tensile.DefaultWire{
		Ctx:  t.Context(),
		Topo: claims,
	}
}

func TestUser_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		user    User
	}{
		"valid":         {false, User{Name: "u"}},
		"valid absent":  {false, User{Name: "u", State: tensile.Absent}},
		"valid report":  {false, User{Name: "u", State: tensile.ReadOnly}},
		"missing name":  {true, User{}},
		"unknown state": {true, User{Name: "u", State: "bogus"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			cas.user.shadow = &fakeShadow{}
			err := cas.user.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUser_Edges(t *testing.T) {
	t.Parallel()

	uid := 1000
	user := &User{
		Name:   "u",
		UID:    &uid,
		Group:  "primary",
		Groups: []string{"a", "b"},
		Home:   "/home/u",
	}

	conflicts, err := user.Conflicts()
	require.NoError(t, err)
	assert.Equal(t, []tensile.Identity{
		uidIdentity(1000),
		GroupMembershipIdentity("u", "a"),
		GroupMembershipIdentity("u", "b"),
	}, conflicts)

	deps, err := user.DependsOn()
	require.NoError(t, err)
	assert.Equal(t, []tensile.Identity{
		GroupIdentity("primary"),
		GroupIdentity("a"),
		GroupIdentity("b"),
	}, deps)

	reqs, err := user.RequiredBy()
	require.NoError(t, err)
	assert.Equal(t, []tensile.Identity{FileIdentity("/home/u")}, reqs, "nodes managing the home must run after the user")

	minimal := &User{Name: "u"}
	conflicts, err = minimal.Conflicts()
	require.NoError(t, err)
	assert.Empty(t, conflicts)
	deps, err = minimal.DependsOn()
	require.NoError(t, err)
	assert.Empty(t, deps)
	reqs, err = minimal.RequiredBy()
	require.NoError(t, err)
	assert.Empty(t, reqs)
}

func TestUser_NeedsExecution(t *testing.T) {
	t.Parallel()

	uid := 1000
	otherUID := 2000
	existing := []shadow.User{{
		Name:   "u",
		UID:    1000,
		Group:  "primary",
		Groups: []string{"a", "claimed"},
		Home:   "/home/u",
		Shell:  "/bin/sh",
	}}

	cases := map[string]struct {
		expected bool
		user     User
		users    []shadow.User
		wantDiff string
	}{
		"create minimal": {true, User{Name: "u"}, nil, "state: absent -> present"},
		"create full": {
			true,
			User{Name: "u", UID: &uid, Group: "primary", Groups: []string{"b", "a"}, Home: "/home/u", Shell: "/bin/sh"},
			nil,
			"state: absent -> present\nuid: (absent) -> 1000\ngroup: (absent) -> primary\n" +
				"groups: (absent) -> a,b\nhome: (absent) -> /home/u\nshell: (absent) -> /bin/sh",
		},
		"unmanaged fields": {false, User{Name: "u"}, existing, ""},
		"unchanged": {
			false,
			User{Name: "u", UID: &uid, Group: "primary", Groups: []string{"claimed", "a"}, Home: "/home/u", Shell: "/bin/sh"},
			existing,
			"",
		},
		"modify": {
			true,
			User{Name: "u", UID: &otherUID, Group: "other", Home: "/srv/u", Shell: "/bin/bash"},
			existing,
			"uid: 1000 -> 2000\ngroup: primary -> other\nhome: /home/u -> /srv/u\nshell: /bin/sh -> /bin/bash",
		},
		"groups keep claimed membership": {
			true,
			User{Name: "u", Groups: []string{}},
			existing,
			"groups: a,claimed -> claimed",
		},
		"groups add": {
			true,
			User{Name: "u", Groups: []string{"a", "b"}},
			existing,
			"groups: a,claimed -> a,b,claimed",
		},
		"delete":         {true, User{Name: "u", State: tensile.Absent}, existing, "state: present -> absent"},
		"already absent": {false, User{Name: "u", State: tensile.Absent}, nil, ""},
		"read only":      {false, User{Name: "u", UID: &otherUID, State: tensile.ReadOnly}, existing, ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			cas.user.shadow = &fakeShadow{users: cas.users}

			wire := membershipWire(t, NoopIdentity("membership"), "u", "claimed")
			needs, d, err := cas.user.NeedsExecution(wire)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
			if cas.wantDiff == "" {
				assert.Nil(t, d)
				return
			}
			require.NotNil(t, d)
			assert.Equal(t, cas.wantDiff, d.String())
		})
	}
}

func TestUser_NeedsExecutionOwnClaims(t *testing.T) {
	t.Parallel()

	user := &User{
		Name:   "u",
		Groups: []string{"a"},
		shadow: &fakeShadow{users: []shadow.User{{Name: "u", Groups: []string{"a", "b"}}}},
	}

	// memberships declared in Groups are claimed by the user itself and must not be kept as foreign claims
	wire := membershipWire(t, user.Identity(), "u", "a", "b")
	needs, d, err := user.NeedsExecution(wire)
	require.NoError(t, err)
	assert.True(t, needs)
	require.NotNil(t, d)
	assert.Equal(t, "groups: a,b -> a", d.String())
}

func TestUser_NeedsExecutionNoTopology(t *testing.T) {
	t.Parallel()

	user := &User{
		Name:   "u",
		Groups: []string{"a"},
		shadow: &fakeShadow{users: []shadow.User{{Name: "u"}}},
	}
	_, _, err := user.NeedsExecution(&tensile.DefaultWire{Ctx: t.Context()})
	require.ErrorIs(t, err, tensile.ErrNoTopology, "managed groups of an existing user need the topology")

	user.Groups = nil
	_, _, err = user.NeedsExecution(&tensile.DefaultWire{Ctx: t.Context()})
	require.NoError(t, err, "unmanaged groups must not need the topology")
}

func TestUser_Execute(t *testing.T) {
	t.Parallel()

	uid := 1000
	existing := []shadow.User{{Name: "u", UID: 1000, Groups: []string{"a", "claimed"}}}

	cases := map[string]struct {
		user         User
		users        []shadow.User
		expected     []shadow.User
		removedHomes []string
	}{
		"create": {
			User{Name: "u", UID: &uid, Group: "primary", Groups: []string{"a"}, Home: "/home/u", Shell: "/bin/sh"},
			nil,
			[]shadow.User{{Name: "u", UID: 1000, Group: "primary", Groups: []string{"a"}, Home: "/home/u", Shell: "/bin/sh"}},
			nil,
		},
		"groups keep claimed membership": {
			User{Name: "u", Groups: []string{"b"}},
			existing,
			[]shadow.User{{Name: "u", UID: 1000, Groups: []string{"b", "claimed"}}},
			nil,
		},
		"delete removes home": {
			User{Name: "u", State: tensile.Absent},
			existing,
			[]shadow.User{},
			[]string{"u"},
		},
		"read only": {
			User{Name: "u", Shell: "/bin/bash", State: tensile.ReadOnly},
			existing,
			existing,
			nil,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			fake := &fakeShadow{users: slices.Clone(cas.users)}
			cas.user.shadow = fake

			_, err := cas.user.Execute(membershipWire(t, NoopIdentity("membership"), "u", "claimed"))
			require.NoError(t, err)
			assert.Equal(t, cas.expected, fake.users)
			assert.Equal(t, cas.removedHomes, fake.removedHomes)
		})
	}
}

func TestUser_Report(t *testing.T) {
	t.Parallel()

	fake := &fakeShadow{users: []shadow.User{{Name: "u", UID: 1000, Group: "g", Groups: []string{"a"}}}}

	output, err := (&User{Name: "u", State: tensile.ReadOnly, shadow: fake}).Report(nil)
	require.NoError(t, err)
	assert.Equal(t, UserData{
		Name:    "u",
		UID:     1000,
		Group:   "g",
		Groups:  []string{"a"},
		Present: true,
	}, output)

	output, err = (&User{Name: "missing", State: tensile.ReadOnly, shadow: fake}).Report(nil)
	require.NoError(t, err)
	assert.Equal(t, UserData{Name: "missing"}, output, "missing user must report not present")
}
