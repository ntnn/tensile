package tensilestd

import (
	"context"

	"github.com/ntnn/tensile"
)

var _ tensile.Validator = (*Dir)(nil)
var _ tensile.Provider = (*Dir)(nil)
var _ tensile.Depender = (*Dir)(nil)
var _ tensile.Executor = (*Dir)(nil)

const DirRef = tensile.Ref("Dir")

type Dir struct {
	Path string
	Chmod
	Chown
}

func (d *Dir) Validate(_ context.Context) error {
	d.Chmod.Path = d.Path
	d.Chown.Path = d.Path
	return nil
}

func (d *Dir) Provides() ([]tensile.NodeRef, error) {
	return []tensile.NodeRef{DirRef.To(d.Path)}, nil
}

func (d *Dir) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(d.Path)), nil
}

func (d *Dir) NeedsExecution(_ context.Context) (bool, error) {
	return false, nil
}

func (d *Dir) Execute(_ context.Context) error {
	return nil
}
