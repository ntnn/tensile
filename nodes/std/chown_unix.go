//go:build unix

package std

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"syscall"

	"github.com/moby/sys/user"
	"github.com/ntnn/tensile/pkg/diff"
)

// chownNeedsExecution reports whether the ownership of path differs from owner and group.
// Empty owner or group means "do not change".
func chownNeedsExecution(path string, owner, group string) (bool, *diff.FieldChange, *diff.FieldChange, error) {
	// If the path does not exist, it needs execution.
	_, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil, nil, nil
	}
	if err != nil {
		return false, nil, nil, fmt.Errorf("stating path: %w", err)
	}

	ownerNeeds, ownerChange, err := ownerNeedsExecution(path, owner)
	if err != nil {
		return false, nil, nil, err
	}

	groupNeeds, groupChange, err := groupNeedsExecution(path, group)
	if err != nil {
		return false, nil, nil, err
	}

	return ownerNeeds || groupNeeds, ownerChange, groupChange, nil
}

func ownerNeedsExecution(path string, owner string) (bool, *diff.FieldChange, error) {
	if owner == "" {
		return false, nil, nil
	}

	u, err := user.LookupUser(owner)
	if err != nil {
		return false, nil, fmt.Errorf("resolving owner %q: %w", owner, err)
	}

	currentUID, _, err := currentChown(path)
	if err != nil {
		return false, nil, err
	}
	if currentUID == u.Uid {
		return false, nil, nil
	}

	currentOwner, err := user.LookupUid(currentUID)
	if err != nil {
		currentOwner.Name = strconv.Itoa(currentUID)
	}

	return true, &diff.FieldChange{
		Field: "owner",
		Old:   currentOwner.Name,
		New:   owner,
	}, nil
}

func groupNeedsExecution(path string, group string) (bool, *diff.FieldChange, error) {
	if group == "" {
		return false, nil, nil
	}

	g, err := user.LookupGroup(group)
	if err != nil {
		return false, nil, fmt.Errorf("resolving group %q: %w", group, err)
	}

	_, currentGID, err := currentChown(path)
	if err != nil {
		return false, nil, err
	}
	if currentGID == g.Gid {
		return false, nil, nil
	}

	currentGroup, err := user.LookupGid(currentGID)
	if err != nil {
		currentGroup.Name = strconv.Itoa(currentGID)
	}

	return true, &diff.FieldChange{
		Field: "group",
		Old:   currentGroup.Name,
		New:   group,
	}, nil
}

func currentChown(path string) (int, int, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, 0, fmt.Errorf("stating path: %w", err)
	}

	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("stat result not *syscall.Stat_t: %T", info.Sys())
	}

	return int(sys.Uid), int(sys.Gid), nil
}

// chownApply sets the owner and group of path.
// Empty owner or group means "do not change".
func chownApply(path string, owner, group string) error {
	uid := -1
	if owner != "" {
		u, err := user.LookupUser(owner)
		if err != nil {
			return fmt.Errorf("resolving owner %q: %w", owner, err)
		}
		uid = u.Uid
	}
	gid := -1
	if group != "" {
		g, err := user.LookupGroup(group)
		if err != nil {
			return fmt.Errorf("resolving group %q: %w", group, err)
		}
		gid = g.Gid
	}
	return os.Chown(path, uid, gid)
}
