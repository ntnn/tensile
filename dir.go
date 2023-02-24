package gorrect

import (
	"context"
	"fmt"
	"log"
)

type Dir struct {
	target string
	dirs   []string
}

func NewDir(target string) (*Dir, error) {
	dirs, err := walkDirs(target)
	if err != nil {
		return nil, err
	}

	return &Dir{
		target: target,
		dirs:   dirs,
	}, nil
}

func (dir Dir) Identity() string {
	return fmt.Sprintf("Dir[%s]", dir.target)
}

func (dir Dir) PreElements() []string {
	return dir.dirs
}

func (dir Dir) Execute(ctx context.Context) error {
	log.Printf("creating dir %s", dir.target)
	return nil
}
