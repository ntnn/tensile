package tensile

import (
	"fmt"
	"maps"
)

var _ Identifier = (*Group)(nil)

// Group is a collection of [Node] and other [Group].
type Group struct {
	name   string
	nodes  map[Identity]*Node
	groups map[Identity]*Group
	// edges are dependency edges as [from, to] pairs
	edges [][2]Identity
}

// NewGroup returns a new [Group] with the given name.
func NewGroup(name string) *Group {
	return &Group{
		name:   name,
		nodes:  make(map[Identity]*Node),
		groups: make(map[Identity]*Group),
	}
}

// Identity implements [Identifier].
func (g *Group) Identity() Identity {
	return AsIdentity("group", "name", g.name)
}

// Add adds nodes to the group.
// Nested [Group] are resolved into their own groups.
func (g *Group) Add(nodes ...Identifier) error {
	for _, raw := range nodes {
		identity := raw.Identity()
		if g.contains(identity) {
			return fmt.Errorf("identity %s already exists in group %s", identity, g.name)
		}

		if sub, ok := raw.(*Group); ok {
			g.groups[identity] = sub
			continue
		}

		g.nodes[identity] = NewNode(raw)
	}
	return nil
}

// Depends adds a dependency from node to each of the nodes in dependsOn.
// All referenced nodes must have been added to the group.
func (g *Group) Depends(node Identifier, dependsOn ...Identifier) error {
	if !g.contains(node.Identity()) {
		return fmt.Errorf("node %s is not in group %s", node.Identity(), g.name)
	}

	for _, dep := range dependsOn {
		if !g.contains(dep.Identity()) {
			return fmt.Errorf("node %s is not in group %s", dep.Identity(), g.name)
		}
		g.edges = append(g.edges, [2]Identity{dep.Identity(), node.Identity()})
	}
	return nil
}

// contains errors if the identity is neither a node nor a group in the group.
func (g *Group) contains(identity Identity) bool {
	if _, exists := g.nodes[identity]; exists {
		return true
	}
	if _, exists := g.groups[identity]; exists {
		return true
	}
	return false
}

// Nodes returns the directly added nodes.
func (g *Group) Nodes() []*Node {
	ret := make([]*Node, 0, len(g.nodes))
	for _, node := range g.nodes {
		ret = append(ret, node)
	}
	return ret
}

// Groups returns the nested groups keyed by the identity they were added as.
func (g *Group) Groups() map[Identity]*Group {
	ret := make(map[Identity]*Group, len(g.groups))
	maps.Copy(ret, g.groups)
	return ret
}

// Edges returns the dependency edges as [from, to] pairs.
func (g *Group) Edges() [][2]Identity {
	ret := make([][2]Identity, len(g.edges))
	copy(ret, g.edges)
	return ret
}
