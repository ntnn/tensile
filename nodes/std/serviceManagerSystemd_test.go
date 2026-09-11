//go:build linux

package std

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
		"unit exists":      {want: true, available: true, loadState: "loaded"},
		"unit unknown":     {available: true, loadState: "not-found"},
		"not available":    {},
		"run error errors": {wantErr: true, available: true},
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
		"enabled active": {
			want:      ServiceStatus{Enabled: true, Active: true},
			loadState: "loaded", unitFileState: "enabled", activeState: "active",
		},
		"enabled-runtime counts": {
			want:      ServiceStatus{Enabled: true},
			loadState: "loaded", unitFileState: "enabled-runtime", activeState: "inactive",
		},
		"disabled inactive": {
			loadState: "loaded", unitFileState: "disabled", activeState: "inactive",
		},
		"static is not enabled": {
			want:      ServiceStatus{Active: true},
			loadState: "loaded", unitFileState: "static", activeState: "active",
		},
		"masked is not enabled": {
			loadState: "masked", unitFileState: "masked", activeState: "failed",
		},
		"unknown unit errors": {
			wantErr:   true,
			loadState: "not-found", activeState: "inactive",
		},
		"bad-setting errors": {
			wantErr:   true,
			loadState: "bad-setting", activeState: "inactive",
		},
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
		"no-op": {
			current: ServiceStatus{Enabled: true, Active: true},
			desired: ServiceStatus{Enabled: true, Active: true},
		},
		"enable": {
			current:  ServiceStatus{Enabled: false, Active: true},
			desired:  ServiceStatus{Enabled: true, Active: true},
			wantRuns: []string{"enable svc"},
		},
		"disable": {
			current:  ServiceStatus{Enabled: true, Active: true},
			desired:  ServiceStatus{Enabled: false, Active: true},
			wantRuns: []string{"disable svc"},
		},
		"start": {
			current:  ServiceStatus{Enabled: true, Active: false},
			desired:  ServiceStatus{Enabled: true, Active: true},
			wantRuns: []string{"start svc"},
		},
		"stop": {
			current:  ServiceStatus{Enabled: true, Active: true},
			desired:  ServiceStatus{Enabled: true, Active: false},
			wantRuns: []string{"stop svc"},
		},
		"enable and stop": {
			current:  ServiceStatus{Enabled: false, Active: true},
			desired:  ServiceStatus{Enabled: true, Active: false},
			wantRuns: []string{"enable svc", "stop svc"},
		},
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
