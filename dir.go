package gorrect

import (
	"context"
	"fmt"
	"log"
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

func (dir Dir) Identity() (Shape, string) {
	return Path, dir.Target
}

func (dir Dir) PreElements() []string {
	return dir.dirs
}

func (dir Dir) Execute(ctx context.Context) error {
	log.Printf("creating dir %s", dir.Target)
	return nil
}
