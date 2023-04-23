package nodes

import (
	"fmt"
	"log"

	"github.com/ntnn/tensile"
)

var _ tensile.Node = (*Dir)(nil)

type Dir struct {
	Target string

	dirs []string
}

func (dir *Dir) Validate() error {
	if dir.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	dirs, err := walkDirs(dir.Target)
	if err != nil {
		return err
	}
	dir.dirs = dirs

	return nil
}

func (dir Dir) Shape() tensile.Shape {
	return tensile.Path
}

func (dir Dir) Identifier() string {
	return dir.Target
}

func (dir Dir) PreElements() []string {
	return dir.dirs
}

func (dir Dir) Execute(ctx tensile.Context) (any, error) {
	log.Printf("creating dir %s", dir.Target)
	return nil, nil
}
