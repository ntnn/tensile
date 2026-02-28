package tensilestd

import (
	"context"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Depender = (*Chmod)(nil)
var _ tensile.Executor = (*Chmod)(nil)

// ChmodRef is the reference type for chmod operations.
const ChmodRef = tensile.Ref("Chmod")

// Chmod ensures a file has the specified permissions.
type Chmod struct {
	Path     string
	FileMode os.FileMode
}

// DependsOn implements [tensile.Depender].
func (c Chmod) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(c.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chmod) NeedsExecution(_ context.Context) (bool, error) {
	info, err := os.Stat(c.Path)
	if err != nil {
		return false, err
	}
	return info.Mode() != c.FileMode, nil
}

// Execute implements [tensile.Executor].
func (c Chmod) Execute(_ context.Context) error {
	return os.Chmod(c.Path, c.FileMode)
}
