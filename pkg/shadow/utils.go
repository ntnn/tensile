package shadow

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// run executes the command.
// The error includes the command output.
func run(ctx context.Context, name string, args ...string) error {
	//nolint:gosec // args are fixed flags plus caller provided names and values
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// sameSet reports whether a and b contain the same strings, ignoring order and duplicates.
func sameSet(a, b []string) bool {
	return slices.Equal(
		slices.Compact(slices.Sorted(slices.Values(a))),
		slices.Compact(slices.Sorted(slices.Values(b))),
	)
}

// changed reports whether a managed, non-empty desired value differs from current.
func changed(desired, current string) bool {
	return desired != "" && desired != current
}

// membershipChanges returns the groups to add and to remove to get from current to desired.
func membershipChanges(current, desired []string) ([]string, []string) {
	add := []string{}
	for _, group := range desired {
		if !slices.Contains(current, group) && !slices.Contains(add, group) {
			add = append(add, group)
		}
	}
	remove := []string{}
	for _, group := range current {
		if !slices.Contains(desired, group) && !slices.Contains(remove, group) {
			remove = append(remove, group)
		}
	}
	return add, remove
}
