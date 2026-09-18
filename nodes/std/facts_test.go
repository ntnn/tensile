package std

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacts_NeedsExecution(t *testing.T) {
	t.Parallel()

	needs, err := (&Facts{}).NeedsExecution(testWire(t))
	require.NoError(t, err)
	assert.False(t, needs, "facts must never execute")
}

func TestFacts_Identity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, FactsIdentity(), (&Facts{}).Identity())
	assert.Equal(t, "facts", FactsIdentity().Kind())
}

func TestParseOSReleaseID(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		content string
		want    string
	}{
		"plain":         {"ID=arch\n", "arch"},
		"double quoted": {"ID=\"opensuse-tumbleweed\"\n", "opensuse-tumbleweed"},
		"single quoted": {"ID='debian'\n", "debian"},
		"with siblings": {"NAME=\"Arch Linux\"\nID=arch\nID_LIKE=archlinux\n", "arch"},
		"missing":       {"NAME=\"Arch Linux\"\n", ""},
		"empty":         {"", ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.want, parseOSReleaseID(cas.content))
		})
	}
}
