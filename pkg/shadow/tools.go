package shadow

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var _ Service = (*tools)(nil)

// tools changes users and groups with the shadow tools.
type tools struct {
	files

	// root is passed as --prefix unless it is "/".
	root string
}

// run executes a shadow tool against root.
func (t *tools) run(ctx context.Context, name string, args ...string) error {
	if t.root != "/" {
		args = append([]string{"--prefix", t.root}, args...)
	}
	return run(ctx, name, args...)
}

// ApplyUser implements [Service].
func (t *tools) ApplyUser(ctx context.Context, desired UserSpec) error {
	current, err := t.User(desired.Name)
	if err != nil {
		return err
	}
	if current == nil {
		return t.run(ctx, "useradd", useraddArgs(desired)...)
	}
	args := usermodArgs(*current, desired)
	if len(args) == 0 {
		return nil
	}
	return t.run(ctx, "usermod", args...)
}

// useraddArgs returns the useradd arguments creating desired.
func useraddArgs(desired UserSpec) []string {
	args := []string{"-m"}
	if desired.UID != nil {
		args = append(args, "-u", strconv.Itoa(*desired.UID))
	}
	if desired.Group != "" {
		args = append(args, "-g", desired.Group)
	}
	if len(desired.Groups) > 0 {
		args = append(args, "-G", strings.Join(desired.Groups, ","))
	}
	if desired.Home != "" {
		args = append(args, "-d", desired.Home)
	}
	if desired.Shell != "" {
		args = append(args, "-s", desired.Shell)
	}
	return append(args, "--", desired.Name)
}

// usermodArgs returns the usermod arguments changing current to desired.
// Returns nil when nothing changes.
func usermodArgs(current User, desired UserSpec) []string {
	args := []string{}
	if desired.UID != nil && *desired.UID != current.UID {
		args = append(args, "-u", strconv.Itoa(*desired.UID))
	}
	if changed(desired.Group, current.Group) {
		args = append(args, "-g", desired.Group)
	}
	if desired.Groups != nil && !sameSet(desired.Groups, current.Groups) {
		args = append(args, "-G", strings.Join(desired.Groups, ","))
	}
	if changed(desired.Home, current.Home) {
		// -m moves the home contents
		args = append(args, "-d", desired.Home, "-m")
	}
	if changed(desired.Shell, current.Shell) {
		args = append(args, "-s", desired.Shell)
	}
	if len(args) == 0 {
		return nil
	}
	return append(args, "--", desired.Name)
}

// DeleteUser implements [Service].
func (t *tools) DeleteUser(ctx context.Context, name string, removeHome bool) error {
	args := []string{}
	if removeHome {
		args = append(args, "-r")
	}
	return t.run(ctx, "userdel", append(args, "--", name)...)
}

// ApplyGroup implements [Service].
func (t *tools) ApplyGroup(ctx context.Context, desired GroupSpec) error {
	current, err := t.Group(desired.Name)
	if err != nil {
		return err
	}

	if current == nil {
		args := []string{}
		if desired.GID != nil {
			args = append(args, "-g", strconv.Itoa(*desired.GID))
		}
		return t.run(ctx, "groupadd", append(args, "--", desired.Name)...)
	}

	if desired.GID == nil || *desired.GID == current.GID {
		return nil
	}
	return t.run(ctx, "groupmod", "-g", strconv.Itoa(*desired.GID), "--", desired.Name)
}

// DeleteGroup implements [Service].
func (t *tools) DeleteGroup(ctx context.Context, name string) error {
	return t.run(ctx, "groupdel", "--", name)
}

// AddMember implements [Service].
func (t *tools) AddMember(ctx context.Context, user, group string) error {
	current, err := t.member(user)
	if err != nil {
		return err
	}
	if slices.Contains(current.Groups, group) {
		return nil
	}
	return t.setGroups(ctx, user, append(slices.Clone(current.Groups), group))
}

// RemoveMember implements [Service].
func (t *tools) RemoveMember(ctx context.Context, user, group string) error {
	current, err := t.member(user)
	if err != nil {
		return err
	}
	if !slices.Contains(current.Groups, group) {
		return nil
	}
	remaining := slices.DeleteFunc(slices.Clone(current.Groups), func(g string) bool { return g == group })
	return t.setGroups(ctx, user, remaining)
}

// member returns the named user, an error when it does not exist.
func (t *tools) member(name string) (*User, error) {
	current, err := t.User(name)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("user %q does not exist", name)
	}
	return current, nil
}

// setGroups sets the supplementary groups of user.
// usermod is used instead of gpasswd, which has no --prefix.
func (t *tools) setGroups(ctx context.Context, user string, groups []string) error {
	return t.run(ctx, "usermod", "-G", strings.Join(groups, ","), "--", user)
}
