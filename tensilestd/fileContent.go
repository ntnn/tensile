package tensilestd

import (
	"crypto/sha256"
	"fmt"
	"io"
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
	return tensile.ToMany(DirRef, parentDirs(f.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (f *FileContent) NeedsExecution(_ tensile.Wire) (bool, error) {
	fd, err := os.Open(f.Path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("error opening file: %w", err)
	}
	defer fd.Close() //nolint:errcheck

	hash := sha256.New()
	if _, err := io.Copy(hash, fd); err != nil {
		return false, fmt.Errorf("error hashing file: %w", err)
	}

	return [sha256.Size]byte(hash.Sum(nil)) != sha256.Sum256([]byte(f.Content)), nil
}

// Execute implements [tensile.Executor].
func (f *FileContent) Execute(_ tensile.Wire) error {
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
