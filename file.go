package gorrect

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
)

type File struct {
	Target string

	dirs []string
}

func (file *File) Validate() error {
	if file.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	dirs, err := walkDirs(file.Target)
	if err != nil {
		return err
	}
	file.dirs = dirs

	return nil
}

func walkDirs(target string) ([]string, error) {
	s, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}

	var last string
	ret := []string{}

	for s = filepath.Dir(s); s != last; s = filepath.Dir(s) {
		ret = append(ret, fmt.Sprintf("Path[%s]", s))
		last = s
	}

	return ret, nil
}

func (file File) Identity() (Identity, string) {
	return Path, file.Target
}

func (file File) PreElements() []string {
	return file.dirs
}

func (file File) Execute(ctx context.Context) error {
	log.Printf("creating file %s", file.Target)
	return nil
}
