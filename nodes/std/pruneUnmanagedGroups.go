package std

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/shadow"
)

var (
	_ tensile.Identifier = (*PruneUnmanagedGroups)(nil)
	_ tensile.Validator  = (*PruneUnmanagedGroups)(nil)
	_ tensile.Depender   = (*PruneUnmanagedGroups)(nil)
	_ tensile.Serializer = (*PruneUnmanagedGroups)(nil)
	_ tensile.Executor   = (*PruneUnmanagedGroups)(nil)
)

// PruneUnmanagedGroupsIdentity returns the identity of the node pruning unmanaged groups.
func PruneUnmanagedGroupsIdentity() tensile.Identity {
	return tensile.AsIdentity("pruneUnmanagedGroups")
}

// PruneUnmanagedGroups deletes groups whose [GroupIdentity] is not claimed by any node.
// Gid 0 and primary groups of existing users are never deleted.
type PruneUnmanagedGroups struct {
	// IncludeSystem also prunes groups with a system GID.
	IncludeSystem bool
	// IDMin nil defaults to [shadow.DefaultIDMin].
	IDMin *int
	// IDMax nil defaults to [shadow.DefaultIDMax].
	IDMax *int

	shadow shadow.Service
}

// Identity implements [tensile.Identifier].
func (p *PruneUnmanagedGroups) Identity() tensile.Identity {
	return PruneUnmanagedGroupsIdentity()
}

// Validate implements [tensile.Validator].
func (p *PruneUnmanagedGroups) Validate(_ tensile.Wire) error {
	svc, err := accountService(p.shadow)
	if err != nil {
		return err
	}
	p.shadow = svc
	return nil
}

// DependsOn implements [tensile.Depender].
// Pruned users release their primary groups first.
func (p *PruneUnmanagedGroups) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{PruneUnmanagedUsersIdentity()}, nil
}

// SerializesOn implements [tensile.Serializer].
func (p *PruneUnmanagedGroups) SerializesOn() []string {
	return []string{accountSerializeKey}
}

// NeedsExecution implements [tensile.Executor].
func (p *PruneUnmanagedGroups) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return false, nil, err
	}
	if len(unmanaged) == 0 {
		return false, nil, nil
	}
	return true, prunedChanges(unmanaged), nil
}

// Execute implements [tensile.Executor].
func (p *PruneUnmanagedGroups) Execute(wire tensile.Wire) (tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return nil, err
	}
	for _, name := range unmanaged {
		if err := p.shadow.DeleteGroup(wire.Context(), name); err != nil {
			return nil, fmt.Errorf("deleting unmanaged group: %w", err)
		}
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// unmanaged returns the names of groups not claimed by any node.
func (p *PruneUnmanagedGroups) unmanaged(wire tensile.Wire) ([]string, error) {
	topo, err := wire.Topology()
	if err != nil {
		return nil, fmt.Errorf("getting topology: %w", err)
	}
	groups, err := p.shadow.Groups()
	if err != nil {
		return nil, fmt.Errorf("reading groups: %w", err)
	}
	primary, err := p.primaryGIDs()
	if err != nil {
		return nil, err
	}

	unmanaged := []string{}
	for _, group := range groups {
		if group.GID == 0 || primary[group.GID] {
			continue
		}
		if !p.IncludeSystem && shadow.IsSystemID(group.GID, p.IDMin, p.IDMax) {
			continue
		}
		if _, claimed := topo.Claimer(GroupIdentity(group.Name)); claimed {
			continue
		}
		unmanaged = append(unmanaged, group.Name)
	}
	return unmanaged, nil
}

// primaryGIDs returns the GIDs used as primary group by any user.
func (p *PruneUnmanagedGroups) primaryGIDs() (map[int]bool, error) {
	users, err := p.shadow.Users()
	if err != nil {
		return nil, fmt.Errorf("reading users: %w", err)
	}
	primary := map[int]bool{}
	for _, user := range users {
		primary[user.GID] = true
	}
	return primary, nil
}
