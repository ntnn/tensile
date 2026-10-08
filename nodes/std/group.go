package std

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/ntnn/tensile/pkg/shadow"
)

var (
	_ tensile.Identifier = (*Group)(nil)
	_ tensile.Validator  = (*Group)(nil)
	_ tensile.Conflictor = (*Group)(nil)
	_ tensile.Serializer = (*Group)(nil)
	_ tensile.Executor   = (*Group)(nil)
	_ tensile.Reporter   = (*Group)(nil)
)

// GroupIdentity returns the identity of the node managing the named group.
func GroupIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("group", "name", name)
}

// gidIdentity returns the conflict identity of a group ID.
func gidIdentity(gid int) tensile.Identity {
	return tensile.AsIdentity("gid", "id", strconv.Itoa(gid))
}

// GroupData is the output reported by [Group].
type GroupData struct {
	shadow.Group

	Present bool
}

// Group ensures a local group is present or absent.
type Group struct {
	Name string
	// GID nil is unmanaged, allocated on creation.
	GID *int
	// State defaults to [tensile.Present].
	State tensile.State

	shadow shadow.Service
}

// Identity implements [tensile.Identifier].
func (g *Group) Identity() tensile.Identity {
	return GroupIdentity(g.Name)
}

// Validate implements [tensile.Validator].
func (g *Group) Validate(_ tensile.Wire) error {
	if g.Name == "" {
		return errors.New("name is required")
	}
	if err := g.State.Valid(); err != nil {
		return err //nolint:wrapcheck // names the state
	}
	svc, err := accountService(g.shadow)
	if err != nil {
		return err
	}
	g.shadow = svc
	return nil
}

// Conflicts implements [tensile.Conflictor].
func (g *Group) Conflicts() ([]tensile.Identity, error) {
	if g.GID == nil {
		return nil, nil
	}
	return []tensile.Identity{gidIdentity(*g.GID)}, nil
}

// SerializesOn implements [tensile.Serializer].
func (g *Group) SerializesOn() []string {
	return []string{accountSerializeKey}
}

// NeedsExecution implements [tensile.Executor].
func (g *Group) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	if g.State.OrDefault() == tensile.ReadOnly {
		return false, nil, nil
	}
	current, err := g.shadow.Group(g.Name)
	if err != nil {
		return false, nil, fmt.Errorf("reading group: %w", err)
	}

	if g.State.OrDefault() == tensile.Absent {
		if current == nil {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: stateField,
			Old:   string(tensile.Present),
			New:   string(tensile.Absent),
		}), nil
	}

	if current == nil {
		var gid *diff.FieldChange
		if g.GID != nil {
			gid = &diff.FieldChange{
				Field: "gid",
				Old:   diff.Absent,
				New:   strconv.Itoa(*g.GID),
			}
		}
		return true, diff.NewFieldChanges(
			&diff.FieldChange{
				Field: stateField,
				Old:   string(tensile.Absent),
				New:   string(tensile.Present),
			},
			gid,
		), nil
	}

	if g.GID == nil || *g.GID == current.GID {
		return false, nil, nil
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{
		Field: "gid",
		Old:   strconv.Itoa(current.GID),
		New:   strconv.Itoa(*g.GID),
	}), nil
}

// Execute implements [tensile.Executor].
func (g *Group) Execute(wire tensile.Wire) (tensile.Diff, error) {
	switch g.State.OrDefault() {
	case tensile.ReadOnly:
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Absent:
		if err := g.shadow.DeleteGroup(wire.Context(), g.Name); err != nil {
			return nil, fmt.Errorf("deleting group: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Present:
		if err := g.shadow.ApplyGroup(wire.Context(), shadow.GroupSpec{
			Name: g.Name,
			GID:  g.GID,
		}); err != nil {
			return nil, fmt.Errorf("applying group: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	default:
		return nil, fmt.Errorf("unknown state %q", g.State)
	}
}

// Report implements [tensile.Reporter].
func (g *Group) Report(_ tensile.Wire) (any, error) {
	current, err := g.shadow.Group(g.Name)
	if err != nil {
		return nil, fmt.Errorf("reading group: %w", err)
	}
	if current == nil {
		return GroupData{Name: g.Name}, nil
	}
	return GroupData{
		Group:   *current,
		Present: true,
	}, nil
}
