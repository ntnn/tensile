package std

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePackageManager struct {
	handles   func(ctx context.Context, name string) (bool, error)
	installed func(ctx context.Context, name string) (bool, error)
	install   func(ctx context.Context, name string) error
	remove    func(ctx context.Context, name string) error
}

func (f *fakePackageManager) Handles(ctx context.Context, name string) (bool, error) {
	if f.handles == nil {
		return false, nil
	}
	return f.handles(ctx, name)
}

func (f *fakePackageManager) Update(_ context.Context) error {
	return nil
}

func (f *fakePackageManager) Installed(ctx context.Context, name string) (bool, error) {
	return f.installed(ctx, name)
}

func (f *fakePackageManager) Install(ctx context.Context, name string) error {
	return f.install(ctx, name)
}

func (f *fakePackageManager) Remove(ctx context.Context, name string) error {
	return f.remove(ctx, name)
}

func TestPackage_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		pkg     Package
	}{
		"name only":     {false, Package{Name: "pkg"}},
		"present":       {false, Package{Name: "pkg", State: PackagePresent}},
		"absent":        {false, Package{Name: "pkg", State: PackageAbsent}},
		"missing name":  {true, Package{State: PackagePresent}},
		"unknown state": {true, Package{Name: "pkg", State: "sideways"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			err := cas.pkg.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPackage_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected  bool
		state     PackageState
		installed bool
	}{
		"present not installed":     {true, PackagePresent, false},
		"present installed":         {false, PackagePresent, true},
		"absent installed":          {true, PackageAbsent, true},
		"absent not installed":      {false, PackageAbsent, false},
		"empty state not installed": {true, "", false},
		"empty state installed":     {false, "", true},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			name := "fake-needs-" + title
			RegisterPackageManager(name, &fakePackageManager{
				installed: func(_ context.Context, _ string) (bool, error) {
					return cas.installed, nil
				},
			})

			p := &Package{
				Name:    "pkg",
				State:   cas.state,
				Manager: name,
			}

			needs, _, err := p.NeedsExecution(testWire(t))
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}

func TestPackage_Execute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		state       PackageState
		wantInstall bool
	}{
		"present installs": {PackagePresent, true},
		"absent removes":   {PackageAbsent, false},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var installed, removed string
			name := "fake-execute-" + title
			RegisterPackageManager(name, &fakePackageManager{
				install: func(_ context.Context, name string) error {
					installed = name
					return nil
				},
				remove: func(_ context.Context, name string) error {
					removed = name
					return nil
				},
			})

			p := &Package{
				Name:    "pkg",
				State:   cas.state,
				Manager: name,
			}
			_, err := p.Execute(testWire(t))
			require.NoError(t, err)

			if cas.wantInstall {
				assert.Equal(t, "pkg", installed)
				assert.Empty(t, removed)
				return
			}
			assert.Equal(t, "pkg", removed)
			assert.Empty(t, installed)
		})
	}
}

func TestPackage_NeedsExecutionUnknownManager(t *testing.T) {
	t.Parallel()

	p := &Package{
		Name:    "pkg",
		Manager: "does-not-exist",
	}

	_, _, err := p.NeedsExecution(testWire(t))
	assert.Error(t, err, "an unknown manager should be an error")
}
