package queue

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/ntnn/tensile"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// graphID hashes an identity into the int64 ID space gonum requires.
func graphID(identity tensile.Identity) int64 {
	h := fnv.New64a()
	_, err := h.Write([]byte(identity.String()))
	if err != nil {
		// fnv.Write never errors, if it does there's something seriously wrong.
		panic(err)
	}
	return int64(h.Sum64()) //nolint:gosec // deliberate wraparound, only used as opaque ID
}

// graphNode adapts a [tensile.Node] to [graph.Node].
type graphNode struct {
	id   int64
	node *tensile.Node
}

func (g graphNode) ID() int64 {
	return g.id
}

// Queue aggregates [tensile.Node] and then orders them based on their
// dependencies for execution.
type Queue struct {
	nodes map[tensile.Identity]*tensile.Node
	// extraEdges are manually added edges to be considered during graph
	// building
	extraEdges [][2]tensile.Identity
	// notifies maps node identities to the identities they notify
	notifies map[tensile.Identity][]tensile.Identity
	// handlers is a map of nodes that are handlers and also carries the
	// list of manually added notfiers
	handlers map[tensile.Identity][]tensile.Identity
}

// New returns a new [Queue].
func New() *Queue {
	return &Queue{
		nodes:    make(map[tensile.Identity]*tensile.Node),
		notifies: make(map[tensile.Identity][]tensile.Identity),
		handlers: make(map[tensile.Identity][]tensile.Identity),
	}
}

// Enqueue adds a value as a [tensile.Node] to the queue.
func (q *Queue) Enqueue(nodes ...tensile.Identifier) error {
	for _, raw := range nodes {
		node := tensile.NewNode(raw)

		if err := q.enqueue(node); err != nil {
			return err
		}

		if _, isHandler := raw.(*tensile.Handler); isHandler {
			q.handlers[node.Identity()] = []tensile.Identity{}
		}
	}
	return nil
}

func (q *Queue) enqueue(node *tensile.Node) error {
	identity := node.Identity()

	if _, exists := q.nodes[identity]; exists {
		return fmt.Errorf("node %s already exists in the queue", identity)
	}

	notifies, err := node.Notifies()
	if err != nil {
		return fmt.Errorf("failed to get notifications for node %s: %w", identity, err)
	}
	if len(notifies) > 0 {
		q.notifies[identity] = notifies
	}

	q.nodes[identity] = node
	return nil
}

// Depends adds a dependency from node to each of the nodes in
// dependsOn. If any of the nodes in dependsOn are not in the queue, an
// error is returned.
func (q *Queue) Depends(node tensile.Identifier, dependsOn ...tensile.Identifier) error {
	if _, exists := q.nodes[node.Identity()]; !exists {
		return fmt.Errorf("node %s is not in the queue", node.Identity())
	}

	for _, dep := range dependsOn {
		if _, exists := q.nodes[dep.Identity()]; !exists {
			return fmt.Errorf("dependency node %s is not in the queue", dep.Identity())
		}
		q.extraEdges = append(q.extraEdges, [2]tensile.Identity{dep.Identity(), node.Identity()})
	}
	return nil
}

// NotifiedBy adds the notifiers as notifiers for the handler.
// If either the handler or notifiers are not in the queue an error is returned.
func (q *Queue) NotifiedBy(handler *tensile.Handler, notifiers ...tensile.Identifier) error {
	if _, isKnown := q.handlers[handler.Identity()]; !isKnown {
		return fmt.Errorf("handler %s is not in the queue", handler.Identity())
	}

	for _, notifier := range notifiers {
		if _, exists := q.nodes[notifier.Identity()]; !exists {
			return fmt.Errorf("notifier node %s is not in the queue", notifier.Identity())
		}
		q.handlers[handler.Identity()] = append(q.handlers[handler.Identity()], notifier.Identity())
	}

	return nil
}

