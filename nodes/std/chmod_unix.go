//go:build unix

package std

import (
	"fmt"
	"os"
)

// chmodMask covers the bits os.Chmod can change on unix.
const chmodMask = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky

// chmodNeedsExecution reports whether the changeable mode bits of path
// differ from mode.
func chmodNeedsExecution(path string, mode os.FileMode) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("checking mode: %w", err)
	}
	return info.Mode()&chmodMask != mode&chmodMask, nil
}

// chmodApply sets the changeable mode bits of path to mode.
func chmodApply(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
