package std

import (
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/shadow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroup_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		group   Group
	}{
		"valid":         {false, Group{Name: "g"}},
		"valid absent":  {false, Group{Name: "g", State: tensile.Absent}},
		"valid report":  {false, Group{Name: "g", State: tensile.ReadOnly}},
		"missing name":  {true, Group{}},
		"unknown state": {true, Group{Name: "g", State: "bogus"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			cas.group.shadow = &fakeShadow{}
			err := cas.group.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGroup_Conflicts(t *testing.T) {
	t.Parallel()

	conflicts, err := (&Group{Name: "g"}).Conflicts()
	require.NoError(t, err)
	assert.Empty(t, conflicts, "unmanaged GID must not conflict")

	gid := 1000
	conflicts, err = (&Group{Name: "g", GID: &gid}).Conflicts()
	require.NoError(t, err)
	assert.Equal(t, []tensile.Identity{gidIdentity(1000)}, conflicts)
}

func TestGroup_NeedsExecution(t *testing.T) {
	t.Parallel()

	gid := 1000
	otherGID := 2000
	existing := []shadow.Group{{Name: "g", GID: 1000}}

	cases := map[string]struct {
		expected bool
		group    Group
		groups   []shadow.Group
		wantDiff string
	}{
		"create":                 {true, Group{Name: "g"}, nil, "state: absent -> present"},
		"create with gid":        {true, Group{Name: "g", GID: &gid}, nil, "state: absent -> present\ngid: (absent) -> 1000"},
		"present":                {false, Group{Name: "g"}, existing, ""},
		"present with gid":       {false, Group{Name: "g", GID: &gid}, existing, ""},
		"gid change":             {true, Group{Name: "g", GID: &otherGID}, existing, "gid: 1000 -> 2000"},
		"delete":                 {true, Group{Name: "g", State: tensile.Absent}, existing, "state: present -> absent"},
		"already absent":         {false, Group{Name: "g", State: tensile.Absent}, nil, ""},
		"read only missing":      {false, Group{Name: "g", State: tensile.ReadOnly}, nil, ""},
		"read only gid mismatch": {false, Group{Name: "g", GID: &otherGID, State: tensile.ReadOnly}, existing, ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			cas.group.shadow = &fakeShadow{groups: cas.groups}

			needs, d, err := cas.group.NeedsExecution(nil)
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

func TestGroup_Execute(t *testing.T) {
	t.Parallel()

	gid := 1000
	otherGID := 2000
	existing := []shadow.Group{{Name: "g", GID: 1000}}

	cases := map[string]struct {
		group    Group
		groups   []shadow.Group
		expected []shadow.Group
	}{
		"create":    {Group{Name: "g", GID: &gid}, nil, existing},
		"change":    {Group{Name: "g", GID: &otherGID}, existing, []shadow.Group{{Name: "g", GID: 2000}}},
		"delete":    {Group{Name: "g", State: tensile.Absent}, existing, []shadow.Group{}},
		"read only": {Group{Name: "g", GID: &otherGID, State: tensile.ReadOnly}, existing, existing},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			fake := &fakeShadow{groups: slices.Clone(cas.groups)}
			cas.group.shadow = fake

			_, err := cas.group.Execute(&tensile.DefaultWire{Ctx: t.Context()})
			require.NoError(t, err)
			assert.Equal(t, cas.expected, fake.groups)
		})
	}
}

func TestGroup_Report(t *testing.T) {
	t.Parallel()

	fake := &fakeShadow{groups: []shadow.Group{{Name: "g", GID: 1000, Members: []string{"u"}}}}

	output, err := (&Group{Name: "g", State: tensile.ReadOnly, shadow: fake}).Report(nil)
	require.NoError(t, err)
	assert.Equal(t, GroupData{
		Name:    "g",
		GID:     1000,
		Members: []string{"u"},
		Present: true,
	}, output)

	output, err = (&Group{Name: "missing", State: tensile.ReadOnly, shadow: fake}).Report(nil)
	require.NoError(t, err)
	assert.Equal(t, GroupData{Name: "missing"}, output, "missing group must report not present")
}
