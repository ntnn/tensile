package graph

import (
	"fmt"
	"slices"
	"strings"
)

// CycleError reports a dependency cycle.
type CycleError[K comparable] struct {
	// Cycle are the members of one detected cycle in edge order.
	Cycle []K
}

func (e *CycleError[K]) Error() string {
	parts := make([]string, len(e.Cycle))
	for i, key := range e.Cycle {
		parts[i] = fmt.Sprintf("%v", key)
	}
	return "cycle detected: " + strings.Join(parts, " -> ")
}

// TopoSort returns the keys ordered so that for every edge From comes before To.
// Returns a [CycleError] if the graph contains a cycle.
func (g *Graph[K, V]) TopoSort() ([]K, error) {
	// map a node to its children
	children := make(map[K][]K, len(g.nodes))
	// number of pending parents of a node
	pendingParents := make(map[K]int, len(g.nodes))
	for _, edge := range g.edges {
		children[edge.From] = append(children[edge.From], edge.To)
		pendingParents[edge.To]++
	}

	// list of all nodes that do not have a parent pending to be sorted
	var ready []K
	for _, key := range g.order {
		if pendingParents[key] == 0 {
			ready = append(ready, key)
		}
	}

	sorted := make([]K, 0, len(g.order))
	for len(ready) > 0 {
		key := ready[0]
		ready = ready[1:]
		sorted = append(sorted, key)
		// the node was sorted, update the children's records
		for _, next := range children[key] {
			pendingParents[next]--
			if pendingParents[next] == 0 {
				ready = append(ready, next)
			}
		}
	}

	// check that no nodes are left
	if len(sorted) != len(g.order) {
		return nil, &CycleError[K]{Cycle: g.cycle(pendingParents)}
	}
	return sorted, nil
}

// cycle walks one dependency cycle among the leftover nodes to try and
// detect the cycle leading to the leftover nodes.
func (g *Graph[K, V]) cycle(pendingParents map[K]int) []K {
	parents := make(map[K][]K)
	for _, edge := range g.edges {
		if pendingParents[edge.From] > 0 && pendingParents[edge.To] > 0 {
			parents[edge.To] = append(parents[edge.To], edge.From)
		}
	}

	var start K
	found := false
	for _, key := range g.order {
		if pendingParents[key] > 0 && len(parents[key]) > 0 {
			start = key
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	visited := map[K]int{}
	path := []K{}
	current := start
	for {
		if at, seen := visited[current]; seen {
			loop := slices.Clone(path[at:])
			// the walk went backwards, reverse into edge order
			slices.Reverse(loop)
			return loop
		}
		visited[current] = len(path)
		path = append(path, current)
		current = parents[current][0]
	}
}
