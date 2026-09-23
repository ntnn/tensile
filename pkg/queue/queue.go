package queue

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/graph"
)

// Queue aggregates [tensile.Node] and then orders them based on their
// dependencies for execution.
type Queue struct {
	graph graph.Graph[tensile.Identity, tensile.Identifier]
	// handlers maps handler identities to their manual notifiers
	handlers map[tensile.Identity][]tensile.Identity
	// members maps group identities to their member node identities
	members map[tensile.Identity][]tensile.Identity
}

// New returns a new [Queue].
func New() *Queue {
	return &Queue{
		handlers: map[tensile.Identity][]tensile.Identity{},
		members:  map[tensile.Identity][]tensile.Identity{},
	}
}

// Add adds values as [tensile.Node] to the queue.
// [tensile.Group] are dissolved into their members.
func (q *Queue) Add(nodes ...tensile.Identifier) error {
	for _, node := range nodes {
		if group, isGroup := node.(*tensile.Group); isGroup {
			if err := q.addGroup(group); err != nil {
				return err
			}
			continue
		}
		if err := q.addNode(node); err != nil {
			return err
		}
	}
	return nil
}

// addNode adds a single non-group value.
func (q *Queue) addNode(node tensile.Identifier) error {
	identity := node.Identity()
	if _, isGroup := q.members[identity]; isGroup {
		return fmt.Errorf("node %s collides with a group in the queue", identity)
	}
	if err := q.graph.Add(identity, node); err != nil {
		return fmt.Errorf("adding node: %w", err)
	}
	if _, isHandler := node.(*tensile.Handler); isHandler {
		q.handlers[identity] = []tensile.Identity{}
	}
	return nil
}

// addGroup dissolves a group into its member nodes, recording the
// membership for dependency and notification resolution.
func (q *Queue) addGroup(group *tensile.Group) error {
	identity := group.Identity()
	if _, exists := q.members[identity]; exists {
		return fmt.Errorf("group %s already exists in the queue", identity)
	}
	if _, exists := q.graph.Get(identity); exists {
		return fmt.Errorf("group %s collides with a node in the queue", identity)
	}
	// reserve before recursing to error on self-containing groups
	q.members[identity] = nil

	members := []tensile.Identity{}
	for _, raw := range group.Nodes() {
		if err := q.Add(raw); err != nil {
			return err
		}
		members = append(members, q.resolve(raw.Identity())...)
	}
	q.members[identity] = members

	for _, pair := range group.Edges() {
		if err := q.edges(pair[0], pair[1]); err != nil {
			return err
		}
	}

	for handler, notifiers := range group.Handlers() {
		if err := q.notify(handler, notifiers); err != nil {
			return err
		}
	}
	return nil
}

// resolve expands a group identity to its member identities.
// Non-group identities resolve to themselves.
func (q *Queue) resolve(identity tensile.Identity) []tensile.Identity {
	if members, isGroup := q.members[identity]; isGroup {
		return members
	}
	return []tensile.Identity{identity}
}

// edges adds dependency edges, expanding groups on both sides.
func (q *Queue) edges(from, to tensile.Identity) error {
	for _, f := range q.resolve(from) {
		for _, t := range q.resolve(to) {
			if err := q.graph.DependsOn(t, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// notify registers resolved notifiers for the handler.
func (q *Queue) notify(handler tensile.Identity, notifiers []tensile.Identity) error {
	raw, exists := q.graph.Get(handler)
	if !exists {
		return fmt.Errorf("handler %s is not in the queue", handler)
	}
	asserted, isHandler := raw.(*tensile.Handler)
	if !isHandler {
		return fmt.Errorf("node %s is not a handler", handler)
	}

	for _, notifier := range notifiers {
		for _, resolved := range q.resolve(notifier) {
			if err := q.graph.NotifiedBy(asserted, resolved); err != nil {
				return err
			}
		}
	}
	return nil
}

// DependsOn adds a dependency from node to each of the nodes in dependsOn.
// All referenced nodes must have been added.
func (q *Queue) DependsOn(node tensile.Identifier, dependsOn ...tensile.Identifier) error {
	for _, dep := range dependsOn {
		if err := q.edges(dep.Identity(), node.Identity()); err != nil {
			return err
		}
	}
	return nil
}

// RequiredBy is like [Queue.DependsOn] but adds a dependency from each
// of the nodes in requiredBy to node.
// All referenced nodes must have been added.
func (q *Queue) RequiredBy(node tensile.Identifier, requiredBy ...tensile.Identifier) error {
	for _, req := range requiredBy {
		if err := q.edges(node.Identity(), req.Identity()); err != nil {
			return err
		}
	}
	return nil
}

// NotifiedBy adds the notifiers as notifiers for the handler.
// If either the handler or notifiers are not in the queue an error is returned.
func (q *Queue) NotifiedBy(handler *tensile.Handler, notifiers ...tensile.Identifier) error {
	identities := make([]tensile.Identity, len(notifiers))
	for i, notifier := range notifiers {
		identities[i] = notifier.Identity()
	}
	return q.notify(handler.Identity(), identities)
}

// Build returns the nodes in the queue in the order they should be
// executed. If there is a cycle in the graph, an error is returned.
func (q *Queue) Build() (*Work, error) { //nolint:cyclop
	nodes := make(map[tensile.Identity]*tensile.Node)
	directed := simple.NewDirectedGraph()
	for _, raw := range q.graph.Nodes() {
		node := tensile.NewNode(raw)
		nodes[node.Identity()] = node
		directed.AddNode(graphNode{id: graphID(node.Identity()), node: node})
	}

	work := newWork()

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
	for handler, notifiers := range q.graph.Handlers() {
		work.handlers[handler] = append(work.handlers[handler], notifiers...)
	}

	// Handlers depend on their notifying nodes.
	for handler, notifiers := range work.handlers {
		for _, notifier := range notifiers {
			edge(notifier, handler)
		}
	}

	// Iterate over all nodes and check if any dependency they declare is claimed by another node.
	// If so add an edge from the provider to the depender.
	// A dependency on a group identity resolves to the group's members.
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
			for _, member := range q.members[dep] {
				edge(member, identity)
			}
		}
	}

	// Add the declared edges.
	for _, declared := range q.graph.Edges() {
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
