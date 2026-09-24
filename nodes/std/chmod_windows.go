//go:build windows

package std

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ntnn/tensile/pkg/diff"
)

// chmodWriteBit is the owner write bit, the only permission windows exposes via the read-only attribute.
const chmodWriteBit = os.FileMode(0o200)

// chmodNeedsExecution reports whether the read-only state of path differs from mode.
// Returns the masked current and desired modes rendered for diffing,
// a missing path renders as [diff.Absent].
func chmodNeedsExecution(path string, mode os.FileMode) (bool, string, string, error) {
	desired := mode & chmodWriteBit
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, diff.Absent, desired.String(), nil
	}
	if err != nil {
		return false, "", "", fmt.Errorf("checking mode: %w", err)
	}
	// directories don't need execution on windows
	if info.IsDir() {
		return false, "", "", nil
	}
	current := info.Mode() & chmodWriteBit
	return current != desired, current.String(), desired.String(), nil
}

// chmodApply sets the read-only attribute of path from the owner write bit of mode.
func chmodApply(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
