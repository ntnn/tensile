package tensile

import "fmt"

// Graph collects nodes, dependency edges and handler registrations.
// The zero value is usable.
// Graph is the shared primitive underneath [Group] and work queues.
type Graph struct {
	// nodes maps identities to the added values
	nodes map[Identity]Identifier
	// edges are dependency edges as [from, to] pairs
	edges [][2]Identity
	// handlers maps handler identities to their manual notifiers
	handlers map[Identity][]Identity
}

// Add adds nodes to the graph.
// [Handler] values are registered as handlers.
func (g *Graph) Add(nodes ...Identifier) error {
	if g.nodes == nil {
		g.nodes = make(map[Identity]Identifier)
	}
	if g.handlers == nil {
		g.handlers = make(map[Identity][]Identity)
	}

	for _, node := range nodes {
		identity := node.Identity()
		if _, exists := g.nodes[identity]; exists {
			return fmt.Errorf("node %s already exists", identity)
		}
		g.nodes[identity] = node

		if _, isHandler := node.(*Handler); isHandler {
			g.handlers[identity] = []Identity{}
		}
	}
	return nil
}

// DependsOn adds a dependency from node to each of the nodes in dependsOn.
// All referenced nodes must have been added.
func (g *Graph) DependsOn(node Identifier, dependsOn ...Identifier) error {
	if err := g.contains(node.Identity()); err != nil {
		return err
	}

	for _, dep := range dependsOn {
		if err := g.contains(dep.Identity()); err != nil {
			return err
		}
		g.edges = append(g.edges, [2]Identity{dep.Identity(), node.Identity()})
	}
	return nil
}

// RequiredBy is like [Graph.DependsOn] but adds a dependency from each
// of the nodes in requiredBy to node.
// All referenced nodes must have been added.
func (g *Graph) RequiredBy(node Identifier, requiredBy ...Identifier) error {
	if err := g.contains(node.Identity()); err != nil {
		return err
	}

	for _, req := range requiredBy {
		if err := g.contains(req.Identity()); err != nil {
			return err
		}
		g.edges = append(g.edges, [2]Identity{node.Identity(), req.Identity()})
	}
	return nil
}

// NotifiedBy adds the notifiers as notifiers for the handler.
// The handler and the notifiers must have been added.
// A group as notifier notifies when any of its nodes executed.
func (g *Graph) NotifiedBy(handler *Handler, notifiers ...Identifier) error {
	identity := handler.Identity()
	if _, isKnown := g.handlers[identity]; !isKnown {
		return fmt.Errorf("handler %s has not been added", identity)
	}

	for _, notifier := range notifiers {
		if err := g.contains(notifier.Identity()); err != nil {
			return err
		}
		g.handlers[identity] = append(g.handlers[identity], notifier.Identity())
	}
	return nil
}

// contains errors if the identity has not been added.
func (g *Graph) contains(identity Identity) error {
	if _, exists := g.nodes[identity]; exists {
		return nil
	}
	return fmt.Errorf("node %s has not been added", identity)
}

// Get returns the value added under the identity.
func (g *Graph) Get(identity Identity) (Identifier, bool) {
	node, ok := g.nodes[identity]
	return node, ok
}

// Nodes returns the added values.
func (g *Graph) Nodes() []Identifier {
	ret := make([]Identifier, 0, len(g.nodes))
	for _, node := range g.nodes {
		ret = append(ret, node)
	}
	return ret
}

// Edges returns the dependency edges as [from, to] pairs.
func (g *Graph) Edges() [][2]Identity {
	ret := make([][2]Identity, len(g.edges))
	copy(ret, g.edges)
	return ret
}

// Handlers returns the handler identities mapped to their manual notifiers.
func (g *Graph) Handlers() map[Identity][]Identity {
	ret := make(map[Identity][]Identity, len(g.handlers))
	for identity, notifiers := range g.handlers {
		ret[identity] = append([]Identity{}, notifiers...)
	}
	return ret
}
