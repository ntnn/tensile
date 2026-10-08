package std

import (
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/ntnn/tensile/pkg/shadow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupMembership_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr    bool
		membership GroupMembership
	}{
		"valid":              {false, GroupMembership{User: "u", Group: "g"}},
		"valid absent":       {false, GroupMembership{User: "u", Group: "g", State: tensile.Absent}},
		"missing user":       {true, GroupMembership{Group: "g"}},
		"missing group":      {true, GroupMembership{User: "u"}},
		"read only":          {true, GroupMembership{User: "u", Group: "g", State: tensile.ReadOnly}},
		"unknown state":      {true, GroupMembership{User: "u", Group: "g", State: "bogus"}},
		"missing everything": {true, GroupMembership{}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			cas.membership.shadow = &fakeShadow{}
			err := cas.membership.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGroupMembership_NeedsExecution(t *testing.T) {
	t.Parallel()

	member := []shadow.Group{{Name: "g", Members: []string{"other", "u"}}}
	notMember := []shadow.Group{{Name: "g", Members: []string{"other"}}}

	cases := map[string]struct {
		expected bool
		state    tensile.State
		groups   []shadow.Group
		wantDiff string
	}{
		"add":               {true, tensile.Present, notMember, "state: absent -> present"},
		"add missing group": {true, tensile.Present, nil, "state: absent -> present"},
		"already member":    {false, tensile.Present, member, ""},
		"remove":            {true, tensile.Absent, member, "state: present -> absent"},
		"already removed":   {false, tensile.Absent, notMember, ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			membership := &GroupMembership{
				User:   "u",
				Group:  "g",
				State:  cas.state,
				shadow: &fakeShadow{groups: cas.groups},
			}

			needs, d, err := membership.NeedsExecution(nil)
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

func TestGroupMembership_Execute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		state    tensile.State
		groups   []shadow.Group
		expected []string
	}{
		"add":    {tensile.Present, []shadow.Group{{Name: "g", Members: []string{"other"}}}, []string{"other", "u"}},
		"remove": {tensile.Absent, []shadow.Group{{Name: "g", Members: []string{"other", "u"}}}, []string{"other"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			fake := &fakeShadow{groups: slices.Clone(cas.groups)}
			membership := &GroupMembership{
				User:   "u",
				Group:  "g",
				State:  cas.state,
				shadow: fake,
			}

			_, err := membership.Execute(&tensile.DefaultWire{Ctx: t.Context()})
			require.NoError(t, err)
			assert.Equal(t, cas.expected, fake.groups[0].Members)
		})
	}
}

func TestGroupMembership_conflictsWithUserGroups(t *testing.T) {
	t.Parallel()

	q := queue.New()
	q.Add(
		&User{Name: "u", Groups: []string{"g"}},
		&GroupMembership{User: "u", Group: "g"},
	)
	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed",
		"a membership declared in User.Groups and a GroupMembership must conflict")

	q = queue.New()
	q.Add(
		&User{Name: "u", Groups: []string{"other"}},
		&GroupMembership{User: "u", Group: "g"},
	)
	_, err = q.Build()
	require.NoError(t, err, "a GroupMembership for a group outside User.Groups must not conflict")
}
