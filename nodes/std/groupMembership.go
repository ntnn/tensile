package std

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/ntnn/tensile/pkg/shadow"
)

var (
	_ tensile.Identifier = (*GroupMembership)(nil)
	_ tensile.Validator  = (*GroupMembership)(nil)
	_ tensile.Depender   = (*GroupMembership)(nil)
	_ tensile.Serializer = (*GroupMembership)(nil)
	_ tensile.Executor   = (*GroupMembership)(nil)
)

// GroupMembershipIdentity returns the identity of the supplementary membership of user in group.
func GroupMembershipIdentity(user, group string) tensile.Identity {
	return tensile.AsIdentity("groupMembership", "user", user, "group", group)
}

// GroupMembership ensures a user is or is not a supplementary member of a group.
type GroupMembership struct {
	User  string
	Group string
	// State defaults to [tensile.Present].
	// [tensile.ReadOnly] is not supported.
	State tensile.State

	shadow shadow.Service
}

// Identity implements [tensile.Identifier].
func (m *GroupMembership) Identity() tensile.Identity {
	return GroupMembershipIdentity(m.User, m.Group)
}

// Validate implements [tensile.Validator].
func (m *GroupMembership) Validate(_ tensile.Wire) error {
	if m.User == "" {
		return errors.New("user is required")
	}
	if m.Group == "" {
		return errors.New("group is required")
	}
	switch m.State.OrDefault() { //nolint:exhaustive // ReadOnly is unsupported, rejected by default
	case tensile.Present, tensile.Absent:
	default:
		return fmt.Errorf("unknown state %q", m.State)
	}
	svc, err := accountService(m.shadow)
	if err != nil {
		return err
	}
	m.shadow = svc
	return nil
}

// DependsOn implements [tensile.Depender].
func (m *GroupMembership) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{
		UserIdentity(m.User),
		GroupIdentity(m.Group),
	}, nil
}

// SerializesOn implements [tensile.Serializer].
func (m *GroupMembership) SerializesOn() []string {
	return []string{accountSerializeKey}
}

// NeedsExecution implements [tensile.Executor].
func (m *GroupMembership) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	group, err := m.shadow.Group(m.Group)
	if err != nil {
		return false, nil, fmt.Errorf("reading group: %w", err)
	}

	current := tensile.Absent
	if group != nil && slices.Contains(group.Members, m.User) {
		current = tensile.Present
	}
	if current == m.State.OrDefault() {
		return false, nil, nil
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{
		Field: stateField,
		Old:   string(current),
		New:   string(m.State.OrDefault()),
	}), nil
}

// Execute implements [tensile.Executor].
func (m *GroupMembership) Execute(wire tensile.Wire) (tensile.Diff, error) {
	switch m.State.OrDefault() { //nolint:exhaustive // ReadOnly is unsupported, rejected by Validate
	case tensile.Present:
		if err := m.shadow.AddMember(wire.Context(), m.User, m.Group); err != nil {
			return nil, fmt.Errorf("adding member: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Absent:
		if err := m.shadow.RemoveMember(wire.Context(), m.User, m.Group); err != nil {
			return nil, fmt.Errorf("removing member: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	default:
		return nil, fmt.Errorf("unknown state %q", m.State)
	}
}
