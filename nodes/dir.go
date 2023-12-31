package nodes

import (
	"fmt"
	"log"

	"github.com/ntnn/tensile"
)

var _ tensile.Node = (*Dir)(nil)

type Dir struct {
	Target string

	parentDirs *ParentDirs
}

func (dir *Dir) Validate() error {
	if dir.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	if dir.parentDirs == nil {
		dir.parentDirs = NewParentDirs(dir.Target)
	}

	return nil
}

func (dir Dir) Shape() tensile.Shape {
	return tensile.Path
}

func (dir Dir) Identifier() string {
	return dir.Target
}

var _ tensile.AfterNoder = (*Dir)(nil)

func (dir Dir) AfterNodes() []string {
	return dir.parentDirs.AfterNodes()
}

func (dir Dir) Execute(ctx tensile.Context) (any, error) {
	log.Printf("creating dir %s", dir.Target)
	return nil, nil
}
