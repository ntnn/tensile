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
	graph tensile.Graph
}

// New returns a new [Queue].
func New() *Queue {
	return &Queue{}
}

// Enqueue adds values as [tensile.Node] to the queue.
func (q *Queue) Enqueue(nodes ...tensile.Identifier) error {
	return q.graph.Add(nodes...)
}

// DependsOn adds a dependency from node to each of the nodes in dependsOn.
// All referenced nodes must have been added.
func (q *Queue) DependsOn(node tensile.Identifier, dependsOn ...tensile.Identifier) error {
	return q.graph.DependsOn(node, dependsOn...)
}

// RequiredBy is like [Queue.DependsOn] but adds a dependency from each
// of the nodes in requiredBy to node.
// All referenced nodes must have been added.
func (q *Queue) RequiredBy(node tensile.Identifier, requiredBy ...tensile.Identifier) error {
	return q.graph.RequiredBy(node, requiredBy...)
}

// NotifiedBy adds the notifiers as notifiers for the handler.
// If either the handler or notifiers are not in the queue an error is returned.
func (q *Queue) NotifiedBy(handler *tensile.Handler, notifiers ...tensile.Identifier) error {
	return q.graph.NotifiedBy(handler, notifiers...)
}

// Build returns the nodes in the queue in the order they should be
// executed. If there is a cycle in the graph, an error is returned.
func (q *Queue) Build() (*Work, error) { //nolint:cyclop
	d := newDecomposition()
	if _, err := d.walk(&q.graph); err != nil {
		return nil, err
	}

	nodes := make(map[tensile.Identity]*tensile.Node)
	directed := simple.NewDirectedGraph()
	for _, raw := range d.flat.Nodes() {
		node := tensile.NewNode(raw)
		nodes[node.Identity()] = node
		directed.AddNode(graphNode{id: graphID(node.Identity()), node: node})
	}

	work := new(Work)
	work.cond = sync.NewCond(&work.lock)
	work.done = make(map[tensile.Identity]bool)
	work.dependencies = make(map[tensile.Identity][]tensile.Identity)
	work.held = make(map[string]tensile.Identity)

	// edge adds a directed edge from one identity to another and
	// records it as a runtime dependency, so manual and automatic
	// edges gate readiness the same way.
	edge := func(from, to tensile.Identity) {
		directed.SetEdge(directed.NewEdge(
			directed.Node(graphID(from)),
			directed.Node(graphID(to)),
		))
		work.dependencies[to] = append(work.dependencies[to], from)
	}

	// Build a map of identities to the node claiming them and collect notifications.
	//
	// A node provides a single identity - its own.
	// But it may claim multiple identities to prevent conflicts, e.g.
	// std.Symlink and std.File both claim the file identity to force an
	// error when both target the same path.
	//
	// Duplicate claims are a conflict.
	claimed, notifies, err := collect(nodes)
	if err != nil {
		return nil, err
	}

	// Build the handlers->notifiers map
	work.handlers = resolveNotifies(claimed, notifies)

	// Add the manual notifiers
	for handler, notifiers := range d.flat.Handlers() {
		work.handlers[handler] = append(work.handlers[handler], notifiers...)
	}

	// Handlers depend on their notifying nodes.
	for handler, notifiers := range work.handlers {
		for _, notifier := range notifiers {
			edge(notifier, handler)
		}
	}

	// Barrier start and end nodes of each group are referenced by the group identity.
	groups := make(map[tensile.Identity][]tensile.Identity, len(d.groups))
	for identity, enclosing := range d.groups {
		groups[identity] = []tensile.Identity{
			enclosing.start.Identity(),
			enclosing.end.Identity(),
		}
	}

	// Iterate over all nodes and check if any dependency they declare is claimed by another node.
	// If so add an edge from the provider to the depender.
	for identity, node := range nodes {
		dependencies, err := node.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for node %s: %w", identity, err)
		}

		for _, dep := range dependencies {
			if owner, ok := claimed[dep]; ok {
				edge(owner, identity)
				continue
			}
			for _, barrier := range groups[dep] {
				edge(barrier, identity)
			}
		}
	}

	// Add the declared edges.
	for _, declared := range d.flat.Edges() {
		edge(declared[0], declared[1])
	}

	sorted, err := topo.Sort(directed)
	if err != nil {
		return nil, fmt.Errorf("cycle detected in graph: %w", unpackCycles(err))
	}

	order := make([]*tensile.Node, len(sorted))
	for i, node := range sorted {
		order[i] = node.(graphNode).node
	}
	work.order = order

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

// collect builds the map of conflict identities to the identity of the
// node claiming them and the map of notifiers to their targets.
func collect(nodes map[tensile.Identity]*tensile.Node) (
	map[tensile.Identity]tensile.Identity,
	map[tensile.Identity][]tensile.Identity,
	error,
) {
	claimed := make(map[tensile.Identity]tensile.Identity)
	notifies := make(map[tensile.Identity][]tensile.Identity)

	claim := func(claim, owner tensile.Identity) error {
		if existing, ok := claimed[claim]; ok && existing != owner {
			return fmt.Errorf("nodes %s and %s conflict on %s", existing, owner, claim)
		}
		claimed[claim] = owner
		return nil
	}

	for identity, node := range nodes {
		if err := claim(identity, identity); err != nil {
			return nil, nil, err
		}

		conflicts, err := node.Conflicts()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get conflicts for node %s: %w", identity, err)
		}
		for _, c := range conflicts {
			if err := claim(c, identity); err != nil {
				return nil, nil, err
			}
		}

		notified, err := node.Notifies()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get notifications for node %s: %w", identity, err)
		}
		if len(notified) > 0 {
			notifies[identity] = notified
		}
	}

	return claimed, notifies, nil
}

// resolveNotifies resolves the notifier->handler map to a handler->notifier map.
func resolveNotifies(
	claimed map[tensile.Identity]tensile.Identity,
	notifierToHandler map[tensile.Identity][]tensile.Identity,
) map[tensile.Identity][]tensile.Identity {
	ret := map[tensile.Identity][]tensile.Identity{}

	for notifier, targets := range notifierToHandler {
		for _, target := range targets {
			if handler, ok := claimed[target]; ok {
				ret[handler] = append(ret[handler], notifier)
			}
		}
	}

	return ret
}
