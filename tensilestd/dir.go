package tensilestd

import (
	"context"

	"github.com/ntnn/tensile"
)

var _ tensile.Validator = (*Dir)(nil)
var _ tensile.Provider = (*Dir)(nil)
var _ tensile.Depender = (*Dir)(nil)
var _ tensile.Executor = (*Dir)(nil)

// DirRef is the reference type for directories.
const DirRef = tensile.Ref("Dir")

// Dir ensures a directory exists with specified ownership and permissions.
type Dir struct {
	Chmod
	Chown

	Path string
}

// Validate implements [tensile.Validator].
func (d *Dir) Validate(_ context.Context) error {
	d.Chmod.Path = d.Path
	d.Chown.Path = d.Path
	return nil
}

// Provides implements [tensile.Provider].
func (d *Dir) Provides() ([]tensile.NodeRef, error) {
	return []tensile.NodeRef{DirRef.To(d.Path)}, nil
}

// DependsOn implements [tensile.Depender].
func (d *Dir) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(d.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (d *Dir) NeedsExecution(_ context.Context) (bool, error) {
	return false, nil
}

// Execute implements [tensile.Executor].
func (d *Dir) Execute(_ context.Context) error {
	return nil
}
