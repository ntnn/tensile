package graph

import (
	"fmt"
	"iter"
	"slices"
)

// Edge is a directed edge from From to To.
type Edge[K comparable] struct {
	From, To K
}

// Graph is a directed graph of values keyed by K.
type Graph[K comparable, V any] struct {
	nodes map[K]V
	// order preserves insertion order for deterministic iteration
	order []K
	edges []Edge[K]
}

// Add adds the value under the key.
// Duplicate keys are rejected.
func (g *Graph[K, V]) Add(key K, value V) error {
	if g.nodes == nil {
		g.nodes = make(map[K]V)
	}
	if _, exists := g.nodes[key]; exists {
		return fmt.Errorf("node %v already exists", key)
	}
	g.nodes[key] = value
	g.order = append(g.order, key)
	return nil
}

// AddEdge adds a directed edge.
// Both keys must have been added, self edges are rejected.
func (g *Graph[K, V]) AddEdge(from, to K) error {
	if from == to {
		return fmt.Errorf("self edge on %v", from)
	}
	if _, exists := g.nodes[from]; !exists {
		return fmt.Errorf("node %v has not been added", from)
	}
	if _, exists := g.nodes[to]; !exists {
		return fmt.Errorf("node %v has not been added", to)
	}
	g.edges = append(g.edges, Edge[K]{From: from, To: to})
	return nil
}

// Get returns the value added under the key.
func (g *Graph[K, V]) Get(key K) (V, bool) {
	value, ok := g.nodes[key]
	return value, ok
}

// Len returns the number of nodes.
func (g *Graph[K, V]) Len() int {
	return len(g.nodes)
}

// Nodes iterates over keys and values in insertion order.
func (g *Graph[K, V]) Nodes() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, key := range g.order {
			if !yield(key, g.nodes[key]) {
				return
			}
		}
	}
}

// Edges returns a copy of the edges.
func (g *Graph[K, V]) Edges() []Edge[K] {
	return slices.Clone(g.edges)
}
