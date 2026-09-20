//go:build linux

package std

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func procdFake(run func(ctx context.Context, name, verb string) (int, []byte, error)) *Procd {
	return &Procd{
		run:       run,
		available: func() bool { return true },
	}
}

func TestProcd_Handles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	//nolint:gosec // executable bit is the subject under test
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc"), []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plain"), []byte(""), 0o600))

	cases := map[string]struct {
		want      bool
		available bool
		name      string
	}{
		"executable script": {want: true, available: true, name: "svc"},
		"not executable":    {available: true, name: "plain"},
		"missing script":    {available: true, name: "nosuch"},
		"not available":     {name: "svc"},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			procd := &Procd{
				available: func() bool { return cas.available },
				dir:       dir,
			}

			got, err := procd.Handles(t.Context(), cas.name)
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestProcd_Status(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		want        ServiceStatus
		enabledCode int
		runningCode int
	}{
		"enabled running":  {want: ServiceStatus{Enabled: true, Active: true}},
		"enabled stopped":  {want: ServiceStatus{Enabled: true}, runningCode: 1},
		"disabled running": {want: ServiceStatus{Active: true}, enabledCode: 1},
		"disabled stopped": {enabledCode: 1, runningCode: 1},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			procd := procdFake(func(_ context.Context, _, verb string) (int, []byte, error) {
				if verb == "enabled" {
					return cas.enabledCode, nil, nil
				}
				return cas.runningCode, nil, nil
			})

			got, err := procd.Status(t.Context(), "svc")
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestProcd_Status_runError(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")
	procd := procdFake(func(_ context.Context, _, _ string) (int, []byte, error) {
		return -1, nil, errRun
	})

	_, err := procd.Status(t.Context(), "svc")
	require.ErrorIs(t, err, errRun)
}

func TestProcd_Apply(t *testing.T) {
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
			current:  ServiceStatus{Active: true},
			desired:  ServiceStatus{Enabled: true, Active: true},
			wantRuns: []string{"enable"},
		},
		"disable": {
			current:  ServiceStatus{Enabled: true, Active: true},
			desired:  ServiceStatus{Active: true},
			wantRuns: []string{"disable"},
		},
		"start": {
			current:  ServiceStatus{Enabled: true},
			desired:  ServiceStatus{Enabled: true, Active: true},
			wantRuns: []string{"start"},
		},
		"stop": {
			current:  ServiceStatus{Enabled: true, Active: true},
			desired:  ServiceStatus{Enabled: true},
			wantRuns: []string{"stop"},
		},
		"enable and stop": {
			current:  ServiceStatus{Active: true},
			desired:  ServiceStatus{Enabled: true},
			wantRuns: []string{"enable", "stop"},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var runs []string
			procd := procdFake(func(_ context.Context, _, verb string) (int, []byte, error) {
				switch verb {
				case "enabled":
					if cas.current.Enabled {
						return 0, nil, nil
					}
					return 1, nil, nil
				case "running":
					if cas.current.Active {
						return 0, nil, nil
					}
					return 1, nil, nil
				default:
					runs = append(runs, verb)
					return 0, nil, nil
				}
			})

			require.NoError(t, procd.Apply(t.Context(), "svc", cas.desired))
			assert.Equal(t, cas.wantRuns, runs)
		})
	}
}

func TestProcd_Apply_transitionFailure(t *testing.T) {
	t.Parallel()

	procd := procdFake(func(_ context.Context, _, verb string) (int, []byte, error) {
		switch verb {
		case "enabled", "running":
			return 1, nil, nil
		default:
			return 1, []byte("boot script missing\n"), nil
		}
	})

	err := procd.Apply(t.Context(), "svc", ServiceStatus{Enabled: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boot script missing", "script output must surface in the error")
}
