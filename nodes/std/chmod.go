package std

import (
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Depender = (*Chmod)(nil)
var _ tensile.Executor = (*Chmod)(nil)

// ChmodRef is the reference type for chmod operations.
const ChmodRef = tensile.Ref("Chmod")

// Chmod ensures a file has the specified permissions.
// FileMode is interpreted as unix permission bits.
type Chmod struct {
	Path     string
	FileMode os.FileMode
}

// DependsOn implements [tensile.Depender].
func (c Chmod) DependsOn() ([]tensile.NodeRef, error) {
	return tensile.ToMany(DirRef, parentDirs(c.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chmod) NeedsExecution(_ tensile.Wire) (bool, error) {
	return chmodNeedsExecution(c.Path, c.FileMode)
}

// Execute implements [tensile.Executor].
func (c Chmod) Execute(_ tensile.Wire) error {
	return chmodApply(c.Path, c.FileMode)
}
