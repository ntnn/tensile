package std

import (
	"os"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*Chmod)(nil)
var _ tensile.Depender = (*Chmod)(nil)
var _ tensile.Executor = (*Chmod)(nil)

// Chmod ensures a file has the specified permissions.
// FileMode is interpreted as unix permission bits.
type Chmod struct {
	Path     string
	FileMode os.FileMode
}

// Identity implements [tensile.Identifier].
func (c Chmod) Identity() tensile.Identity {
	return tensile.AsIdentity("chmod", "path", c.Path)
}

// DependsOn implements [tensile.Depender].
func (c Chmod) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(c.Path),
		FileIdentity(c.Path),
	), nil
}

func (c Chmod) needsExecution(_ tensile.Wire) (bool, *diff.FieldChange, error) {
	needs, current, desired, err := chmodNeedsExecution(c.Path, c.FileMode)
	if err != nil || !needs {
		return needs, nil, err
	}
	return true, &diff.FieldChange{Field: "mode", Old: current, New: desired}, nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chmod) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	needs, change, err := c.needsExecution(wire)
	return needs, diff.NewFieldChanges(change), err
}

// Execute implements [tensile.Executor].
func (c Chmod) Execute(_ tensile.Wire) (tensile.Diff, error) {
	return nil, chmodApply(c.Path, c.FileMode)
}
