package queue

import (
	"fmt"

	"github.com/ntnn/tensile"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type Queue struct {
	nodes map[int64]*tensile.Node
}

func New() *Queue {
	return &Queue{
		nodes: make(map[int64]*tensile.Node),
	}
}

func (q *Queue) Enqueue(nodes ...any) error {
	for _, node := range nodes {
		tensileNode, ok := node.(*tensile.Node)
		if !ok {
			var err error
			tensileNode, err = tensile.NewNode(node)
			if err != nil {
				return fmt.Errorf("failed to create node for input: %w", err)
			}
		}

		if err := q.enqueue(tensileNode); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) enqueue(node *tensile.Node) error {
	if err := node.Validate(); err != nil {
		return fmt.Errorf("validation failed for node with ID %d: %w", node.ID(), err)
	}

	if _, exists := q.nodes[node.ID()]; exists {
		return fmt.Errorf("node with ID %d already exists in the queue", node.ID())
	}
	q.nodes[node.ID()] = node
	return nil
}

// Build returns the nodes in the queue in the order they should be
// executed. If there is a cycle in the graph, an error is returned.
func (q *Queue) Build() (*Work, error) {
	// Add all nodes to the graph first
	directed := simple.NewDirectedGraph()
	for _, node := range q.nodes {
		directed.AddNode(node)
	}

	work := new(Work)
	work.done = make(map[int64]struct{})

	// Build a map of provided values to the IDs of nodes that provide them
	providedValues, err := q.buildProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to build providers: %w", err)
	}
	work.providedValues = providedValues

	// Iterate over all nodes and check if any dependency they declare
	// is provided by another node. If so add an edge from the provider
	// to the depender.
	for _, node := range q.nodes {
		dependencies, err := node.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for node with ID %d: %w", node.ID(), err)
		}

		for _, dep := range dependencies {
			providers, ok := providedValues[dep]
			if !ok {
				// dependencies are not required, nodes are giving
				// every possible value they can depend on
				continue
			}
			for _, providerId := range providers {
				directed.SetEdge(directed.NewEdge(directed.Node(providerId), node))
			}
		}
	}

	sorted, err := topo.Sort(directed)
	if err != nil {
		// TODO unpack error? the error semes to contain the IDs etcpp
		// which is not useful for the end user when the ID is the
		// hashed node.
		return nil, fmt.Errorf("cycle detected in graph: %w", err)
	}

	ret := make([]*tensile.Node, len(sorted))
	for i, node := range sorted {
		ret[i] = node.(*tensile.Node)
	}
	work.order = ret

	return work, nil
}

// buildProviders builds a map of provided values to the IDs of the
// nodes that provider them.
func (q *Queue) buildProviders() (map[string][]int64, error) {
	ret := make(map[string][]int64)
	for _, node := range q.nodes {
		provides, err := node.Provides()
		if err != nil {
			return nil, fmt.Errorf("failed to get provides for node with ID %d: %w", node.ID(), err)
		}
		for _, p := range provides {
			ret[p] = append(ret[p], node.ID())
		}
	}
	return ret, nil
}
