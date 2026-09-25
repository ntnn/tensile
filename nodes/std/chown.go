package std

import (
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Chown)(nil)
var _ tensile.Depender = (*Chown)(nil)
var _ tensile.Executor = (*Chown)(nil)

// Chown ensures a file has the specified owner and group.
type Chown struct {
	Path  string
	Owner string
	Group string
}

// Identity implements [tensile.Identifier].
func (c Chown) Identity() tensile.Identity {
	return tensile.AsIdentity("chown", "path", c.Path)
}

// DependsOn implements [tensile.Depender].
func (c Chown) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(c.Path),
		FileIdentity(c.Path),
	), nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chown) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	// TODO resolve owner and group names to numeric IDs
	// TODO check if the current owner and group match the desired ones
	return true, nil, nil
}

// Execute implements [tensile.Executor].
func (c Chown) Execute(_ tensile.Wire) (tensile.Diff, error) {
	// TODO resolve owner and group names to numeric IDs
	return nil, os.Chown(c.Path, -1, -1)
}
