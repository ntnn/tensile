package std

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/ntnn/tensile/pkg/shadow"
)

var (
	_ tensile.Identifier = (*PruneUnmanagedGroupMemberships)(nil)
	_ tensile.Validator  = (*PruneUnmanagedGroupMemberships)(nil)
	_ tensile.Depender   = (*PruneUnmanagedGroupMemberships)(nil)
	_ tensile.Serializer = (*PruneUnmanagedGroupMemberships)(nil)
	_ tensile.Executor   = (*PruneUnmanagedGroupMemberships)(nil)
)

// PruneUnmanagedGroupMembershipsIdentity returns the identity of the node pruning unmanaged memberships of group.
// Empty group covers all groups.
func PruneUnmanagedGroupMembershipsIdentity(group string) tensile.Identity {
	return tensile.AsIdentity("pruneUnmanagedGroupMemberships", "group", group)
}

// PruneUnmanagedGroupMemberships removes supplementary members
// whose [GroupMembershipIdentity] is not claimed by any node.
// Primary group membership is not affected.
type PruneUnmanagedGroupMemberships struct {
	// Group limits pruning to the named group.
	// Empty prunes all groups.
	Group string
	// IncludeSystem also prunes members of groups with a system GID.
	// If the group is a system group (e.g. wheel) this must be true.
	IncludeSystem bool
	// IDMin nil defaults to [shadow.DefaultIDMin].
	IDMin *int
	// IDMax nil defaults to [shadow.DefaultIDMax].
	IDMax *int

	shadow shadow.Service
}

// membership is a supplementary membership of user in group.
type membership struct {
	user  string
	group string
}

// Identity implements [tensile.Identifier].
func (p *PruneUnmanagedGroupMemberships) Identity() tensile.Identity {
	return PruneUnmanagedGroupMembershipsIdentity(p.Group)
}

// Validate implements [tensile.Validator].
func (p *PruneUnmanagedGroupMemberships) Validate(_ tensile.Wire) error {
	svc, err := accountService(p.shadow)
	if err != nil {
		return err
	}
	p.shadow = svc
	return nil
}

// DependsOn implements [tensile.Depender].
func (p *PruneUnmanagedGroupMemberships) DependsOn() ([]tensile.Identity, error) {
	if p.Group == "" {
		return nil, nil
	}
	return []tensile.Identity{GroupIdentity(p.Group)}, nil
}

// SerializesOn implements [tensile.Serializer].
func (p *PruneUnmanagedGroupMemberships) SerializesOn() []string {
	return []string{accountSerializeKey}
}

// NeedsExecution implements [tensile.Executor].
func (p *PruneUnmanagedGroupMemberships) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return false, nil, err
	}
	if len(unmanaged) == 0 {
		return false, nil, nil
	}
	changes := make([]*diff.FieldChange, len(unmanaged))
	for i, member := range unmanaged {
		changes[i] = &diff.FieldChange{
			Field: member.group + "/" + member.user,
			Old:   string(tensile.Present),
			New:   diff.Absent,
		}
	}
	return true, diff.NewFieldChanges(changes...), nil
}

// Execute implements [tensile.Executor].
func (p *PruneUnmanagedGroupMemberships) Execute(wire tensile.Wire) (tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return nil, err
	}
	for _, member := range unmanaged {
		if err := p.shadow.RemoveMember(wire.Context(), member.user, member.group); err != nil {
			return nil, fmt.Errorf("removing unmanaged member: %w", err)
		}
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// unmanaged returns the memberships not claimed by any node.
func (p *PruneUnmanagedGroupMemberships) unmanaged(wire tensile.Wire) ([]membership, error) {
	topo, err := wire.Topology()
	if err != nil {
		return nil, fmt.Errorf("getting topology: %w", err)
	}
	groups, err := p.shadow.Groups()
	if err != nil {
		return nil, fmt.Errorf("reading groups: %w", err)
	}

	unmanaged := []membership{}
	for _, group := range groups {
		if p.Group != "" && group.Name != p.Group {
			continue
		}
		if !p.IncludeSystem && shadow.IsSystemID(group.GID, p.IDMin, p.IDMax) {
			continue
		}
		for _, member := range group.Members {
			if _, claimed := topo.Claimer(GroupMembershipIdentity(member, group.Name)); claimed {
				continue
			}
			unmanaged = append(unmanaged, membership{
				user:  member,
				group: group.Name,
			})
		}
	}
	return unmanaged, nil
}
