package std

import (
	"context"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeServiceManager struct {
	handles func(ctx context.Context, name string) (bool, error)
	status  func(ctx context.Context, name string) (ServiceStatus, error)
	apply   func(ctx context.Context, name string, desired ServiceStatus) error
}

func (f *fakeServiceManager) Handles(ctx context.Context, name string) (bool, error) {
	if f.handles == nil {
		return false, nil
	}
	return f.handles(ctx, name)
}

func (f *fakeServiceManager) Status(ctx context.Context, name string) (ServiceStatus, error) {
	return f.status(ctx, name)
}

func (f *fakeServiceManager) Apply(ctx context.Context, name string, desired ServiceStatus) error {
	return f.apply(ctx, name, desired)
}

func testWire(t *testing.T) tensile.Wire {
	t.Helper()
	return &tensile.DefaultWire{Ctx: t.Context()}
}

func TestService_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		service Service
	}{
		"name and enabled":  {false, Service{Name: "svc", Enabled: new(true)}},
		"name and running":  {false, Service{Name: "svc", Running: new(true)}},
		"missing name":      {true, Service{Enabled: new(true)}},
		"no state to apply": {true, Service{Name: "svc"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			err := cas.service.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestService_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		enabled  *bool
		running  *bool
		status   ServiceStatus
	}{
		"enabled mismatch":        {true, new(true), nil, ServiceStatus{Enabled: false, Active: true}},
		"running mismatch":        {true, nil, new(true), ServiceStatus{Enabled: true, Active: false}},
		"all matching":            {false, new(true), new(true), ServiceStatus{Enabled: true, Active: true}},
		"unmanaged drift ignored": {false, new(true), nil, ServiceStatus{Enabled: true, Active: false}},
		"both mismatch":           {true, new(true), new(true), ServiceStatus{Enabled: false, Active: false}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			name := "fake-needs-" + title
			RegisterServiceManager(name, &fakeServiceManager{
				status: func(_ context.Context, _ string) (ServiceStatus, error) {
					return cas.status, nil
				},
			})

			s := &Service{
				Name:    "svc",
				Enabled: cas.enabled,
				Running: cas.running,
				Manager: name,
			}

			needs, err := s.NeedsExecution(testWire(t))
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}

func TestService_Execute(t *testing.T) {
	t.Parallel()

	var applied ServiceStatus
	RegisterServiceManager("fake-execute", &fakeServiceManager{
		status: func(_ context.Context, _ string) (ServiceStatus, error) {
			return ServiceStatus{Enabled: true, Active: false}, nil
		},
		apply: func(_ context.Context, _ string, desired ServiceStatus) error {
			applied = desired
			return nil
		},
	})

	s := &Service{
		Name:    "svc",
		Running: new(true),
		Manager: "fake-execute",
	}
	require.NoError(t, s.Execute(testWire(t)))

	assert.True(t, applied.Active, "managed field must be applied")
	assert.True(t, applied.Enabled, "unmanaged field must keep current state")
}

func TestService_NeedsExecutionUnknownManager(t *testing.T) {
	t.Parallel()

	s := &Service{
		Name:    "svc",
		Enabled: new(true),
		Manager: "does-not-exist",
	}

	_, err := s.NeedsExecution(testWire(t))
	assert.Error(t, err, "an unknown manager should be an error")
}
