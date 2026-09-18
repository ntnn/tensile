package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Get(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		seed    any
		want    int
		wantErr string
	}{
		"missing key":   {nil, 0, "no value for key"},
		"type mismatch": {"str", 0, "not int"},
		"typed value":   {42, 42, ""},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			b := NewDefaultBackend[string]()
			if cas.seed != nil {
				require.NoError(t, b.Store("key", cas.seed))
			}

			got, err := New(b).Get[int]("key")
			if cas.wantErr != "" {
				require.ErrorContains(t, err, cas.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestStore_GetWithoutBackend(t *testing.T) {
	t.Parallel()

	_, err := New[string](nil).Get[int]("key")
	require.ErrorContains(t, err, "no backend")
}

func TestDefaultBackend_Store(t *testing.T) {
	t.Parallel()

	b := NewDefaultBackend[string]()
	require.NoError(t, b.Store("key", 1))
	require.ErrorContains(t, b.Store("key", 2), "already has a value")
}
