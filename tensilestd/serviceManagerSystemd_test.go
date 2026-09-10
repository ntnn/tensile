//go:build linux

package tensilestd

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// showOutput renders systemctl show property output.
func showOutput(loadState, unitFileState, activeState string) []byte {
	return fmt.Appendf(nil, "LoadState=%s\nUnitFileState=%s\nActiveState=%s\n",
		loadState, unitFileState, activeState)
}

func systemdFake(run func(ctx context.Context, args ...string) ([]byte, error)) *Systemd {
	return &Systemd{
		run:       run,
		available: func() bool { return true },
	}
}

func TestSystemd_Handles(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want      bool
		wantErr   bool
		available bool
		loadState string
	}{
		"unit exists":      {true, false, true, "loaded"},
		"unit unknown":     {false, false, true, "not-found"},
		"not available":    {false, false, false, ""},
		"run error errors": {false, true, true, ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var gotArgs []string
			systemd := &Systemd{
				available: func() bool { return cas.available },
				run: func(_ context.Context, args ...string) ([]byte, error) {
					gotArgs = args
					if cas.wantErr {
						return nil, errRun
					}
					return showOutput(cas.loadState, "enabled", "active"), nil
				},
			}

			got, err := systemd.Handles(t.Context(), "svc")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
			if cas.available {
				assert.Equal(t,
					[]string{"show", "--property=LoadState,UnitFileState,ActiveState", "--", "svc"},
					gotArgs)
			}
		})
	}
}

func TestSystemd_Status(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		want          ServiceStatus
		wantErr       bool
		loadState     string
		unitFileState string
		activeState   string
	}{
		"enabled active":         {ServiceStatus{true, true}, false, "loaded", "enabled", "active"},
		"enabled-runtime counts": {ServiceStatus{true, false}, false, "loaded", "enabled-runtime", "inactive"},
		"disabled inactive":      {ServiceStatus{false, false}, false, "loaded", "disabled", "inactive"},
		"static is not enabled":  {ServiceStatus{false, true}, false, "loaded", "static", "active"},
		"masked is not enabled":  {ServiceStatus{false, false}, false, "masked", "masked", "failed"},
		"unknown unit errors":    {ServiceStatus{}, true, "not-found", "", "inactive"},
		"bad-setting errors":     {ServiceStatus{}, true, "bad-setting", "", "inactive"},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			systemd := systemdFake(func(_ context.Context, _ ...string) ([]byte, error) {
				return showOutput(cas.loadState, cas.unitFileState, cas.activeState), nil
			})

			got, err := systemd.Status(t.Context(), "svc")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestSystemd_Status_runError(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")
	systemd := systemdFake(func(_ context.Context, _ ...string) ([]byte, error) {
		return nil, errRun
	})

	_, err := systemd.Status(t.Context(), "svc")
	require.ErrorIs(t, err, errRun)
}

func TestSystemd_Apply(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		current  ServiceStatus
		desired  ServiceStatus
		wantRuns []string
	}{
		"no-op":           {ServiceStatus{true, true}, ServiceStatus{true, true}, nil},
		"enable":          {ServiceStatus{false, true}, ServiceStatus{true, true}, []string{"enable svc"}},
		"disable":         {ServiceStatus{true, true}, ServiceStatus{false, true}, []string{"disable svc"}},
		"start":           {ServiceStatus{true, false}, ServiceStatus{true, true}, []string{"start svc"}},
		"stop":            {ServiceStatus{true, true}, ServiceStatus{true, false}, []string{"stop svc"}},
		"enable and stop": {ServiceStatus{false, true}, ServiceStatus{true, false}, []string{"enable svc", "stop svc"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var runs []string
			systemd := systemdFake(func(_ context.Context, args ...string) ([]byte, error) {
				if args[0] == "show" {
					unitFileState := "disabled"
					if cas.current.Enabled {
						unitFileState = "enabled"
					}
					activeState := "inactive"
					if cas.current.Active {
						activeState = "active"
					}
					return showOutput("loaded", unitFileState, activeState), nil
				}
				runs = append(runs, args[0]+" "+args[len(args)-1])
				return nil, nil
			})

			require.NoError(t, systemd.Apply(t.Context(), "svc", cas.desired))
			assert.Equal(t, cas.wantRuns, runs)
		})
	}
}

func TestSystemd_Apply_transitionFailure(t *testing.T) {
	t.Parallel()

	systemd := systemdFake(func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] == "show" {
			return showOutput("masked", "masked", "inactive"), nil
		}
		return []byte("Unit svc.service is masked.\n"), errors.New("exit status 1")
	})

	err := systemd.Apply(t.Context(), "svc", ServiceStatus{Enabled: true, Active: false})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "masked", "systemctl output must surface in the error")
}
