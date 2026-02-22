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

func (q *Queue) Enqueue(node *tensile.Node) error {
	if _, exists := q.nodes[node.ID()]; exists {
		return fmt.Errorf("node with ID %d already exists in the queue", node.ID())
	}
	q.nodes[node.ID()] = node
	return nil
}

// Build returns the nodes in the queue in the order they should be
// executed. If there is a cycle in the graph, an error is returned.
func (q *Queue) Build() ([]*tensile.Node, error) {
	directed := simple.NewDirectedGraph()
	for _, node := range q.nodes {
		directed.AddNode(node)
	}

	// TODO add edges based on node dependencies

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

	return ret, nil
}
