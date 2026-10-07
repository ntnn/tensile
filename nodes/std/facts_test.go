package std

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacts_NeedsExecution(t *testing.T) {
	t.Parallel()

	needs, _, err := (&Facts{}).NeedsExecution(testWire(t))
	require.NoError(t, err)
	assert.False(t, needs, "facts must never execute")
}

func TestFacts_Identity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, FactsIdentity(), (&Facts{}).Identity())
	assert.Equal(t, "facts", FactsIdentity().Kind())
}

func TestParseOSRelease(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		content string
		want    OSRelease
	}{
		"plain":         {"ID=arch\n", OSRelease{ID: "arch"}},
		"double quoted": {"ID=\"opensuse-tumbleweed\"\n", OSRelease{ID: "opensuse-tumbleweed"}},
		"single quoted": {"ID='debian'\n", OSRelease{ID: "debian"}},
		"all fields": {
			"NAME=\"Rocky Linux\"\n" +
				"PRETTY_NAME=\"Rocky Linux 9.4 (Blue Onyx)\"\n" +
				"ID=\"rocky\"\n" +
				"ID_LIKE=\"rhel centos fedora\"\n" +
				"VERSION=\"9.4 (Blue Onyx)\"\n" +
				"VERSION_ID=\"9.4\"\n" +
				"BUILD_ID=rolling\n",
			OSRelease{
				Name:       "Rocky Linux",
				PrettyName: "Rocky Linux 9.4 (Blue Onyx)",
				ID:         "rocky",
				IDLike:     []string{"rhel", "centos", "fedora"},
				Version:    "9.4 (Blue Onyx)",
				VersionID:  "9.4",
				BuildID:    "rolling",
			},
		},
		"unknown keys and comments": {"# comment\nVARIANT_ID=server\nID=arch\n", OSRelease{ID: "arch"}},
		"missing":                   {"VARIANT_ID=server\n", OSRelease{}},
		"empty":                     {"", OSRelease{}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.want, parseOSRelease(cas.content))
		})
	}
}
