package tensilestd

import (
	"context"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Depender = (*Chown)(nil)
var _ tensile.Executor = (*Chown)(nil)

// ChownRef is the reference type for chown operations.
const ChownRef = tensile.Ref("Chown")

// Chown ensures a file has the specified owner and group.
type Chown struct {
	Path  string
	Owner string
	Group string
}

// DependsOn implements [tensile.Depender].
func (c Chown) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(c.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chown) NeedsExecution(_ context.Context) (bool, error) {
	// TODO resolve owner and group names to numeric IDs
	// TODO check if the current owner and group match the desired ones
	return true, nil
}

// Execute implements [tensile.Executor].
func (c Chown) Execute(_ context.Context) error {
	// TODO resolve owner and group names to numeric IDs
	return os.Chown(c.Path, -1, -1)
}
