package graph

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraph_Add(t *testing.T) {
	t.Parallel()

	var g Graph[string, int]
	require.NoError(t, g.Add("a", 1))
	require.NoError(t, g.Add("b", 2))

	require.ErrorContains(t, g.Add("a", 3), "already exists")

	got, ok := g.Get("a")
	require.True(t, ok)
	assert.Equal(t, 1, got, "duplicate add must not overwrite")
	assert.Equal(t, 2, g.Len())

	_, ok = g.Get("missing")
	assert.False(t, ok)
}

func TestGraph_AddEdge(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		from, to string
		wantErr  string
	}{
		"valid":        {from: "a", to: "b"},
		"self edge":    {from: "a", to: "a", wantErr: "self edge"},
		"unknown from": {from: "x", to: "b", wantErr: "has not been added"},
		"unknown to":   {from: "a", to: "x", wantErr: "has not been added"},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var g Graph[string, int]
			require.NoError(t, g.Add("a", 1))
			require.NoError(t, g.Add("b", 2))

			err := g.AddEdge(cas.from, cas.to)
			if cas.wantErr != "" {
				require.ErrorContains(t, err, cas.wantErr)
				assert.Empty(t, g.Edges(), "rejected edge must not be stored")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []Edge[string]{{From: cas.from, To: cas.to}}, g.Edges())
		})
	}
}

func TestGraph_Nodes(t *testing.T) {
	t.Parallel()

	var g Graph[string, int]
	require.NoError(t, g.Add("c", 3))
	require.NoError(t, g.Add("a", 1))
	require.NoError(t, g.Add("b", 2))

	keys := slices.Collect(maps.Keys(maps.Collect(g.Nodes())))
	assert.ElementsMatch(t, []string{"a", "b", "c"}, keys)

	var ordered []string
	for key := range g.Nodes() {
		ordered = append(ordered, key)
	}
	assert.Equal(t, []string{"c", "a", "b"}, ordered, "iteration must follow insertion order")
}
