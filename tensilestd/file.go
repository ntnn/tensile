package tensilestd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ntnn/tensile"
)

var FileRef = tensile.Ref("File")

type File struct {
	Chmod
	Chown

	Path    string
	Content string
}

func (f *File) Validate() error {
	f.Chmod.Path = f.Path
	f.Chown.Path = f.Path
	return nil
}

func (f *File) Provides() ([]tensile.NodeRef, error) {
	return []tensile.NodeRef{FileRef.To(f.Path)}, nil
}

// parentDirs returns a list of all parent directories.
// It does not handle relative paths.
func parentDirs(p string) []string {
	ret := []string{}
	previous := ""
	for {
		previous = p
		p = filepath.Dir(p)
		if previous == p {
			return ret
		}
		ret = append(ret, p)
	}
}

func (f *File) DependsOn() ([]tensile.NodeRef, error) {
	return DirRef.ToMany(parentDirs(f.Path)), nil
}

func (f *File) NeedsExecution() (bool, error) {
	chmodNeeded, err := f.Chmod.NeedsExecution()
	if err != nil {
		return false, fmt.Errorf("error checking chmod: %w", err)
	}
	if chmodNeeded {
		return true, nil
	}

	chownNeeded, err := f.Chown.NeedsExecution()
	if err != nil {
		return false, fmt.Errorf("error checking chown: %w", err)
	}
	if chownNeeded {
		return true, nil
	}

	info, err := os.Stat(f.Path)
	if err != nil {
		return false, fmt.Errorf("error stating file: %w", err)
	}

	// TODO compare content
	return info.Size() != int64(len(f.Content)), nil
}

func (f *File) Execute() error {
	fd, err := os.Create(f.Path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer fd.Close()

	if _, err := fd.WriteString(f.Content); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	if err := f.Chmod.Execute(); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}

	if err := f.Chown.Execute(); err != nil {
		return fmt.Errorf("error setting file ownership: %w", err)
	}

	return nil
}
