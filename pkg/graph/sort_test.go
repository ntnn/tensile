package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraph_TopoSort(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		nodes       []string
		edges       []Edge[string]
		want        []string
		wantParents map[string][]string
		wantCycle   []string
	}{
		"empty": {
			want: []string{},
		},
		"linear chain": {
			nodes: []string{"c", "b", "a"},
			edges: []Edge[string]{
				{From: "a", To: "b"},
				{From: "b", To: "c"},
			},
			want: []string{"a", "b", "c"},
			wantParents: map[string][]string{
				"b": {"a"},
				"c": {"b"},
			},
		},
		"diamond": {
			nodes: []string{"d", "c", "b", "a"},
			edges: []Edge[string]{
				{From: "a", To: "b"},
				{From: "a", To: "c"},
				{From: "b", To: "d"},
				{From: "c", To: "d"},
			},
			want: []string{"a", "b", "c", "d"},
		},
		"disjoint insertion order": {
			nodes: []string{"c", "a", "b"},
			want:  []string{"c", "a", "b"},
		},
		"duplicate edges": {
			nodes: []string{"b", "a"},
			edges: []Edge[string]{
				{From: "a", To: "b"},
				{From: "a", To: "b"},
			},
			want: []string{"a", "b"},
		},
		"simple cycle": {
			nodes: []string{"a", "b"},
			edges: []Edge[string]{
				{From: "a", To: "b"},
				{From: "b", To: "a"},
			},
			wantCycle: []string{"a", "b"},
		},
		"cycle with downstream node": {
			nodes: []string{"a", "b", "c"},
			edges: []Edge[string]{
				{From: "a", To: "b"},
				{From: "b", To: "a"},
				{From: "a", To: "c"},
			},
			wantCycle: []string{"a", "b"},
		},
		"cycle across nodes": {
			nodes: []string{"a", "b", "c"},
			edges: []Edge[string]{
				{From: "a", To: "b"},
				{From: "b", To: "c"},
				{From: "c", To: "a"},
			},
			wantCycle: []string{"a", "b", "c"},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var g Graph[string, int]
			for _, key := range cas.nodes {
				require.NoError(t, g.Add(key, 0))
			}
			for _, edge := range cas.edges {
				require.NoError(t, g.AddEdge(edge.From, edge.To))
			}

			sorted, parents, err := g.TopoSort()
			if cas.wantCycle != nil {
				var cycleErr *CycleError[string]
				require.ErrorAs(t, err, &cycleErr)
				assert.ElementsMatch(t, cas.wantCycle, cycleErr.Cycle,
					"cycle must list exactly the cycle members, not downstream nodes")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, sorted)
			if cas.wantParents != nil {
				assert.Equal(t, cas.wantParents, parents)
			}
		})
	}
}

func TestCycleError_Error(t *testing.T) {
	t.Parallel()

	err := error(&CycleError[string]{Cycle: []string{"a", "b"}})
	assert.Equal(t, "cycle detected: a -> b", err.Error())
}
