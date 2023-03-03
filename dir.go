package gorrect

import (
	"context"
	"fmt"
	"log"
)

type Dir struct {
	Base

	Target string

	dirs []string
}

func (dir *Dir) Validate() error {
	dir.Base.Shape = Path

	if dir.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	if dir.Base.Name == "" {
		dir.Base.Name = dir.Target
	}

	dirs, err := walkDirs(dir.Target)
	if err != nil {
		return err
	}
	dir.dirs = dirs

	return nil
}

func (dir Dir) PreElements() []string {
	return dir.dirs
}

func (dir Dir) Execute(ctx context.Context) error {
	log.Printf("creating dir %s", dir.Target)
	return nil
}
