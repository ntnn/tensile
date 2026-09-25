package std

import (
	"errors"
	"fmt"
	"os"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
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

var _ tensile.Identifier = (*File)(nil)
var _ tensile.Validator = (*File)(nil)
var _ tensile.Depender = (*File)(nil)
var _ tensile.Executor = (*File)(nil)
var _ queue.NamedQueuer = (*File)(nil)

// FileState is the desired existence of a file.
type FileState string

// Desired existence states for [File].
const (
	FilePresent FileState = "present"
	FileAbsent  FileState = "absent"
)

// File ensures a file is present or absent.
type File struct {
	Path string
	// State is the desired existence.
	// Defaults to [FilePresent].
	// [FileAbsent] ignores all fields but Path.
	State FileState

	// Content is the managed file content.
	// Empty means only existence is ensured, existing content is left alone.
	Content string

	// FileMode, Owner and Group are applied by the queue returned by [NewFile], not by the node itself.
	FileMode os.FileMode
	Owner    string
	Group    string
}

// desired returns the desired state, defaulting to [FilePresent].
func (f *File) desired() FileState {
	if f.State == "" {
		return FilePresent
	}
	return f.State
}

// Queue implements [queue.NamedQueuer].
// The queue contains the [File] itself and, when the fields are set, [Chmod], [Chown] and [FileContent] nodes.
func (f *File) Queue() *queue.NamedQueue {
	q := queue.NewNamed(FileQueueIdentity(f.Path))
	// wrapped in a Node so Add does not dissolve the File again
	q.Add(tensile.NewNode(f))
	if f.desired() == FileAbsent {
		return q
	}
	if f.FileMode != 0 {
		q.Add(Chmod{Path: f.Path, FileMode: f.FileMode})
	}
	if f.Owner != "" || f.Group != "" {
		q.Add(Chown{Path: f.Path, Owner: f.Owner, Group: f.Group})
	}
	if f.Content != "" {
		q.Add(&FileContent{Path: f.Path, Content: f.Content})
	}
	return q
}

// Identity implements [tensile.Identifier].
func (f *File) Identity() tensile.Identity {
	return FileIdentity(f.Path)
}

// Validate implements [tensile.Validator].
func (f *File) Validate(_ tensile.Wire) error {
	if f.Path == "" {
		return errors.New("path is required")
	}
	switch f.State {
	case "", FilePresent, FileAbsent:
	default:
		return fmt.Errorf("unknown state %q", f.State)
	}
	return nil
}

// DependsOn implements [tensile.Depender].
func (f *File) DependsOn() ([]tensile.Identity, error) {
	return ParentDirIdentities(f.Path), nil
}

// NeedsExecution implements [tensile.Executor].
func (f *File) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	_, err := os.Stat(f.Path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("checking file: %w", err)
	}

	if f.desired() == FileAbsent {
		if !exists {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{Field: "file", Old: f.Path, New: diff.Absent}), nil
	}
	if exists {
		return false, nil, nil
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{Field: "file", Old: diff.Absent, New: f.Path}), nil
}

// Execute implements [tensile.Executor].
func (f *File) Execute(_ tensile.Wire) (tensile.Diff, error) {
	if f.desired() == FileAbsent {
		if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("removing file: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	}

	// perms apply only on create, an existing file keeps content and perms
	fd, err := os.OpenFile(f.Path, os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec,mnd
	if err != nil {
		return nil, fmt.Errorf("creating file: %w", err)
	}
	if err := fd.Close(); err != nil {
		return nil, fmt.Errorf("closing file: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}
