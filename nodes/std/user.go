package std

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/ntnn/tensile/pkg/shadow"
)

var (
	_ tensile.Identifier = (*User)(nil)
	_ tensile.Validator  = (*User)(nil)
	_ tensile.Conflictor = (*User)(nil)
	_ tensile.Depender   = (*User)(nil)
	_ tensile.Requisite  = (*User)(nil)
	_ tensile.Serializer = (*User)(nil)
	_ tensile.Executor   = (*User)(nil)
	_ tensile.Reporter   = (*User)(nil)
)

// UserIdentity returns the identity of the node managing the named user.
func UserIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("user", "name", name)
}

// uidIdentity returns the conflict identity of a user ID.
func uidIdentity(uid int) tensile.Identity {
	return tensile.AsIdentity("uid", "id", strconv.Itoa(uid))
}

// GroupMembershipIdentity returns the identity of the supplementary membership of user in group.
func GroupMembershipIdentity(user, group string) tensile.Identity {
	return tensile.AsIdentity("groupMembership", "user", user, "group", group)
}

// UserData is the output reported by [User].
type UserData struct {
	shadow.User

	Present bool
}

// User ensures a local user is present or absent.
// Absent also removes the home directory as far as the tools do.
type User struct {
	Name string
	// UID nil is unmanaged, allocated on creation.
	UID *int
	// Group is the primary group.
	// Empty applies the tool default on creation and is unmanaged afterwards.
	Group string
	// Groups are the supplementary groups.
	// Nil is unmanaged.
	// Memberships claimed by other nodes are kept.
	Groups []string
	// Home empty applies the tool default on creation and is unmanaged afterwards.
	Home string
	// Shell empty applies the tool default on creation and is unmanaged afterwards.
	Shell string
	// State defaults to [tensile.Present].
	State tensile.State

	shadow shadow.Service
}

// Identity implements [tensile.Identifier].
func (u *User) Identity() tensile.Identity {
	return UserIdentity(u.Name)
}

// Validate implements [tensile.Validator].
func (u *User) Validate(_ tensile.Wire) error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if err := u.State.Valid(); err != nil {
		return err //nolint:wrapcheck // names the state
	}
	svc, err := accountService(u.shadow)
	if err != nil {
		return err
	}
	u.shadow = svc
	return nil
}

// Conflicts implements [tensile.Conflictor].
func (u *User) Conflicts() ([]tensile.Identity, error) {
	conflicts := []tensile.Identity{}
	if u.UID != nil {
		conflicts = append(conflicts, uidIdentity(*u.UID))
	}
	for _, group := range u.Groups {
		conflicts = append(conflicts, GroupMembershipIdentity(u.Name, group))
	}
	return conflicts, nil
}

// DependsOn implements [tensile.Depender].
func (u *User) DependsOn() ([]tensile.Identity, error) {
	deps := []tensile.Identity{}
	if u.Group != "" {
		deps = append(deps, GroupIdentity(u.Group))
	}
	for _, group := range u.Groups {
		deps = append(deps, GroupIdentity(group))
	}
	return deps, nil
}

// RequiredBy implements [tensile.Requisite].
// Nodes managing the home run after the user exists.
func (u *User) RequiredBy() ([]tensile.Identity, error) {
	if u.Home == "" {
		return nil, nil
	}
	return []tensile.Identity{FileIdentity(u.Home)}, nil
}

// SerializesOn implements [tensile.Serializer].
func (u *User) SerializesOn() []string {
	return []string{accountSerializeKey}
}

// NeedsExecution implements [tensile.Executor].
func (u *User) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	if u.State.OrDefault() == tensile.ReadOnly {
		return false, nil, nil
	}
	current, err := u.shadow.User(u.Name)
	if err != nil {
		return false, nil, fmt.Errorf("reading user: %w", err)
	}

	if u.State.OrDefault() == tensile.Absent {
		if current == nil {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: stateField,
			Old:   string(tensile.Present),
			New:   string(tensile.Absent),
		}), nil
	}

	spec, err := u.spec(wire, current)
	if err != nil {
		return false, nil, err
	}
	if current == nil {
		return true, diff.NewFieldChanges(creationChanges(spec)...), nil
	}
	changes := modificationChanges(*current, spec)
	if len(changes) == 0 {
		return false, nil, nil
	}
	return true, diff.NewFieldChanges(changes...), nil
}

