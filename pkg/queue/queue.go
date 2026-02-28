package queue

import (
	"fmt"

	"github.com/ntnn/tensile"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type Queue struct {
	nodes      map[int64]*tensile.Node
	extraEdges []graph.Edge
}

func New() *Queue {
	return &Queue{
		nodes: make(map[int64]*tensile.Node),
	}
}

func asTensileNode(in any) (*tensile.Node, error) {
	asserted, ok := in.(*tensile.Node)
	if ok {
		return asserted, nil
	}
	tensiled, err := tensile.NewNode(in)
	if err != nil {
		return nil, fmt.Errorf("failed to create node for input: %w", err)
	}
	return tensiled, nil
}

func (q *Queue) Enqueue(nodes ...any) error {
	for _, node := range nodes {
		tensileNode, err := asTensileNode(node)
		if err != nil {
			return err
		}

		if err := q.enqueue(tensileNode); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) enqueue(node *tensile.Node) error {
	if _, exists := q.nodes[node.ID()]; exists {
		return fmt.Errorf("node with ID %d already exists in the queue", node.ID())
	}
	q.nodes[node.ID()] = node
	return nil
}

// Depends adds a dependency from node to each of the nodes in
// dependsOn. If any of the nodes in dependsOn are not in the queue, an
// error is returned.
func (q *Queue) Depends(node any, dependsOn ...any) error {
	tensileNode, err := asTensileNode(node)
	if err != nil {
		return err
	}
	if _, exists := q.nodes[tensileNode.ID()]; !exists {
		return fmt.Errorf("node with ID %d is not in the queue", tensileNode.ID())
	}

	for _, dep := range dependsOn {
		tensileDep, err := asTensileNode(dep)
		if err != nil {
			return err
		}
		if _, exists := q.nodes[tensileDep.ID()]; !exists {
			return fmt.Errorf("dependency node with ID %d is not in the queue", tensileDep.ID())
		}
		q.extraEdges = append(q.extraEdges, simple.Edge{F: tensileDep, T: tensileNode})
	}
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

	// Build a map of provided node refs to the IDs of nodes that provide them
	providedRefs, err := q.buildProvidedRefs()
	if err != nil {
		return nil, fmt.Errorf("failed to build providers: %w", err)
	}
	work.providedRefs = providedRefs

	// Iterate over all nodes and check if any dependency they declare
	// is provided by another node. If so add an edge from the provider
	// to the depender.
	for _, node := range q.nodes {
		dependencies, err := node.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for node with ID %d: %w", node.ID(), err)
		}

		for _, dep := range dependencies {
			providers, ok := providedRefs[dep]
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

	// Add any extra edges that were added via Depends.
	for _, edge := range q.extraEdges {
		directed.SetEdge(edge)
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

// buildProvidedRefs builds a map of provided refs to the IDs of the
// nodes that provide them.
func (q *Queue) buildProvidedRefs() (map[tensile.NodeRef][]int64, error) {
	ret := make(map[tensile.NodeRef][]int64)
	for _, node := range q.nodes {
		provides, err := node.Provides()
		if err != nil {
			return nil, fmt.Errorf("failed to get provides for node with ID %d: %w", node.ID(), err)
		}
		for _, p := range provides {
			if ret[p] == nil {
				ret[p] = []int64{node.ID()}
			} else {
				ret[p] = append(ret[p], node.ID())
			}
		}
	}
	return ret, nil
}
