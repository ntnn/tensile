package tensilestd

import (
	"context"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Depender = (*Chown)(nil)
var _ tensile.Executor = (*Chown)(nil)

const ChownRef = tensile.Ref("Chown")

type Chown struct {
	Path  string
	Owner string
	Group string
}

func (c Chown) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(c.Path)), nil
}

func (c Chown) NeedsExecution(_ context.Context) (bool, error) {
	// TODO resolve owner and group names to numeric IDs
	// TODO check if the current owner and group match the desired ones
	return true, nil
}

func (c Chown) Execute(_ context.Context) error {
	// TODO resolve owner and group names to numeric IDs
	return os.Chown(c.Path, -1, -1)
}
