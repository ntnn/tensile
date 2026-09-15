package std

import (
	"os"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*File)(nil)
var _ tensile.Validator = (*File)(nil)
var _ tensile.Provider = (*File)(nil)
var _ tensile.Depender = (*File)(nil)
var _ tensile.Executor = (*File)(nil)

// FileIdentity returns the identity of the node managing the file at path.
func FileIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("file", "path", path)
}

// File manages file creation with ownership and permissions.
type File struct {
	*Aggregate

	Path     string
	FileMode os.FileMode
	Owner    string
	Group    string
	Content  string
}

// Identity implements [tensile.Identifier].
func (f *File) Identity() tensile.Identity {
	return FileIdentity(f.Path)
}

// Validate implements [tensile.Validator].
func (f *File) Validate(s tensile.Wire) error {
	f.Aggregate = NewAggregate(
		Chmod{Path: f.Path, FileMode: f.FileMode},
		Chown{Path: f.Path, Owner: f.Owner, Group: f.Group},
		&FileContent{Path: f.Path, Content: f.Content},
	)
	return f.Aggregate.Validate(s)
}

// parentDirs returns a list of all parent directories.
// It does not handle relative paths.
func parentDirs(p string) []string {
	ret := []string{}
	var previous string
	for {
		previous = p
		p = filepath.Dir(p)
		if previous == p {
			return ret
		}
		ret = append(ret, p)
	}
}
