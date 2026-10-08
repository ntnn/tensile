package shadow

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

var _ Service = (*busybox)(nil)

// busybox changes users and groups with the busybox applets.
// The applets parse arguments by position, "--" is not supported.
type busybox struct {
	files
}

// ApplyUser implements [Service].
// Changing UID, primary group, home or shell of an existing user is unsupported.
func (b *busybox) ApplyUser(ctx context.Context, desired UserSpec) error {
	current, err := b.User(desired.Name)
	if err != nil {
		return err
	}
	if current == nil {
		if err := run(ctx, "adduser", adduserArgs(desired)...); err != nil {
			return err
		}
		// adduser -G also lists the user as member of its primary group
		created, err := b.User(desired.Name)
		if err != nil {
			return err
		}
		if created == nil {
			return fmt.Errorf("user %q missing after adduser", desired.Name)
		}
		return b.applyGroups(ctx, desired.Name, created.Groups, desired.Groups)
	}
	if err := unsupportedUserChange(*current, desired); err != nil {
		return err
	}
	if desired.Groups == nil {
		return nil
	}
	return b.applyGroups(ctx, desired.Name, current.Groups, desired.Groups)
}

// adduserArgs returns the adduser arguments creating desired without supplementary groups.
func adduserArgs(desired UserSpec) []string {
	args := []string{"-D"}
	if desired.UID != nil {
		args = append(args, "-u", strconv.Itoa(*desired.UID))
	}
	if desired.Group != "" {
		args = append(args, "-G", desired.Group)
	}
	if desired.Home != "" {
		args = append(args, "-h", desired.Home)
	}
	if desired.Shell != "" {
		args = append(args, "-s", desired.Shell)
	}
	return append(args, desired.Name)
}

// unsupportedUserChange returns an error wrapping [errors.ErrUnsupported]
// when desired changes a field busybox cannot modify.
func unsupportedUserChange(current User, desired UserSpec) error {
	var field string
	switch {
	case desired.UID != nil && *desired.UID != current.UID:
		field = "uid"
	case changed(desired.Group, current.Group):
		field = "primary group"
	case changed(desired.Home, current.Home):
		field = "home"
	case changed(desired.Shell, current.Shell):
		field = "shell"
	default:
		return nil
	}
	return fmt.Errorf("changing %s of %q with busybox: %w", field, desired.Name, errors.ErrUnsupported)
}

// applyGroups changes the supplementary groups of user from current to desired.
func (b *busybox) applyGroups(ctx context.Context, user string, current, desired []string) error {
	add, remove := membershipChanges(current, desired)
	for _, group := range add {
		if err := b.AddMember(ctx, user, group); err != nil {
			return err
		}
	}
	for _, group := range remove {
		if err := b.RemoveMember(ctx, user, group); err != nil {
			return err
		}
	}
	return nil
}

// DeleteUser implements [Service].
func (b *busybox) DeleteUser(ctx context.Context, name string, removeHome bool) error {
	if removeHome {
		return run(ctx, "deluser", "--remove-home", name)
	}
	return run(ctx, "deluser", name)
}

// ApplyGroup implements [Service].
// Changing the GID of an existing group is unsupported.
func (b *busybox) ApplyGroup(ctx context.Context, desired GroupSpec) error {
	current, err := b.Group(desired.Name)
	if err != nil {
		return err
	}

	if current == nil {
		args := []string{}
		if desired.GID != nil {
			args = append(args, "-g", strconv.Itoa(*desired.GID))
		}
		return run(ctx, "addgroup", append(args, desired.Name)...)
	}

	if desired.GID == nil || *desired.GID == current.GID {
		return nil
	}
	return fmt.Errorf("changing gid of %q with busybox: %w", desired.Name, errors.ErrUnsupported)
}

// DeleteGroup implements [Service].
func (b *busybox) DeleteGroup(ctx context.Context, name string) error {
	return run(ctx, "delgroup", name)
}

// AddMember implements [Service].
func (b *busybox) AddMember(ctx context.Context, user, group string) error {
	return run(ctx, "addgroup", user, group)
}

// RemoveMember implements [Service].
func (b *busybox) RemoveMember(ctx context.Context, user, group string) error {
	return run(ctx, "delgroup", user, group)
}
