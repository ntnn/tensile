package tensilestd

import "github.com/ntnn/tensile"

var DirRef = tensile.Ref("Dir")

type Dir struct {
	Path string
	Chmod
	Chown
}

func (d *Dir) NodeRef() tensile.NodeRef {
	return DirRef.To(d.Path)
}

func (d *Dir) Validate() error {
	d.Chmod.Path = d.Path
	d.Chown.Path = d.Path
	return nil
}

func (d *Dir) Provides() ([]string, error) {
	return nil, nil
}

func (d *Dir) DependsOn() ([]string, error) {
	return parentDirs(d.Path), nil
}

func (d *Dir) NeedsExecution() (bool, error) {
	return false, nil
}

func (d *Dir) Execute() error {
	return nil
}
