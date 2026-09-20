package openwrt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUCISection_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		section UCISection
		wantErr bool
	}{
		"valid": {
			section: UCISection{Config: "c", Section: "s", Type: "t"},
		},
		"valid absent without type": {
			section: UCISection{Config: "c", Section: "s", State: UCIAbsent},
		},
		"missing config": {
			section: UCISection{Section: "s", Type: "t"},
			wantErr: true,
		},
		"missing section": {
			section: UCISection{Config: "c", Type: "t"},
			wantErr: true,
		},
		"missing type": {
			section: UCISection{Config: "c", Section: "s"},
			wantErr: true,
		},
		"unknown state": {
			section: UCISection{Config: "c", Section: "s", Type: "t", State: "bogus"},
			wantErr: true,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			err := cas.section.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUCISection_NeedsExecution(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want    bool
		wantErr bool
		state   UCIState
		out     string
		err     error
	}{
		"matches": {
			out: "settings\n",
		},
		"type differs": {
			want: true,
			out:  "other\n",
		},
		"missing": {
			want: true,
			out:  "uci: Entry not found",
			err:  errRun,
		},
		"absent but exists": {
			want:  true,
			state: UCIAbsent,
			out:   "settings\n",
		},
		"absent and missing": {
			state: UCIAbsent,
			out:   "uci: Entry not found",
			err:   errRun,
		},
		"run failure": {
			wantErr: true,
			out:     "boom",
			err:     errRun,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			section := &UCISection{
				Config: "t", Section: "main", Type: "settings",
				State: cas.state,
				run: func(_ context.Context, _ ...string) ([]byte, error) {
					return []byte(cas.out), cas.err
				},
			}

			got, err := section.NeedsExecution(testWire(t))
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}
