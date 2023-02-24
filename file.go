package gorrect

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
)

type File struct {
	target string
	dirs   []string
}

func NewFile(target string) (*File, error) {
	dirs, err := walkDirs(target)
	if err != nil {
		return nil, err
	}

	return &File{
		target: target,
		dirs:   dirs,
	}, nil
}

func walkDirs(target string) ([]string, error) {
	s, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}

	var last string
	ret := []string{}

	for s = filepath.Dir(s); s != last; s = filepath.Dir(s) {
		ret = append(ret, fmt.Sprintf("Dir[%s]", s))
		last = s
	}

	return ret, nil
}

func (file File) Identity() string {
	return fmt.Sprintf("File[%s]", file.target)
}

func (file File) PreElements() []string {
	return file.dirs
}

func (file File) Execute(ctx context.Context) error {
	log.Printf("creating file %s", file.target)
	return nil
}
