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

// FileContentRef is the reference type for file content operations.
const FileContentRef = tensile.Ref("FileContent")

// FileContent ensures a file is created with the specified content.
type FileContent struct {
	Path    string
	Content string
}

// Provides implements [tensile.Provider].
func (f *FileContent) Provides() ([]tensile.NodeRef, error) {
	return []tensile.NodeRef{FileRef.To(f.Path)}, nil
}

// DependsOn implements [tensile.Depender].
func (f *FileContent) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(f.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (f *FileContent) NeedsExecution(_ context.Context) (bool, error) {
	// TODO
	return true, nil
}

// Execute implements [tensile.Executor].
func (f *FileContent) Execute(_ context.Context) error {
	fd, err := os.Create(f.Path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer fd.Close() //nolint:errcheck

	if _, err := fd.WriteString(f.Content); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	return nil
}