// Build returns the nodes in the queue in the order they should be
// executed. If there is a cycle in the graph, an error is returned.
func (q *Queue) Build() (*Work, error) { //nolint:cyclop
	// Add all nodes to the graph first
	directed := simple.NewDirectedGraph()
	for identity, node := range q.nodes {
		directed.AddNode(graphNode{id: graphID(identity), node: node})
	}

	// edge adds a directed edge from one identity to another.
	edge := func(from, to tensile.Identity) {
		directed.SetEdge(directed.NewEdge(
			directed.Node(graphID(from)),
			directed.Node(graphID(to)),
		))
	}

	work := new(Work)
	work.cond = sync.NewCond(&work.lock)
	work.done = make(map[tensile.Identity]bool)

	// Build a map of provided identities to the nodes that provide them
	provided, err := q.buildProvided()
	if err != nil {
		return nil, fmt.Errorf("failed to build providers: %w", err)
	}
	work.provided = provided

	// Build the handlers->notifiers map
	work.handlers = resolveNotifies(provided, q.notifies)

	// Add the manual notifiers
	for handler, notifiers := range q.handlers {
		work.handlers[handler] = append(work.handlers[handler], notifiers...)
	}

	// Handlers depend on their notifying nodes.
	for handler, notifiers := range work.handlers {
		for _, notifier := range notifiers {
			edge(notifier, handler)
		}
	}

	// Iterate over all nodes and check if any dependency they declare
	// is provided by another node. If so add an edge from the provider
	// to the depender.
	for identity, node := range q.nodes {
		dependencies, err := node.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for node %s: %w", identity, err)
		}

		for _, dep := range dependencies {
			providers, ok := provided[dep]
			if !ok {
				// dependencies are not required, nodes are giving
				// every possible value they can depend on
				continue
			}
			for _, provider := range providers {
				edge(provider, identity)
			}
		}
	}

	// Add any extra edges that were added via Depends.
	for _, extra := range q.extraEdges {
		edge(extra[0], extra[1])
	}

	sorted, err := topo.Sort(directed)
	if err != nil {
		return nil, fmt.Errorf("cycle detected in graph: %w", unpackCycles(err))
	}

	ret := make([]*tensile.Node, len(sorted))
	for i, node := range sorted {
		ret[i] = node.(graphNode).node
	}
	work.order = ret

	return work, nil
}

// unpackCycles translates the graph IDs in a topo.Unorderable error
// into identities.
func unpackCycles(err error) error {
	var unorderable topo.Unorderable
	if !errors.As(err, &unorderable) {
		return err
	}

	cycles := make([]string, len(unorderable))
	for i, cycle := range unorderable {
		identities := make([]string, len(cycle))
		for j, node := range cycle {
			identities[j] = node.(graphNode).node.Identity().String()
		}
		cycles[i] = fmt.Sprintf("%v", identities)
	}
	return fmt.Errorf("%v", cycles)
}

// buildProvided builds a map of provided identities to the identities
// of the nodes that provide them.
func (q *Queue) buildProvided() (map[tensile.Identity][]tensile.Identity, error) {
	ret := make(map[tensile.Identity][]tensile.Identity)
	for identity, node := range q.nodes {
		provides, err := node.Provides()
		if err != nil {
			return nil, fmt.Errorf("failed to get provides for node %s: %w", identity, err)
		}
		for _, p := range provides {
			ret[p] = append(ret[p], identity)
		}
	}
	return ret, nil
}

// resolveNotifies resolves the notifier->handler map to a handler->notifier map.
func resolveNotifies(
	provided map[tensile.Identity][]tensile.Identity,
	notifierToHandler map[tensile.Identity][]tensile.Identity,
) map[tensile.Identity][]tensile.Identity {
	ret := map[tensile.Identity][]tensile.Identity{}

	for notifier, targets := range notifierToHandler {
		for _, target := range targets {
			for _, handler := range provided[target] {
				ret[handler] = append(ret[handler], notifier)
			}
		}
	}

	return ret
}
