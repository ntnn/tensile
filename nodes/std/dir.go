package std

import (
	"fmt"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Dir)(nil)
var _ tensile.Validator = (*Dir)(nil)
var _ tensile.Provider = (*Dir)(nil)
var _ tensile.Depender = (*Dir)(nil)
var _ tensile.Executor = (*Dir)(nil)

// DirIdentity returns the identity of the node managing the directory at path.
func DirIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("dir", "path", path)
}

// DefaultDirMode is the mode applied to directories when none is set.
const DefaultDirMode = 0o755 | os.ModeDir

// Dir ensures a directory exists with specified ownership and permissions.
type Dir struct {
	Chmod
	Chown

	Path string
}

// Validate implements [tensile.Validator].
func (d *Dir) Validate(_ tensile.Wire) error {
	d.Chmod.Path = d.Path
	d.Chown.Path = d.Path
	if d.FileMode == 0 {
		d.FileMode = DefaultDirMode
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (d *Dir) Identity() tensile.Identity {
	return DirIdentity(d.Path)
}

// Provides implements [tensile.Provider].
func (d *Dir) Provides() ([]tensile.Identity, error) {
	return []tensile.Identity{DirIdentity(d.Path)}, nil
}

// DependsOn implements [tensile.Depender].
func (d *Dir) DependsOn() ([]tensile.Identity, error) {
	return parentDirIdentities(d.Path), nil
}

// NeedsExecution implements [tensile.Executor].
func (d *Dir) NeedsExecution(s tensile.Wire) (bool, error) {
	info, err := os.Stat(d.Path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("error checking directory: %w", err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%q exists but is not a directory", d.Path)
	}
	return d.Chmod.NeedsExecution(s)
}

// Execute implements [tensile.Executor].
func (d *Dir) Execute(s tensile.Wire) error {
	if err := os.MkdirAll(d.Path, d.FileMode.Perm()); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	if err := d.Chmod.Execute(s); err != nil {
		return fmt.Errorf("error setting mode: %w", err)
	}
	return nil
}
