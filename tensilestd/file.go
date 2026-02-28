package tensilestd

import (
	"os"
	"path/filepath"

	"github.com/ntnn/tensile"
)

var _ tensile.ValidatorCtx = (*File)(nil)
var _ tensile.Provider = (*File)(nil)
var _ tensile.Depender = (*File)(nil)
var _ tensile.ExecutorCtx = (*File)(nil)

// FileRef is the reference type for files.
const FileRef = tensile.Ref("File")

// File manages file creation with ownership and permissions.
type File struct {
	*Aggregate

	Path     string
	FileMode os.FileMode
	Owner    string
	Group    string
	Content  string
}

// Validate implements [tensile.Validator].
func (f *File) Validate(ctx tensile.Context) error {
	agg, err := NewAggregate(
		Chmod{Path: f.Path, FileMode: f.FileMode},
		Chown{Path: f.Path, Owner: f.Owner, Group: f.Group},
		FileContent{Path: f.Path, Content: f.Content},
	)
	if err != nil {
		return err
	}
	f.Aggregate = agg
	return f.Aggregate.Validate(ctx)
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
