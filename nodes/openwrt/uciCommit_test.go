package openwrt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUCICommit_NeedsExecution(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want     bool
		wantErr  bool
		config   string
		out      string
		err      error
		wantArgs []string
	}{
		"staged changes": {
			want:     true,
			config:   "dhcp",
			out:      "dhcp.cfg02host=''\n",
			wantArgs: []string{"changes", "dhcp"},
		},
		"nothing staged": {
			config:   "dhcp",
			wantArgs: []string{"changes", "dhcp"},
		},
		"global commit": {
			want:     true,
			out:      "network.lan.dns='1.1.1.1'\n",
			wantArgs: []string{"changes"},
		},
		"run failure": {
			wantErr: true,
			config:  "dhcp",
			out:     "boom",
			err:     errRun,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var gotArgs []string
			commit := &UCICommit{
				Config: cas.config,
				run: func(_ context.Context, args ...string) ([]byte, error) {
					gotArgs = args
					return []byte(cas.out), cas.err
				},
			}

			got, _, err := commit.NeedsExecution(testWire(t))
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
			assert.Equal(t, cas.wantArgs, gotArgs, "config must scope the changes query")
		})
	}
}
