package std

import (
	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Noop)(nil)
var _ tensile.Executor = (*Noop)(nil)

// NoopIdentity returns the identity of the named noop node.
func NoopIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("noop", "name", name)
}

// Noop does nothing.
//
// When Noop is added as a notifier it always notifies.
type Noop struct {
	Name string
}

// Identity implements [tensile.Identifier].
func (n *Noop) Identity() tensile.Identity {
	return NoopIdentity(n.Name)
}

// NeedsExecution implements [tensile.Executor].
// A Noop always executes so notifications propagate.
func (n *Noop) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return true, nil, nil
}

// Execute implements [tensile.Executor].
func (n *Noop) Execute(_ tensile.Wire) (tensile.Diff, error) {
	return nil, nil //nolint:nilnil // nil Diff is valid
}
