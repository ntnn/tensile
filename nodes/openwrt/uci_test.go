package openwrt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUCIValues(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		in   string
		want []string
	}{
		"scalar": {
			in:   "'world'",
			want: []string{"world"},
		},
		"list": {
			in:   "'192.168.178.5' '1.1.1.1'",
			want: []string{"192.168.178.5", "1.1.1.1"},
		},
		"space inside quotes": {
			in:   "'managed-config' 'other config'",
			want: []string{"managed-config", "other config"},
		},
		"shell-escaped quote": {
			in:   `'it'\''s'`,
			want: []string{"it's"},
		},
		"empty value": {
			in:   "''",
			want: []string{""},
		},
		"empty input": {
			in:   "",
			want: nil,
		},
		"unquoted fallback": {
			in:   "bare",
			want: []string{"bare"},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.want, parseUCIValues(cas.in))
		})
	}
}
