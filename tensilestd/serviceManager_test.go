package tensilestd

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handlingFake() *fakeServiceManager {
	return &fakeServiceManager{
		handles: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
}

func TestServiceManagerRegistry_register(t *testing.T) {
	t.Parallel()

	reg := &serviceManagerRegistry{}
	reg.register("one", &fakeServiceManager{})

	assert.Panics(t, func() {
		reg.register("one", &fakeServiceManager{})
	}, "duplicate registration must panic")
}

func TestServiceManagerRegistry_byName(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		name    string
	}{
		"registered name resolves": {false, "one"},
		"unknown name errors":      {true, "unknown"},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			mgr := &fakeServiceManager{}
			reg := &serviceManagerRegistry{}
			reg.register("one", mgr)

			got, err := reg.byName(cas.name)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Same(t, mgr, got)
		})
	}
}

func TestServiceManagerRegistry_detectLaterRegistrationWins(t *testing.T) {
	t.Parallel()

	first := handlingFake()
	second := handlingFake()
	reg := &serviceManagerRegistry{}
	reg.register("first", first)
	reg.register("second", second)

	got, err := reg.detect(t.Context(), "svc")
	require.NoError(t, err)
	assert.Same(t, second, got)
}

func TestServiceManagerRegistry_detectSkipsNonHandling(t *testing.T) {
	t.Parallel()

	first := handlingFake()
	reg := &serviceManagerRegistry{}
	reg.register("first", first)
	reg.register("second", &fakeServiceManager{})

	got, err := reg.detect(t.Context(), "svc")
	require.NoError(t, err)
	assert.Same(t, first, got)
}

func TestServiceManagerRegistry_detectNoneHandling(t *testing.T) {
	t.Parallel()

	reg := &serviceManagerRegistry{}
	reg.register("first", &fakeServiceManager{})

	_, err := reg.detect(t.Context(), "svc")
	assert.Error(t, err)
}

func TestServiceManagerRegistry_detectHandlesError(t *testing.T) {
	t.Parallel()

	reg := &serviceManagerRegistry{}
	reg.register("first", &fakeServiceManager{
		handles: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("boom")
		},
	})

	_, err := reg.detect(t.Context(), "svc")
	assert.Error(t, err)
}
