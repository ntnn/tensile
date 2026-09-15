package std

import (
	"fmt"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Symlink)(nil)
var _ tensile.Provider = (*Symlink)(nil)
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

// Provides implements [tensile.Provider].
func (s *Symlink) Provides() ([]tensile.Identity, error) {
	return []tensile.Identity{FileIdentity(s.Path)}, nil
}

// DependsOn implements [tensile.Depender].
func (s *Symlink) DependsOn() ([]tensile.Identity, error) {
	return parentDirIdentities(s.Path), nil
}

// NeedsExecution implements [tensile.Executor].
func (s *Symlink) NeedsExecution(_ tensile.Wire) (bool, error) {
	info, err := os.Lstat(s.Path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("error checking symlink: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, fmt.Errorf("%q exists but is not a symlink", s.Path)
	}

	target, err := os.Readlink(s.Path)
	if err != nil {
		return false, fmt.Errorf("error reading symlink: %w", err)
	}
	return target != s.Target, nil
}

// Execute implements [tensile.Executor].
func (s *Symlink) Execute(_ tensile.Wire) error {
	// os.Symlink fails when trying to overwrite, remove a pre-existing link first
	if _, err := os.Lstat(s.Path); err == nil {
		if err := os.Remove(s.Path); err != nil {
			return fmt.Errorf("error removing existing symlink: %w", err)
		}
	}
	if err := os.Symlink(s.Target, s.Path); err != nil {
		return fmt.Errorf("error creating symlink: %w", err)
	}
	return nil
}
