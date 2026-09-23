package queue

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/graph"
)

// build is the intermediary between a [Queue] and the resulting [Work].
//
// It essentially processes the nodes, dependencies and notifications,
// as well as the implicit dependencies, etc.pp. recorded in the [Queue]
// and puts them into a [graph.Graph] and uses the sort to build an
// execution order and find cycles.
// That is then placed into the [Work] for engines to process.
//
// NOTE(ntnn): Previously all this was part of [Queue], but that was
// getting quite messy and didn't look very nice.
// I also don't like this intermediary but at least the concerns are
// separated and it reads well.
type build struct {
	execution graph.Graph[tensile.Identity, *tensile.Node]
	// implicitDeps records the implicit dependencies of each node
	implicitDeps map[tensile.Identity][]tensile.Identity
	// claimed records which identities have been claimed by which nodes
	claimed map[tensile.Identity]tensile.Identity
	// handlers maps handler identities to their notifiers
	handlers map[tensile.Identity][]tensile.Identity
}

func newBuild() *build {
	return &build{
		implicitDeps: map[tensile.Identity][]tensile.Identity{},
		claimed:      map[tensile.Identity]tensile.Identity{},
		handlers:     map[tensile.Identity][]tensile.Identity{},
	}
}

// addNodes calls [build.addNode] for each nodes.
func (b *build) addNodes(nodes []tensile.Identifier) error {
	for i, node := range nodes {
		if err := b.addNode(node); err != nil {
			return fmt.Errorf("error adding node %d %q: %w", i, node, err)
		}
	}
	return nil
}

// addNodes registers the given node as [tensile.Node] in the graph and as handler if it is a [tensile.Handler].
func (b *build) addNode(identifier tensile.Identifier) error {
	node := tensile.NewNode(identifier)
	identity := node.Identity()

	// add claimed identities
	if claimer, claimed := b.claimed[identity]; claimed {
		return fmt.Errorf("identity %s is already claimed by node %s", identity, claimer)
	}
	claims, err := node.Conflicts()
	if err != nil {
		return fmt.Errorf("error getting conflicts for node %s: %w", identity, err)
	}
	for _, claim := range claims {
		if claimer, ok := b.claimed[claim]; ok {
			return fmt.Errorf("identity %s is already claimed by node %s", claim, claimer)
		}
		b.claimed[claim] = identity
	}
	b.claimed[identity] = identity

	// add to graph
	if err := b.execution.Add(identity, tensile.NewNode(node)); err != nil {
		return fmt.Errorf("adding node: %w", err)
	}

	// add as handler
	if _, isHandler := identifier.(*tensile.Handler); isHandler {
		b.handlers[identity] = []tensile.Identity{}
	}

	// add as notifier for implicitly notified nodes
	notified, err := node.Notifies()
	if err != nil {
		return fmt.Errorf("error getting notifications for node %s: %w", identity, err)
	}
	for _, handler := range notified {
		b.handlers[handler] = append(b.handlers[handler], identity)
	}

	// add the implicit deps for later processing
	deps, err := node.DependsOn()
	if err != nil {
		return fmt.Errorf("error getting dependencies for node %s: %w", identity, err)
	}
	b.implicitDeps[identity] = deps

	return nil
}

// addDependencies adds the edges to the graph.
func (b *build) addDependencies(deps []graph.Edge[tensile.Identity]) error {
	for _, dep := range deps {
		if err := b.execution.AddEdge(dep.From, dep.To); err != nil {
			return fmt.Errorf("adding dependency: %w", err)
		}
	}
	return nil
}

// addNotifies verifies that the handler and each node is registered and
// adds edges between them.
func (b *build) addNotifies(notifies []notification) error {
	for _, intent := range notifies {
		if _, isHandler := b.handlers[intent.handler]; !isHandler {
			return fmt.Errorf("handler %s has not been added", intent.handler)
		}
		for _, notifier := range intent.notifiers {
			if _, exists := b.execution.Get(notifier); !exists {
				return fmt.Errorf("node %s has not been added", notifier)
			}
			b.handlers[intent.handler] = append(b.handlers[intent.handler], notifier)
			if err := b.execution.AddEdge(notifier, intent.handler); err != nil {
				return fmt.Errorf("adding edge: %w", err)
			}
		}
	}
	return nil
}

// implicit adds the implicit edges between nodes as reported by the nodes themselves.
// should be called after adding all nodes.
func (b *build) implicit() error {
	for identity, deps := range b.implicitDeps {
		for _, dep := range deps {
			// get the identity of the node claiming the identity of the dependency
			claimer, ok := b.claimed[dep]
			if !ok {
				// no node claimed the dependent identity, skip the implicit edge
				continue
			}
			if err := b.execution.AddEdge(claimer, identity); err != nil {
				return fmt.Errorf("error adding implicit dependency edge from %q to %q: %w", dep, identity, err)
			}
		}
	}

	for handler, notifiers := range b.handlers {
		if _, ok := b.claimed[handler]; !ok {
			// no node claimed the handler identity, skip the implicit edges
			continue
		}
		for _, notifier := range notifiers {
			if err := b.execution.AddEdge(notifier, handler); err != nil {
				return fmt.Errorf("error adding implicit notification from %q to  %q: %w", notifier, handler, err)
			}
		}
	}

	return nil
}

// work assembles the [Work] from the wired execution graph.
func (b *build) work() (*Work, error) {
	sorted, dependencies, err := b.execution.TopoSort()
	if err != nil {
		return nil, fmt.Errorf("cycle detected in graph: %w", err)
	}

	work := newWork()
	work.handlers = b.handlers
	work.dependencies = dependencies

	order := make([]*tensile.Node, len(sorted))
	for i, identity := range sorted {
		node, ok := b.execution.Get(identity)
		if !ok {
			return nil, fmt.Errorf("sorted node %s missing from graph", identity)
		}
		order[i] = node
	}
	work.order = order

	return work, nil
}
