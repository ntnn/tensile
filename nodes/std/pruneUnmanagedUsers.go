package std

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/ntnn/tensile/pkg/shadow"
)

var (
	_ tensile.Identifier = (*PruneUnmanagedUsers)(nil)
	_ tensile.Validator  = (*PruneUnmanagedUsers)(nil)
	_ tensile.Serializer = (*PruneUnmanagedUsers)(nil)
	_ tensile.Executor   = (*PruneUnmanagedUsers)(nil)
)

// PruneUnmanagedUsersIdentity returns the identity of the node pruning unmanaged users.
func PruneUnmanagedUsersIdentity() tensile.Identity {
	return tensile.AsIdentity("pruneUnmanagedUsers")
}

// PruneUnmanagedUsers deletes users whose [UserIdentity] is not claimed by any node,
// including their home directory as far as the tools do.
// Root (uid 0) is never deleted.
type PruneUnmanagedUsers struct {
	// IncludeSystem also prunes users with a system UID.
	IncludeSystem bool
	// IDMin nil defaults to [shadow.DefaultIDMin].
	IDMin *int
	// IDMax nil defaults to [shadow.DefaultIDMax].
	IDMax *int

	shadow shadow.Service
}

// Identity implements [tensile.Identifier].
func (p *PruneUnmanagedUsers) Identity() tensile.Identity {
	return PruneUnmanagedUsersIdentity()
}

// Validate implements [tensile.Validator].
func (p *PruneUnmanagedUsers) Validate(_ tensile.Wire) error {
	svc, err := accountService(p.shadow)
	if err != nil {
		return err
	}
	p.shadow = svc
	return nil
}

// SerializesOn implements [tensile.Serializer].
func (p *PruneUnmanagedUsers) SerializesOn() []string {
	return []string{accountSerializeKey}
}

// NeedsExecution implements [tensile.Executor].
func (p *PruneUnmanagedUsers) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
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
func (p *PruneUnmanagedUsers) Execute(wire tensile.Wire) (tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return nil, err
	}
	for _, name := range unmanaged {
		if err := p.shadow.DeleteUser(wire.Context(), name, true); err != nil {
			return nil, fmt.Errorf("deleting unmanaged user: %w", err)
		}
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// unmanaged returns the names of users not claimed by any node.
func (p *PruneUnmanagedUsers) unmanaged(wire tensile.Wire) ([]string, error) {
	topo, err := wire.Topology()
	if err != nil {
		return nil, fmt.Errorf("getting topology: %w", err)
	}
	users, err := p.shadow.Users()
	if err != nil {
		return nil, fmt.Errorf("reading users: %w", err)
	}

	unmanaged := []string{}
	for _, user := range users {
		if user.UID == 0 {
			continue
		}
		if !p.IncludeSystem && shadow.IsSystemID(user.UID, p.IDMin, p.IDMax) {
			continue
		}
		if _, claimed := topo.Claimer(UserIdentity(user.Name)); claimed {
			continue
		}
		unmanaged = append(unmanaged, user.Name)
	}
	return unmanaged, nil
}

// prunedChanges returns one present to absent change per name.
func prunedChanges(names []string) *diff.FieldChanges {
	changes := make([]*diff.FieldChange, len(names))
	for i, name := range names {
		changes[i] = &diff.FieldChange{
			Field: name,
			Old:   string(tensile.Present),
			New:   diff.Absent,
		}
	}
	return diff.NewFieldChanges(changes...)
}
