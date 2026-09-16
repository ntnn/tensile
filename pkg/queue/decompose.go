package queue

import (
	"fmt"

	"github.com/ntnn/tensile"
)

// decomposition flattens a graph with groups into a graph without groups.
// Every group is decomposed into its nodes with internal barrier nodes
// to facilitate dependencies to/from the group and notifications for
// handlers.
type decomposition struct {
	flat tensile.Graph
	// groups maps group identities to their enclosing barriers
	groups map[tensile.Identity]barriers
}

func newDecomposition() *decomposition {
	return &decomposition{
		groups: make(map[tensile.Identity]barriers),
	}
}

// walk iterates over a another graph and adds all of its nodes,
// dependencies and handlers to decomposition.
func (d *decomposition) walk(graph *tensile.Graph) ([]tensile.Identity, error) {
	members := []tensile.Identity{}
	for _, raw := range graph.Nodes() {
		if group, ok := raw.(*tensile.Group); ok {
			if err := d.group(group); err != nil {
				return nil, err
			}
			members = append(members, group.Identity())
			continue
		}

		if _, isGroup := d.groups[raw.Identity()]; isGroup {
			return nil, fmt.Errorf("node %s collides with a group in the queue", raw.Identity())
		}
		if err := d.flat.Add(raw); err != nil {
			return nil, err
		}
		members = append(members, raw.Identity())
	}

	for _, edge := range graph.Edges() {
		if err := d.flat.DependsOn(d.start(edge[1]), d.end(edge[0])); err != nil {
			return nil, err
		}
	}

	for handler, notifiers := range graph.Handlers() {
		if err := d.notifiedBy(handler, notifiers); err != nil {
			return nil, err
		}
	}

	return members, nil
}

// group is like walk but iterates over a group and calls walk on the wrapped graph.
func (d *decomposition) group(group *tensile.Group) error {
	identity := group.Identity()
	if _, exists := d.groups[identity]; exists {
		return fmt.Errorf("group %s already exists in the queue", identity)
	}
	if _, exists := d.flat.Get(identity); exists {
		return fmt.Errorf("group %s collides with a node in the queue", identity)
	}

	enclosing := newBarriers(group.Name())
	// reserve before recursing to error on self-containing groups
	d.groups[identity] = enclosing

	endHandler := tensile.NewHandler(enclosing.end)
	if err := d.flat.Add(enclosing.start, endHandler); err != nil {
		return err
	}
	if err := d.flat.DependsOn(enclosing.end.Identity(), enclosing.start.Identity()); err != nil {
		return err
	}

	members, err := d.walk(&group.Graph)
	if err != nil {
		return err
	}
	for _, member := range members {
		if err := d.flat.DependsOn(d.start(member), enclosing.start.Identity()); err != nil {
			return err
		}
		if err := d.flat.DependsOn(enclosing.end.Identity(), d.end(member)); err != nil {
			return err
		}
		if err := d.flat.NotifiedBy(endHandler, d.end(member)); err != nil {
			return err
		}
	}
	return nil
}

// notifiedBy registers the notifiers for the handler on the flat graph.
func (d *decomposition) notifiedBy(handler tensile.Identity, notifiers []tensile.Identity) error {
	raw, exists := d.flat.Get(handler)
	if !exists {
		return fmt.Errorf("handler %s is not in the queue", handler)
	}

	asserted, isHandler := raw.(*tensile.Handler)
	if !isHandler {
		return fmt.Errorf("node %s is not a handler", handler)
	}

	for _, notifier := range notifiers {
		if err := d.flat.NotifiedBy(asserted, d.end(notifier)); err != nil {
			return err
		}
	}
	return nil
}

// start is a helper to return the internal barrier group start identities if the identity is a group.
func (d *decomposition) start(identity tensile.Identity) tensile.Identity {
	if enclosing, isGroup := d.groups[identity]; isGroup {
		return enclosing.start.Identity()
	}
	return identity
}

// end is a helper to return the internal barrier group end identity if the identity is a group.
func (d *decomposition) end(identity tensile.Identity) tensile.Identity {
	if enclosing, isGroup := d.groups[identity]; isGroup {
		return enclosing.end.Identity()
	}
	return identity
}
