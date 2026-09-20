package std

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errApkExit stands in for a non-zero apk exit.
var errApkExit = &exec.ExitError{}

func apkFake(run func(ctx context.Context, args ...string) ([]byte, error)) *Apk {
	return &Apk{
		run:       run,
		available: func() bool { return true },
	}
}

func TestApk_Handles(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want      bool
		wantErr   bool
		available bool
		err       error
	}{
		"known":             {want: true, available: true},
		"unknown":           {available: true, err: errApkExit},
		"apk not available": {},
		"run failure":       {wantErr: true, available: true, err: errRun},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			apk := &Apk{
				available: func() bool { return cas.available },
				run: func(_ context.Context, _ ...string) ([]byte, error) {
					return nil, cas.err
				},
			}

			got, err := apk.Handles(t.Context(), "pkg")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestApk_Installed(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want    bool
		wantErr bool
		err     error
	}{
		"installed":     {want: true},
		"not installed": {err: errApkExit},
		"run failure":   {wantErr: true, err: errRun},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var gotArgs []string
			apk := apkFake(func(_ context.Context, args ...string) ([]byte, error) {
				gotArgs = args
				return nil, cas.err
			})

			got, err := apk.Installed(t.Context(), "pkg")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
			assert.Equal(t, []string{"info", "-e", "pkg"}, gotArgs)
		})
	}
}
