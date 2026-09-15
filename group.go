package tensile

var _ Identifier = (*Group)(nil)

// Group is a collection of [Node].
// Groups may contain other groups.
type Group struct {
	Graph

	name string
}

// NewGroup returns a new [Group] with the given name.
func NewGroup(name string) *Group {
	return &Group{name: name}
}

// Name returns the group's name.
func (g *Group) Name() string {
	return g.name
}

// Identity implements [Identifier].
func (g *Group) Identity() Identity {
	return AsIdentity("group", "name", g.name)
}
