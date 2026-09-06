//go:build windows

package tensilestd

import (
	"fmt"
	"os"
)

// chmodWriteBit is the owner write bit, the only permission windows exposes via the read-only attribute.
const chmodWriteBit = os.FileMode(0o200)

// chmodNeedsExecution reports whether the read-only state of path differs from mode.
func chmodNeedsExecution(path string, mode os.FileMode) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("checking mode: %w", err)
	}
	// directories don't need execution on windows
	if info.IsDir() {
		return false, nil
	}
	return info.Mode()&chmodWriteBit != mode&chmodWriteBit, nil
}

// chmodApply sets the read-only attribute of path from the owner write bit of mode.
func chmodApply(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
