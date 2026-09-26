package std

import (
	"fmt"
	"os"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

// DescribeFileMode returns an approximate description the [os.FileMode].
// e.g. "Directory" if it contains [os.ModeDir].
func DescribeFileMode(mode os.FileMode) string {
	switch {
	case mode.IsRegular():
		return "File"
	case mode.IsDir():
		return "Directory"
	case mode&os.ModeSymlink != 0:
		return "Symlink"
	case mode&os.ModeCharDevice != 0:
		return "Character Device"
	case mode&os.ModeDevice != 0:
		return "Block Device"
	case mode&os.ModeNamedPipe != 0:
		return "Named Pipe"
	case mode&os.ModeSocket != 0:
		return "Socket"
	default:
		return "Unknown"
	}
}

var _ tensile.Identifier = (*Symlink)(nil)
var _ tensile.Conflictor = (*Symlink)(nil)
var _ tensile.Depender = (*Symlink)(nil)
var _ tensile.Executor = (*Symlink)(nil)

// Symlink ensures a symbolic link at Path points to Target.
type Symlink struct {
	Path   string
	Target string
}

// Identity implements [tensile.Identifier].
func (s *Symlink) Identity() tensile.Identity {
	return tensile.AsIdentity("symlink", "path", s.Path)
}

// Conflicts implements [tensile.Conflictor].
func (s *Symlink) Conflicts() ([]tensile.Identity, error) {
	return []tensile.Identity{FileIdentity(s.Path)}, nil
}

// DependsOn implements [tensile.Depender].
func (s *Symlink) DependsOn() ([]tensile.Identity, error) {
	return ParentDirIdentities(s.Path), nil
}

// NeedsExecution implements [tensile.Executor].
func (s *Symlink) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	info, err := os.Lstat(s.Path)
	if os.IsNotExist(err) {
		return true, s.targetDiff(diff.Absent), nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("error checking symlink: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		// path exists but is something other than a symlink
		return true, s.targetDiff(DescribeFileMode(info.Mode())), nil
	}

	target, err := os.Readlink(s.Path)
	if err != nil {
		return false, nil, fmt.Errorf("error reading symlink: %w", err)
	}
	if target == s.Target {
		return false, nil, nil
	}
	return true, s.targetDiff(target), nil
}

// targetDiff builds the diff from the current target to the desired one.
func (s *Symlink) targetDiff(current string) diff.FieldChanges {
	return diff.FieldChanges{
		Fields: []diff.FieldChange{
			{Field: "target", Old: current, New: s.Target},
		},
	}
}

// Execute implements [tensile.Executor].
func (s *Symlink) Execute(_ tensile.Wire) (tensile.Diff, error) {
	// os.Symlink fails when the path exists, remove anything pre-existing first
	if _, err := os.Lstat(s.Path); err == nil {
		if err := os.RemoveAll(s.Path); err != nil {
			return nil, fmt.Errorf("error removing existing symlink: %w", err)
		}
	}
	if err := os.Symlink(s.Target, s.Path); err != nil {
		return nil, fmt.Errorf("error creating symlink: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}
