package std

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func opkgFake(run func(ctx context.Context, args ...string) ([]byte, error)) *Opkg {
	return &Opkg{
		run:       run,
		available: func() bool { return true },
	}
}

// opkgStatusInstalled renders opkg status output for an installed package.
var opkgStatusInstalled = []byte("Package: pkg\nVersion: 1.0-1\nStatus: install ok installed\n")

// opkgInfoNotInstalled renders opkg info output for a known, not installed package.
var opkgInfoNotInstalled = []byte("Package: pkg\nVersion: 1.0-1\nStatus: unknown ok not-installed\n")

func TestOpkg_Handles(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want      bool
		wantErr   bool
		available bool
		info      []byte
		status    []byte
		err       error
	}{
		"known in lists":     {want: true, available: true, info: opkgInfoNotInstalled},
		"installed only":     {want: true, available: true, status: opkgStatusInstalled},
		"unknown":            {available: true},
		"opkg not available": {},
		"run failure":        {wantErr: true, available: true, err: errRun},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			opkg := &Opkg{
				available: func() bool { return cas.available },
				run: func(_ context.Context, args ...string) ([]byte, error) {
					if args[0] == "info" {
						return cas.info, cas.err
					}
					return cas.status, cas.err
				},
			}

			got, err := opkg.Handles(t.Context(), "pkg")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestOpkg_Installed(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want    bool
		wantErr bool
		out     []byte
		err     error
	}{
		"installed":     {want: true, out: opkgStatusInstalled},
		"not installed": {},
		"known but not installed keeps false": {
			out: opkgInfoNotInstalled,
		},
		"run failure": {wantErr: true, out: []byte("boom"), err: errRun},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			opkg := opkgFake(func(_ context.Context, _ ...string) ([]byte, error) {
				return cas.out, cas.err
			})

			got, err := opkg.Installed(t.Context(), "pkg")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}
