package std

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*FileContent)(nil)
var _ tensile.Conflictor = (*FileContent)(nil)
var _ tensile.Depender = (*FileContent)(nil)
var _ tensile.Executor = (*FileContent)(nil)
var _ tensile.Reporter = (*FileContent)(nil)

// FileContentOutput is the output reported by [FileContent].
type FileContentOutput struct {
	Path string
	// SHA256 is the hex-encoded checksum of the content.
	SHA256 string
}

// FileContent ensures a file is created with the specified content.
type FileContent struct {
	Path    string
	Content string
}

// Identity implements [tensile.Identifier].
func (f *FileContent) Identity() tensile.Identity {
	return tensile.AsIdentity("fileContent", "path", f.Path)
}

// Conflicts implements [tensile.Conflictor].
func (f *FileContent) Conflicts() ([]tensile.Identity, error) {
	return []tensile.Identity{FileIdentity(f.Path)}, nil
}

// DependsOn implements [tensile.Depender].
func (f *FileContent) DependsOn() ([]tensile.Identity, error) {
	return ParentDirIdentities(f.Path), nil
}

// NeedsExecution implements [tensile.Executor].
func (f *FileContent) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	current, err := os.ReadFile(f.Path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("error reading file: %w", err)
	}

	if exists && string(current) == f.Content {
		return false, nil, nil
	}
	return true, diff.Unified{Path: f.Path, Old: string(current), New: f.Content}, nil
}

// Execute implements [tensile.Executor].
func (f *FileContent) Execute(_ tensile.Wire) (tensile.Diff, error) {
	fd, err := os.Create(f.Path)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}
	defer fd.Close() //nolint:errcheck

	if _, err := fd.WriteString(f.Content); err != nil {
		return nil, fmt.Errorf("error writing to file: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// Report implements [tensile.Reporter].
func (f *FileContent) Report(_ tensile.Wire) (any, error) {
	return FileContentOutput{
		Path:   f.Path,
		SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(f.Content))),
	}, nil
}
