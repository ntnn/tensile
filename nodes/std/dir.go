package std

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*Dir)(nil)
var _ tensile.Validator = (*Dir)(nil)
var _ tensile.Conflictor = (*Dir)(nil)
var _ tensile.Depender = (*Dir)(nil)
var _ tensile.Executor = (*Dir)(nil)

// DirIdentity returns the identity of the node managing the directory at path.
func DirIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("dir", "path", path)
}

// parentDirs returns a list of all parent directories.
// It does not handle relative paths.
func parentDirs(p string) []string {
	ret := []string{}
	var previous string
	for {
		previous = p
		p = filepath.Dir(p)
		if previous == p {
			return ret
		}
		ret = append(ret, p)
	}
}

// ParentDirIdentities returns [Dir] identities for all parents of p.
func ParentDirIdentities(p string) []tensile.Identity {
	dirs := parentDirs(p)
	ret := make([]tensile.Identity, len(dirs))
	for i, dir := range dirs {
		ret[i] = DirIdentity(dir)
	}
	return ret
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

// DependsOn implements [tensile.Depender].
func (d *Dir) DependsOn() ([]tensile.Identity, error) {
	return ParentDirIdentities(d.Path), nil
}

// Conflicts implements [tensile.Conflictor].
func (d *Dir) Conflicts() ([]tensile.Identity, error) {
	return []tensile.Identity{FileIdentity(d.Path)}, nil
}

func (d *Dir) needsExecution(_ tensile.Wire) (bool, *diff.FieldChange, error) {
	info, err := os.Stat(d.Path)
	if os.IsNotExist(err) {
		return true, &diff.FieldChange{Field: "directory", Old: diff.Absent, New: d.FileMode.String()}, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("error checking directory: %w", err)
	}
	if !info.IsDir() {
		return false, nil, fmt.Errorf("%q exists but is not a directory", d.Path)
	}
	return false, nil, nil
}

// NeedsExecution implements [tensile.Executor].
func (d *Dir) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	dirNeeds, dirChange, err := d.needsExecution(wire)
	if err != nil {
		return false, nil, err
	}
	chmodNeeds, chmodChange, err := d.Chmod.needsExecution(wire)
	if err != nil {
		return false, nil, err
	}
	return dirNeeds || chmodNeeds, diff.NewFieldChanges(dirChange, chmodChange), nil
}

// Execute implements [tensile.Executor].
func (d *Dir) Execute(s tensile.Wire) (tensile.Diff, error) {
	if err := os.MkdirAll(d.Path, d.FileMode.Perm()); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}
	if _, err := d.Chmod.Execute(s); err != nil {
		return nil, fmt.Errorf("error setting mode: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}
