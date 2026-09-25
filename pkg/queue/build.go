package queue

import (
	"fmt"
	"strings"

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
	// implicitReqs records the identities each node is required by
	implicitReqs map[tensile.Identity][]tensile.Identity
	// claimed records which identities have been claimed by which nodes
	claimed map[tensile.Identity]tensile.Identity
	// handlers maps handler identities to their notifiers
	handlers map[tensile.Identity][]tensile.Identity
	// subqueues maps [NamedQueue] identities to the node identities they contain
	subqueues map[tensile.Identity][]tensile.Identity
	// breadcrumbs maps node identities to the [NamedQueue] chain they
	// were added through, innermost first
	breadcrumbs map[tensile.Identity][]tensile.Identity
}

func newBuild() *build {
	return &build{
		implicitDeps: map[tensile.Identity][]tensile.Identity{},
		implicitReqs: map[tensile.Identity][]tensile.Identity{},
		claimed:      map[tensile.Identity]tensile.Identity{},
		handlers:     map[tensile.Identity][]tensile.Identity{},
		subqueues:    map[tensile.Identity][]tensile.Identity{},
		breadcrumbs:  map[tensile.Identity][]tensile.Identity{},
	}
}

// describe describes an identity in the context of the graph.
// If the node was added to the graph directly the identity is returned as a string.
// If the node originates from a [NamedQueue] the chain of breadcrumbs is added to the output.
func (b *build) describe(identity tensile.Identity) string {
	crumbs, ok := b.breadcrumbs[identity]
	if !ok {
		return identity.String()
	}

	parts := make([]string, len(crumbs))
	for i, crumb := range crumbs {
		parts[i] = crumb.String()
	}

	// identity from queue1->queue2->...
	return identity.String() + " from " + strings.Join(parts, "->")
}

// resolve resolves an identity.
// If the identity belongs to a subqueue it resolves to each node that subqueue provides.
// Otherwise it resolves to the identity of the node claiming the identity.
func (b *build) resolve(identity tensile.Identity) []tensile.Identity {
	if nodes, ok := b.subqueues[identity]; ok {
		return nodes
	}
	if claimer, ok := b.claimed[identity]; ok {
		return []tensile.Identity{claimer}
	}
	return []tensile.Identity{}
}

// addSubqueues registers the named queue memberships and breadcrumbs,
// claiming each queue identity.
func (b *build) addSubqueues(
	members map[tensile.Identity][]tensile.Identity,
	breadcrumbs map[tensile.Identity][]tensile.Identity,
) error {
	b.breadcrumbs = breadcrumbs
	for identity, nodes := range members {
		if claimer, claimed := b.claimed[identity]; claimed {
			return fmt.Errorf("identity %s is already claimed by %s", identity, b.describe(claimer))
		}
		if _, exists := b.subqueues[identity]; exists {
			return fmt.Errorf("subqueue %s already exists", identity)
		}
		b.claimed[identity] = identity
		b.subqueues[identity] = nodes
	}
	return nil
}

// addNodes calls [build.addNode] for each nodes.
func (b *build) addNodes(nodes []tensile.Identifier) error {
	for _, node := range nodes {
		if err := b.addNode(node); err != nil {
			return fmt.Errorf("error adding node %s: %w", b.describe(node.Identity()), err)
		}
	}
	return nil
}

