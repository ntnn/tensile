package std

import (
	"os"

	"github.com/ntnn/tensile"
)

// FileIdentity returns the identity of the node managing the file at path.
func FileIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("file", "path", path)
}

// File describes a file with ownership and permissions.
type File struct {
	Path     string
	FileMode os.FileMode
	Owner    string
	Group    string
	Content  string
}

// NewFile returns a [tensile.Group] managing a file.
func NewFile(file File) (*tensile.Group, error) {
	content := &FileContent{Path: file.Path, Content: file.Content}
	chmod := Chmod{Path: file.Path, FileMode: file.FileMode}
	chown := Chown{Path: file.Path, Owner: file.Owner, Group: file.Group}

	group := tensile.NewGroup("file " + file.Path)
	if err := group.Add(content, chmod, chown); err != nil {
		return nil, err
	}
	return group, nil
}
