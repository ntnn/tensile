package nodes

import (
	"fmt"
	"log"

	"github.com/ntnn/tensile"
)

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

func (dir Dir) Identity() (tensile.Shape, string) {
	return tensile.Path, dir.Target
}

func (dir Dir) PreElements() []string {
	return dir.dirs
}

func (dir Dir) Execute(ctx tensile.Context) error {
	log.Printf("creating dir %s", dir.Target)
	return nil
}
