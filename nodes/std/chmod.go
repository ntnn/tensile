package std

import (
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Chmod)(nil)
var _ tensile.Depender = (*Chmod)(nil)
var _ tensile.Executor = (*Chmod)(nil)

// Chmod ensures a file has the specified permissions.
// FileMode is interpreted as unix permission bits.
type Chmod struct {
	Path     string
	FileMode os.FileMode
}

// Identity implements [tensile.Identifier].
func (c Chmod) Identity() tensile.Identity {
	return tensile.AsIdentity("chmod", "path", c.Path)
}

// DependsOn implements [tensile.Depender].
func (c Chmod) DependsOn() ([]tensile.Identity, error) {
	return parentDirIdentities(c.Path), nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chmod) NeedsExecution(_ tensile.Wire) (bool, error) {
	return chmodNeedsExecution(c.Path, c.FileMode)
}

// Execute implements [tensile.Executor].
func (c Chmod) Execute(_ tensile.Wire) error {
	return chmodApply(c.Path, c.FileMode)
}