// Execute implements [tensile.Executor].
func (u *User) Execute(wire tensile.Wire) (tensile.Diff, error) {
	switch u.State.OrDefault() {
	case tensile.ReadOnly:
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Absent:
		if err := u.shadow.DeleteUser(wire.Context(), u.Name, true); err != nil {
			return nil, fmt.Errorf("deleting user: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Present:
		current, err := u.shadow.User(u.Name)
		if err != nil {
			return nil, fmt.Errorf("reading user: %w", err)
		}
		spec, err := u.spec(wire, current)
		if err != nil {
			return nil, err
		}
		if err := u.shadow.ApplyUser(wire.Context(), spec); err != nil {
			return nil, fmt.Errorf("applying user: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	default:
		return nil, fmt.Errorf("unknown state %q", u.State)
	}
}

// Report implements [tensile.Reporter].
func (u *User) Report(_ tensile.Wire) (any, error) {
	current, err := u.shadow.User(u.Name)
	if err != nil {
		return nil, fmt.Errorf("reading user: %w", err)
	}
	if current == nil {
		return UserData{Name: u.Name}, nil
	}
	return UserData{
		User:    *current,
		Present: true,
	}, nil
}

// spec returns the desired user.
// current is nil when the user does not exist.
func (u *User) spec(wire tensile.Wire, current *shadow.User) (shadow.UserSpec, error) {
	spec := shadow.UserSpec{
		Name:  u.Name,
		UID:   u.UID,
		Group: u.Group,
		Home:  u.Home,
		Shell: u.Shell,
	}
	if u.Groups == nil {
		return spec, nil
	}

	spec.Groups = slices.Clone(u.Groups)
	if current == nil {
		return spec, nil
	}
	topo, err := wire.Topology()
	if err != nil {
		return spec, fmt.Errorf("getting topology: %w", err)
	}
	for _, group := range current.Groups {
		if slices.Contains(spec.Groups, group) {
			continue
		}
		claimer, claimed := topo.Claimer(GroupMembershipIdentity(u.Name, group))
		if claimed && claimer != u.Identity() {
			spec.Groups = append(spec.Groups, group)
		}
	}
	return spec, nil
}

// creationChanges returns the changes creating spec.
func creationChanges(spec shadow.UserSpec) []*diff.FieldChange {
	changes := []*diff.FieldChange{{
		Field: stateField,
		Old:   string(tensile.Absent),
		New:   string(tensile.Present),
	}}
	if spec.UID != nil {
		changes = appendChange(changes, "uid", diff.Absent, strconv.Itoa(*spec.UID))
	}
	if spec.Group != "" {
		changes = appendChange(changes, "group", diff.Absent, spec.Group)
	}
	if len(spec.Groups) > 0 {
		changes = appendChange(changes, "groups", diff.Absent, joinSorted(spec.Groups))
	}
	if spec.Home != "" {
		changes = appendChange(changes, "home", diff.Absent, spec.Home)
	}
	if spec.Shell != "" {
		changes = appendChange(changes, "shell", diff.Absent, spec.Shell)
	}
	return changes
}

// modificationChanges returns the changes from current to spec, unmanaged fields are skipped.
func modificationChanges(current shadow.User, spec shadow.UserSpec) []*diff.FieldChange {
	changes := []*diff.FieldChange{}
	if spec.UID != nil {
		changes = appendChange(changes, "uid", strconv.Itoa(current.UID), strconv.Itoa(*spec.UID))
	}
	if spec.Group != "" {
		changes = appendChange(changes, "group", current.Group, spec.Group)
	}
	if spec.Groups != nil {
		changes = appendChange(changes, "groups", joinSorted(current.Groups), joinSorted(spec.Groups))
	}
	if spec.Home != "" {
		changes = appendChange(changes, "home", current.Home, spec.Home)
	}
	if spec.Shell != "" {
		changes = appendChange(changes, "shell", current.Shell, spec.Shell)
	}
	return changes
}

// appendChange appends a change of field when old and desired differ.
func appendChange(changes []*diff.FieldChange, field, old, desired string) []*diff.FieldChange {
	if old == desired {
		return changes
	}
	return append(changes, &diff.FieldChange{
		Field: field,
		Old:   old,
		New:   desired,
	})
}

// joinSorted renders groups sorted and comma separated.
func joinSorted(groups []string) string {
	return strings.Join(slices.Sorted(slices.Values(groups)), ",")
}
