package tensilestd

import (
	"context"
	"fmt"
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Provider = (*FileContent)(nil)
var _ tensile.Depender = (*FileContent)(nil)
var _ tensile.Executor = (*FileContent)(nil)

const FileContentRef = tensile.Ref("FileContent")

type FileContent struct {
	Path    string
	Content string
}

func (f *FileContent) Provides() ([]tensile.NodeRef, error) {
	return []tensile.NodeRef{FileRef.To(f.Path)}, nil
}

func (f *FileContent) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(f.Path)), nil
}

func (f *FileContent) NeedsExecution(ctx context.Context) (bool, error) {
	// TODO
	return true, nil
}

func (f *FileContent) Execute(ctx context.Context) error {
	fd, err := os.Create(f.Path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer fd.Close()

	if _, err := fd.WriteString(f.Content); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	return nil
}
