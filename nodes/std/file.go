package std

import (
	"os"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
)

// FileIdentity returns the identity of the node managing the file at path.
func FileIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("file", "path", path)
}

// FileQueueIdentity returns the identity of the queue managing the file at path.
func FileQueueIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("queue", "file", path)
}

// File describes a file with ownership and permissions.
type File struct {
	Path     string
	FileMode os.FileMode
	Owner    string
	Group    string
	Content  string
}

// NewFile returns a named queue managing a file.
func NewFile(file File) *queue.NamedQueue {
	q := queue.NewNamed(FileQueueIdentity(file.Path))
	content := &FileContent{Path: file.Path, Content: file.Content}
	chmod := Chmod{Path: file.Path, FileMode: file.FileMode}
	chown := Chown{Path: file.Path, Owner: file.Owner, Group: file.Group}
	q.Add(content, chmod, chown)
	// [FileContent] creates the file if it does not exist, chmod/chown do not
	q.RequiredBy(content, chmod, chown)
	return q
}
