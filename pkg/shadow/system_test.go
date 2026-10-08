package shadow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSystemID(t *testing.T) {
	t.Parallel()

	low := 500
	high := 2000

	cases := map[string]struct {
		expected bool
		id       int
		idMin    *int
		idMax    *int
	}{
		"root":                {true, 0, nil, nil},
		"below default min":   {true, 999, nil, nil},
		"default min":         {false, 1000, nil, nil},
		"default max":         {false, 60000, nil, nil},
		"above default max":   {true, 60001, nil, nil},
		"nobody":              {true, 65534, nil, nil},
		"custom min includes": {false, 500, &low, nil},
		"custom min excludes": {true, 499, &low, nil},
		"custom max includes": {false, 2000, nil, &high},
		"custom max excludes": {true, 2001, nil, &high},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cas.expected, IsSystemID(cas.id, cas.idMin, cas.idMax))
		})
	}
}
