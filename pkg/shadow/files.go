package shadow

import (
	"fmt"
	"strings"

	"github.com/moby/sys/user"
)

// files reads local users and groups from passwd and group files.
type files struct {
	passwdPath string
	groupPath  string
}

// User returns the named user, nil when it does not exist.
func (f files) User(name string) (*User, error) {
	users, err := f.Users()
	if err != nil {
		return nil, err
	}
	for _, entry := range users {
		if entry.Name == name {
			return &entry, nil
		}
	}
	return nil, nil //nolint:nilnil // nil means the user does not exist
}

// Group returns the named group, nil when it does not exist.
func (f files) Group(name string) (*Group, error) {
	groups, err := f.Groups()
	if err != nil {
		return nil, err
	}
	for _, entry := range groups {
		if entry.Name == name {
			return &entry, nil
		}
	}
	return nil, nil //nolint:nilnil // nil means the group does not exist
}

// Users returns all users in file order.
func (f files) Users() ([]User, error) {
	entries, err := user.ParsePasswdFileFilter(
		f.passwdPath,
		func(entry user.User) bool {
			return isLocalEntry(entry.Name)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading users from %q: %w", f.passwdPath, err)
	}

	groups, err := f.Groups()
	if err != nil {
		return nil, err
	}
	groupNames := map[int]string{}
	memberships := map[string][]string{}
	for _, group := range groups {
		// first entry wins on duplicate GIDs, matching libc
		if _, ok := groupNames[group.GID]; !ok {
			groupNames[group.GID] = group.Name
		}
		for _, member := range group.Members {
			memberships[member] = append(memberships[member], group.Name)
		}
	}

	users := make([]User, len(entries))
	for i, entry := range entries {
		users[i] = User{
			Name:   entry.Name,
			UID:    entry.Uid,
			GID:    entry.Gid,
			Group:  groupNames[entry.Gid],
			Groups: memberships[entry.Name],
			Home:   entry.Home,
			Shell:  entry.Shell,
		}
	}
	return users, nil
}

// Groups returns all groups in file order.
func (f files) Groups() ([]Group, error) {
	entries, err := user.ParseGroupFileFilter(
		f.groupPath,
		func(entry user.Group) bool {
			return isLocalEntry(entry.Name)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading groups from %q: %w", f.groupPath, err)
	}

	groups := make([]Group, len(entries))
	for i, entry := range entries {
		groups[i] = Group{
			Name:    entry.Name,
			GID:     entry.Gid,
			Members: entry.List,
		}
	}
	return groups, nil
}

// isLocalEntry reports whether name is a local account entry.
// The passwd parser keeps comment lines,
// NIS compat entries (+name, -name, +) are not local accounts.
func isLocalEntry(name string) bool {
	return name != "" && !strings.ContainsAny(name[:1], "#+-")
}