// addNodes registers the given node as [tensile.Node] in the graph and as handler if it is a [tensile.Handler].
//
//nolint:cyclop // it's mostly straight forward, could at most extract claimed identities
func (b *build) addNode(identifier tensile.Identifier) error {
	node := tensile.NewNode(identifier)
	identity := node.Identity()

	// add claimed identities
	if claimer, claimed := b.claimed[identity]; claimed {
		return fmt.Errorf("identity %s is already claimed by node %s", identity, b.describe(claimer))
	}
	claims, err := node.Conflicts()
	if err != nil {
		return fmt.Errorf("error getting conflicts for node %s: %w", b.describe(identity), err)
	}
	for _, claim := range claims {
		if claimer, ok := b.claimed[claim]; ok {
			return fmt.Errorf("identity %s is already claimed by node %s", claim, b.describe(claimer))
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
		if _, exists := b.handlers[identity]; !exists {
			b.handlers[identity] = []tensile.Identity{}
		}
	}

	// add as notifier for implicitly notified nodes
	notified, err := node.Notifies()
	if err != nil {
		return fmt.Errorf("error getting notifications for node %s: %w", b.describe(identity), err)
	}
	for _, handler := range notified {
		b.handlers[handler] = append(b.handlers[handler], identity)
	}

	// add the implicit deps for later processing
	deps, err := node.DependsOn()
	if err != nil {
		return fmt.Errorf("error getting dependencies for node %s: %w", b.describe(identity), err)
	}
	b.implicitDeps[identity] = deps

	// add the implicit requisites for later processing
	reqs, err := node.RequiredBy()
	if err != nil {
		return fmt.Errorf("error getting requisites for node %s: %w", b.describe(identity), err)
	}
	b.implicitReqs[identity] = reqs

	return nil
}

// addDependencies adds the edges to the graph.
func (b *build) addDependencies(deps []graph.Edge[tensile.Identity]) error {
	for _, dep := range deps {
		if _, ok := b.claimed[dep.From]; !ok {
			return fmt.Errorf("node %s has not been added", b.describe(dep.From))
		}
		if _, ok := b.claimed[dep.To]; !ok {
			return fmt.Errorf("node %s has not been added", b.describe(dep.To))
		}
		froms := b.resolve(dep.From)
		tos := b.resolve(dep.To)
		for _, from := range froms {
			for _, to := range tos {
				if err := b.execution.AddEdge(from, to); err != nil {
					return fmt.Errorf("error adding dependency from %s to %s: %w",
						b.describe(from), b.describe(to), err)
				}
			}
		}
	}
	return nil
}

// addNotifies verifies that the handler and each node is registered and
// adds edges between them.
func (b *build) addNotifies(notifies []notification) error {
	for _, intent := range notifies {
		if _, isHandler := b.handlers[intent.handler]; !isHandler {
			return fmt.Errorf("handler %s has not been added", b.describe(intent.handler))
		}
		for _, notifier := range intent.notifiers {
			if _, ok := b.claimed[notifier]; !ok {
				return fmt.Errorf("node %s has not been added", b.describe(notifier))
			}
			for _, resolved := range b.resolve(notifier) {
				b.handlers[intent.handler] = append(b.handlers[intent.handler], resolved)
				if err := b.execution.AddEdge(resolved, intent.handler); err != nil {
					return fmt.Errorf("error adding notification edge from %s to %s: %w",
						b.describe(resolved), b.describe(intent.handler), err)
				}
			}
		}
	}
	return nil
}

// implicit adds the implicit edges between nodes as reported by the nodes themselves.
// should be called after adding all nodes.
//
//nolint:cyclop // this is fine, the linter is just overzealous
func (b *build) implicit() error {
	for identity, deps := range b.implicitDeps {
		for _, dep := range deps {
			resolved := b.resolve(dep)
			// get the identity of the node claiming the identity of the dependency
			if len(resolved) == 0 {
				// no node claimed the dependent identity, skip the implicit edge
				continue
			}
			for _, claimer := range resolved {
				if err := b.execution.AddEdge(claimer, identity); err != nil {
					return fmt.Errorf("error adding implicit dependency edge from %s to %s: %w",
						b.describe(claimer), b.describe(identity), err)
				}
			}
		}
	}

	for identity, reqs := range b.implicitReqs {
		for _, req := range reqs {
			resolved := b.resolve(req)
			if len(resolved) == 0 {
				// no node claimed the requisite identity, skip the implicit edge
				continue
			}
			for _, claimer := range resolved {
				if err := b.execution.AddEdge(identity, claimer); err != nil {
					return fmt.Errorf("error adding implicit requisite edge from %s to %s: %w",
						b.describe(identity), b.describe(claimer), err)
				}
			}
		}
	}

	for handler, notifiers := range b.handlers {
		claimer, ok := b.claimed[handler]
		if !ok {
			// no node claimed the handler identity, skip the implicit edges
			continue
		}
		for _, notifier := range notifiers {
			for _, r := range b.resolve(notifier) {
				if err := b.execution.AddEdge(r, claimer); err != nil {
					return fmt.Errorf("error adding implicit notification from %s to %s: %w",
						b.describe(r), b.describe(claimer), err)
				}
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
	work.dependencies = dependencies
	work.claims = b.claimed

	// map handlers to the claiming nodes, otherwise a notification
	// doesn't trigger the intended node
	work.handlers = make(map[tensile.Identity][]tensile.Identity, len(b.handlers))
	for handler, notifiers := range b.handlers {
		claimer, ok := b.claimed[handler]
		if !ok {
			// nothing claims the target, no node to trigger
			continue
		}
		work.handlers[claimer] = notifiers
	}

	order := make([]*tensile.Node, len(sorted))
	for i, identity := range sorted {
		node, ok := b.execution.Get(identity)
		if !ok {
			return nil, fmt.Errorf("sorted node %s missing from graph", b.describe(identity))
		}
		order[i] = node
	}
	work.order = order

	return work, nil
}
