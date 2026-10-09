package std

import (
	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*Chown)(nil)
var _ tensile.Depender = (*Chown)(nil)
var _ tensile.Executor = (*Chown)(nil)

// Chown ensures a file has the specified owner and group.
type Chown struct {
	Path  string
	Owner string
	Group string
}

// Identity implements [tensile.Identifier].
func (c Chown) Identity() tensile.Identity {
	return tensile.AsIdentity("chown", "path", c.Path)
}

// DependsOn implements [tensile.Depender].
func (c Chown) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(c.Path),
		FileIdentity(c.Path),
		UserIdentity(c.Owner),
		GroupIdentity(c.Group),
	), nil
}

// NeedsExecution implements [tensile.Executor].
func (c Chown) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	needs, ownerChange, groupChange, err := chownNeedsExecution(c.Path, c.Owner, c.Group)
	if err != nil || !needs {
		return false, nil, err
	}
	return true, diff.NewFieldChanges(ownerChange, groupChange), nil
}

// Execute implements [tensile.Executor].
func (c Chown) Execute(_ tensile.Wire) (tensile.Diff, error) {
	return nil, chownApply(c.Path, c.Owner, c.Group)
}
