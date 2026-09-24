//go:build unix

package std

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ntnn/tensile/pkg/diff"
)

// chmodMask covers the bits os.Chmod can change on unix.
const chmodMask = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky

// chmodNeedsExecution reports whether the changeable mode bits of path
// differ from mode.
// Returns the masked current and desired modes rendered for diffing,
// a missing path renders as [diff.Absent].
func chmodNeedsExecution(path string, mode os.FileMode) (bool, string, string, error) {
	desired := mode & chmodMask
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, diff.Absent, desired.String(), nil
	}
	if err != nil {
		return false, "", "", fmt.Errorf("checking mode: %w", err)
	}
	current := info.Mode() & chmodMask
	return current != desired, current.String(), desired.String(), nil
}

// chmodApply sets the changeable mode bits of path to mode.
func chmodApply(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
