package tensilestd

import (
	"context"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Depender = (*Chmod)(nil)
var _ tensile.Executor = (*Chmod)(nil)

const ChmodRef = tensile.Ref("Chmod")

type Chmod struct {
	Path     string
	FileMode os.FileMode
}

func (c Chmod) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(c.Path)), nil
}

func (c Chmod) NeedsExecution(_ context.Context) (bool, error) {
	info, err := os.Stat(c.Path)
	if err != nil {
		return false, err
	}
	return info.Mode() != c.FileMode, nil
}

func (c Chmod) Execute(_ context.Context) error {
	return os.Chmod(c.Path, c.FileMode)
}
